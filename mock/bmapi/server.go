// Package bmapi is an in-process mock of the first-class bare-metal API (dev.envmgmt.io) for
// tests. It reproduces what was observed on live controllers: apply is an upsert by name;
// provisioning is asynchronous (in_progress, then running or failed); failures read "no capacity
// available right now" or "due to an internal error"; actions (start, stop, reboot, Reinstall OS)
// run asynchronously; DELETE starts a teardown (deleting, then gone); and the compute instance
// behind each instance, which holds the workflow error, is only visible with
// include_first_class=true.
//
// Phase changes follow a timeline measured from the write, so tests need no goroutines and run in
// milliseconds (Options.Step).
package bmapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

const (
	devPrefix  = "/apis/dev.envmgmt.io/v1/projects/"
	paasPrefix = "/apis/paas.envmgmt.io/v1/projects/"
)

// Options configures a Server.
type Options struct {
	// Profiles are the bare-metal SKUs (BmProfile) known to the mock.
	Profiles []apiv1.BmProfile
	// SSHKeys are the SSH keys that exist in every workspace and project. When non-nil, an instance
	// naming another key is rejected with 400.
	SSHKeys []string
	// Capacity is how many instances of one SKU can run at once. 0 means unlimited; an instance
	// beyond it ends failed with "no capacity available right now", as on the live platform.
	Capacity int
	// Step is the delay between phase changes. Defaults to 20ms.
	Step time.Duration
	// MirrorLinger keeps the compute instance behind a deleted instance visible ("destroying",
	// with include_first_class=true) for this long after the record is gone.
	MirrorLinger time.Duration
	// NotFoundStatus is the HTTP status for a missing instance. Defaults to 404.
	NotFoundStatus int
	// DropMetadataText makes the mock store no display name and no description, as the live platform did for a
	// description on a bare-metal machine.
	DropMetadataText bool
	// UnsupportedActions are action names the instances refuse with 400, as the live platform does
	// for "reboot" on BCM instances: `action "reboot" is not supported for this instance`.
	UnsupportedActions []string
}

// Server is the mock API. Create one with New and close it with Close.
type Server struct {
	*httptest.Server
	opts Options

	// FailProvision makes provisioning end failed with the returned reason (empty means success).
	FailProvision func(name string) string
	// FailDelete makes a delete end in destroyfailed with the returned reason; the next DELETE retries.
	FailDelete func(name string) string
	// FailureCause returns the workflow error behind a failed instance; the mock serves it on the
	// compute instance of the same name, with include_first_class=true.
	FailureCause func(name string) string
	// ParkDelete parks a delete for the operator when it returns true: the instance ends
	// delete_pending, and every further DELETE is refused with 412, as on the live platform.
	ParkDelete func(name string) bool
	// ReasonDetail returns the workflow error the mock reports as status.reasonDetail on a failed
	// instance itself, as the live API does.
	ReasonDetail func(name string) string

	mu        sync.Mutex
	instances map[string]*instance // project/workspace/name
	mirrors   map[string]time.Time
	requests  []string
	orgs      map[string]bool
	seq       int
}

type phaseAt struct {
	at     time.Time
	status string // lowercase status, or "gone"
	reason string
	action string
}

type instance struct {
	obj      convert.BareMetalObject
	password string // write-only: never returned
	timeline []phaseAt
	// capacityFailed marks an instance that could not get a machine.
	capacityFailed bool
}

// New starts a mock server.
func New(opts Options) *Server {
	if opts.Step <= 0 {
		opts.Step = 20 * time.Millisecond
	}
	if opts.NotFoundStatus == 0 {
		opts.NotFoundStatus = http.StatusNotFound
	}
	s := &Server{opts: opts, instances: map[string]*instance{}, mirrors: map[string]time.Time{}, orgs: map[string]bool{}}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

// Requests returns "METHOD path?query" for every request received, in order.
func (s *Server) Requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

// Organizations returns the distinct X-Organization-ID header values the server received.
func (s *Server) Organizations() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.orgs))
	for o := range s.orgs {
		out = append(out, o)
	}
	return out
}

// BareMetal returns a snapshot of the stored instance at its current phase, or nil.
func (s *Server) BareMetal(project, workspace, name string) *apiv1.BareMetal {
	s.mu.Lock()
	defer s.mu.Unlock()
	in := s.instances[key(project, workspace, name)]
	if in == nil {
		return nil
	}
	w, _ := s.view(in)
	return convert.FromBareMetal(w, project, workspace)
}

// RawSpec returns the spec exactly as stored, including the write-only password, for tests.
func (s *Server) RawSpec(project, workspace, name string) *apiv1.BareMetalSpec {
	s.mu.Lock()
	defer s.mu.Unlock()
	in := s.instances[key(project, workspace, name)]
	if in == nil {
		return nil
	}
	sp := in.obj.Spec
	sp.UserPassword = in.password
	return &sp
}

// AdminDelete removes an instance record at once, as an administrator can for a parked delete.
func (s *Server) AdminDelete(project, workspace, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.instances, key(project, workspace, name))
}

// Exists reports whether the instance record is stored (it may be deleting).
func (s *Server) Exists(project, workspace, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.instances[key(project, workspace, name)]
	return ok
}

// MirrorExists reports whether the compute instance behind a deleted instance is still around.
func (s *Server) MirrorExists(project, workspace, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Now().Before(s.mirrors[key(project, workspace, name)])
}

func key(project, workspace, name string) string { return project + "/" + workspace + "/" + name }

func (s *Server) profile(name string) *apiv1.BmProfile {
	for i := range s.opts.Profiles {
		if s.opts.Profiles[i].Metadata.Name == name {
			return &s.opts.Profiles[i]
		}
	}
	return nil
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	line := r.Method + " " + r.URL.Path
	if r.URL.RawQuery != "" {
		line += "?" + r.URL.RawQuery
	}
	s.requests = append(s.requests, line)
	if org := r.Header.Get("X-Organization-ID"); org != "" {
		s.orgs[org] = true
	}
	switch {
	case strings.HasPrefix(r.URL.Path, devPrefix):
		s.dev(w, r)
	case strings.HasPrefix(r.URL.Path, paasPrefix):
		s.computeInstances(w, r)
	default:
		writeErr(w, http.StatusNotFound, "404 page not found")
	}
}

// route is a parsed dev API path.
type route struct {
	project, workspace string
	kind               string // baremetals | bmprofiles | sshkeys
	name, sub, verb    string
}

func parseDev(path string) (route, bool) {
	parts := strings.Split(strings.TrimPrefix(path, devPrefix), "/")
	r := route{project: parts[0]}
	rest := parts[1:]
	if len(rest) >= 2 && rest[0] == "workspaces" {
		r.workspace = rest[1]
		rest = rest[2:]
	}
	if len(rest) == 0 {
		return route{}, false
	}
	r.kind = rest[0]
	switch r.kind {
	case "baremetals", "bmprofiles", "sshkeys":
	default:
		return route{}, false
	}
	if len(rest) > 1 {
		r.name = rest[1]
	}
	if len(rest) > 2 {
		r.sub = rest[2]
	}
	if len(rest) > 3 {
		r.verb = strings.Join(rest[3:], "/") // the server has already percent-decoded r.URL.Path
	}
	return r, true
}

func (s *Server) dev(w http.ResponseWriter, r *http.Request) {
	rt, ok := parseDev(r.URL.Path)
	if !ok {
		writeErr(w, http.StatusNotFound, "404 page not found")
		return
	}
	switch rt.kind {
	case "bmprofiles":
		s.profiles(w, r, rt)
	case "sshkeys":
		s.sshKeys(w, r, rt)
	default:
		s.baremetals(w, r, rt)
	}
}

func (s *Server) profiles(w http.ResponseWriter, r *http.Request, rt route) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	wire := func(p *apiv1.BmProfile) convert.BmProfileObject {
		st := p.Status
		if st.Status == "" {
			st.Status = "success"
		}
		return convert.BmProfileObject{APIVersion: convert.DevAPIVersion, Kind: convert.DevBmProfileKind,
			Metadata: convert.DevMetadata{Name: p.Metadata.Name, Project: rt.project, Labels: p.Metadata.Labels}, Spec: p.Spec, Status: &st}
	}
	if rt.name != "" {
		p := s.profile(rt.name)
		if p == nil {
			writeErr(w, http.StatusNotFound, "bm profile not found")
			return
		}
		out := wire(p)
		writeJSON(w, http.StatusOK, out)
		return
	}
	list := convert.BmProfileObjectList{APIVersion: convert.DevAPIVersion, Kind: convert.DevBmProfileListKind, Items: []convert.BmProfileObject{}}
	for i := range s.opts.Profiles {
		list.Items = append(list.Items, wire(&s.opts.Profiles[i]))
	}
	list.Metadata.Count = int64(len(list.Items))
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) sshKeys(w http.ResponseWriter, r *http.Request, rt route) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	key := func(name string) map[string]any {
		return map[string]any{"apiVersion": convert.DevAPIVersion, "kind": "SshKey", "metadata": map[string]any{"name": name, "project": rt.project, "workspace": rt.workspace}, "spec": map[string]any{}}
	}
	if rt.name != "" {
		if !contains(s.opts.SSHKeys, rt.name) {
			writeErr(w, s.opts.NotFoundStatus, "ent: ssh key not found")
			return
		}
		writeJSON(w, http.StatusOK, key(rt.name))
		return
	}
	items := []map[string]any{}
	for _, n := range s.opts.SSHKeys {
		items = append(items, key(n))
	}
	writeJSON(w, http.StatusOK, map[string]any{"apiVersion": convert.DevAPIVersion, "kind": "SshKeyList", "metadata": map[string]any{"count": len(items)}, "items": items})
}

func (s *Server) baremetals(w http.ResponseWriter, r *http.Request, rt route) {
	switch {
	case r.Method == http.MethodGet && rt.name == "":
		s.list(w, rt)
	case r.Method == http.MethodGet && rt.sub == "":
		in := s.instances[key(rt.project, rt.workspace, rt.name)]
		if in == nil {
			writeErr(w, s.opts.NotFoundStatus, "ent: bare metal not found")
			return
		}
		view, status := s.view(in)
		if status == "gone" {
			s.mirrors[key(rt.project, rt.workspace, rt.name)] = time.Now().Add(s.opts.MirrorLinger)
			delete(s.instances, key(rt.project, rt.workspace, rt.name))
			writeErr(w, s.opts.NotFoundStatus, "ent: bare metal not found")
			return
		}
		writeJSON(w, http.StatusOK, view)
	case r.Method == http.MethodPost && rt.name == "":
		s.apply(w, r, rt)
	case r.Method == http.MethodPost && rt.sub == "action":
		s.action(w, r, rt)
	case r.Method == http.MethodDelete && rt.name != "":
		s.remove(w, rt)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) list(w http.ResponseWriter, rt route) {
	list := convert.BareMetalObjectList{APIVersion: convert.DevAPIVersion, Kind: convert.DevBareMetalListKind, Items: []convert.BareMetalObject{}}
	for k, in := range s.instances {
		ks := strings.SplitN(k, "/", 3)
		if ks[0] != rt.project || (rt.workspace != "" && ks[1] != rt.workspace) {
			continue
		}
		if v, status := s.view(in); status != "gone" {
			list.Items = append(list.Items, *v)
		}
	}
	list.Metadata.Count = int64(len(list.Items))
	writeJSON(w, http.StatusOK, list)
}

// apply upserts an instance, validating it against its SKU like the backend does.
func (s *Server) apply(w http.ResponseWriter, r *http.Request, rt route) {
	var in convert.BareMetalObject
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	name := in.Metadata.Name
	if name == "" {
		writeErr(w, http.StatusBadRequest, "metadata.name is required")
		return
	}
	k := key(rt.project, rt.workspace, name)
	existing := s.instances[k]
	sp := in.Spec
	if existing != nil && sp.Type != "" && sp.Type != existing.obj.Spec.Type {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("unable to apply bare metal: spec.type cannot be changed (is %q)", existing.obj.Spec.Type))
		return
	}
	if existing != nil && sp.Type == "" {
		sp.Type = existing.obj.Spec.Type
	}
	if sp.Type == "" {
		writeErr(w, http.StatusBadRequest, "unable to apply bare metal: spec.type is required")
		return
	}
	prof := s.profile(sp.Type)
	if prof == nil {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("unable to apply bare metal: bm profile %q not found", sp.Type))
		return
	}
	if msg := s.validate(sp, prof, existing == nil); msg != "" {
		writeErr(w, http.StatusBadRequest, "unable to apply bare metal: "+msg)
		return
	}

	password := sp.UserPassword
	if existing != nil && password == "" {
		password = existing.password // omitted on update: keep the existing password
	}
	next := convert.BareMetalObject{APIVersion: convert.DevAPIVersion, Kind: convert.DevBareMetalKind}
	next.Metadata = convert.DevMetadata{Name: name, Project: rt.project, Workspace: rt.workspace, DisplayName: in.Metadata.DisplayName, Description: in.Metadata.Description, Labels: in.Metadata.Labels, Annotations: in.Metadata.Annotations}
	if s.opts.DropMetadataText {
		next.Metadata.DisplayName, next.Metadata.Description = "", ""
	}
	next.Spec = derive(sp, prof)
	next.Spec.UserPassword = ""
	now := time.Now()

	if existing != nil {
		// An in-place update: the machine keeps running, only the stored spec changes.
		next.Spec.ComputeInstanceID = existing.obj.Spec.ComputeInstanceID
		existing.obj.Metadata, existing.obj.Spec, existing.password = next.Metadata, next.Spec, password
		view, _ := s.view(existing)
		writeJSON(w, http.StatusOK, view)
		return
	}
	s.seq++
	next.Spec.ComputeInstanceID = fmt.Sprintf("01a00000-0000-7000-8000-%012d", s.seq)
	st := &apiv1.BareMetalStatus{}
	next.Status = st
	inst := &instance{obj: next, password: password}
	tl := []phaseAt{{now, apiv1.BareMetalPhaseInProgress, "Provisioning instance " + name + ".", "deploy"}}
	switch {
	case s.opts.Capacity > 0 && s.running(rt.project, sp.Type) >= s.opts.Capacity:
		inst.capacityFailed = true
		tl = append(tl, phaseAt{now.Add(s.opts.Step), apiv1.BareMetalPhaseFailed, "Couldn't provision instance " + name + " because there's no capacity available right now. Please contact your administrator.", "deploy"})
	case s.FailProvision != nil && s.FailProvision(name) != "":
		tl = append(tl, phaseAt{now.Add(s.opts.Step), apiv1.BareMetalPhaseFailed, "Couldn't provision instance " + name + " due to an internal error. Please contact your administrator.", "deploy"})
	default:
		tl = append(tl, phaseAt{now.Add(s.opts.Step), apiv1.BareMetalPhaseRunning, "Instance " + name + " is running.", "deploy"})
	}
	inst.timeline = tl
	s.instances[k] = inst
	view, _ := s.view(inst)
	writeJSON(w, http.StatusOK, view)
}

// running counts the instances of a SKU that hold a machine (not failed, not gone).
func (s *Server) running(project, sku string) int {
	n := 0
	for k, in := range s.instances {
		if !strings.HasPrefix(k, project+"/") || in.obj.Spec.Type != sku {
			continue
		}
		if _, st := s.view(in); st != apiv1.BareMetalPhaseFailed && st != "gone" {
			n++
		}
	}
	return n
}

func (s *Server) action(w http.ResponseWriter, r *http.Request, rt route) {
	in := s.instances[key(rt.project, rt.workspace, rt.name)]
	if in == nil {
		writeErr(w, s.opts.NotFoundStatus, "ent: bare metal not found")
		return
	}
	_, status := s.view(in)
	verb := strings.ToLower(strings.ReplaceAll(rt.verb, " ", ""))
	for _, u := range s.opts.UnsupportedActions {
		if strings.EqualFold(strings.ReplaceAll(u, " ", ""), verb) {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("action %q is not supported for this instance; supported actions: start, stop, Power cycle, Power reset, Reinstall OS, Replace Instance", rt.verb))
			return
		}
	}
	var payload map[string]any
	_ = json.NewDecoder(r.Body).Decode(&payload)
	var wantStatus string
	switch verb {
	case "start":
		if status != apiv1.BareMetalPhaseStopped {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("cannot start an instance in state %q", status))
			return
		}
		wantStatus = apiv1.BareMetalPhaseRunning
	case "stop":
		if status != apiv1.BareMetalPhaseRunning {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("cannot stop an instance in state %q", status))
			return
		}
		wantStatus = apiv1.BareMetalPhaseStopped
	case "reboot", "restart":
		if status != apiv1.BareMetalPhaseRunning {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("cannot reboot an instance in state %q", status))
			return
		}
		wantStatus, verb = apiv1.BareMetalPhaseRunning, "reboot"
	case "powercycle", "powerreset":
		if status != apiv1.BareMetalPhaseRunning {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("cannot run %q on an instance in state %q", rt.verb, status))
			return
		}
		wantStatus, verb = apiv1.BareMetalPhaseRunning, rt.verb
	case "reinstallos":
		if status != apiv1.BareMetalPhaseRunning {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("cannot reinstall the OS of an instance in state %q", status))
			return
		}
		if img, _ := payload["image"].(string); img == "" {
			writeErr(w, http.StatusBadRequest, "reinstall requires an image")
			return
		}
		wantStatus = apiv1.BareMetalPhaseRunning
	default:
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("action %q is not supported", rt.verb))
		return
	}
	now := time.Now()
	in.timeline = []phaseAt{
		{now, apiv1.BareMetalPhaseInProgress, "Running action " + verb + ".", verb},
		{now.Add(s.opts.Step), wantStatus, "Action " + verb + " completed.", verb},
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) remove(w http.ResponseWriter, rt route) {
	k := key(rt.project, rt.workspace, rt.name)
	in := s.instances[k]
	if in == nil {
		writeErr(w, s.opts.NotFoundStatus, "ent: bare metal not found")
		return
	}
	_, status := s.view(in)
	if status == apiv1.BareMetalPhaseDeleting {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	if status == apiv1.BareMetalPhaseDeletePending {
		writeErr(w, http.StatusPreconditionFailed, "This instance's deletion is taking longer than expected to finalize. Please contact your administrator.")
		return
	}
	now := time.Now()
	tl := []phaseAt{{now, apiv1.BareMetalPhaseDeleting, "Deleting instance " + rt.name + ".", "destroy"}}
	if s.ParkDelete != nil && s.ParkDelete(rt.name) {
		tl = append(tl, phaseAt{now.Add(s.opts.Step), apiv1.BareMetalPhaseDeletePending, "Instance " + rt.name + "'s deletion is taking longer than expected to finalize. Please contact your administrator.", "destroy"})
		in.timeline = tl
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	if s.FailDelete != nil {
		if reason := s.FailDelete(rt.name); reason != "" {
			tl = append(tl, phaseAt{now.Add(s.opts.Step), apiv1.BareMetalPhaseDestroyFailed, reason, "destroy"})
			in.timeline = tl
			writeJSON(w, http.StatusOK, map[string]any{})
			return
		}
	}
	tl = append(tl, phaseAt{now.Add(2 * s.opts.Step), "gone", "", "destroy"})
	in.timeline = tl
	writeJSON(w, http.StatusOK, map[string]any{})
}

// view returns the instance as the API shows it now, and its status.
func (s *Server) view(in *instance) (*convert.BareMetalObject, string) {
	cur := in.timeline[0]
	for _, p := range in.timeline {
		if !time.Now().Before(p.at) {
			cur = p
		}
	}
	out := in.obj
	st := apiv1.BareMetalStatus{Status: cur.status, Reason: cur.reason, Action: cur.action}
	if cur.status == apiv1.BareMetalPhaseFailed && s.ReasonDetail != nil {
		st.ReasonDetail = s.ReasonDetail(in.obj.Metadata.Name)
	}
	if cur.status == apiv1.BareMetalPhaseRunning || cur.status == apiv1.BareMetalPhaseStopped {
		st.Output = &apiv1.BareMetalOutput{HostName: in.obj.Metadata.Name + "-host", PrivateIP: "10.0.0.7", UserName: "ubuntu"}
	}
	out.Status = &st
	return &out, cur.status
}

// validate checks the spec against the SKU like the backend does.
func (s *Server) validate(sp apiv1.BareMetalSpec, p *apiv1.BmProfile, creating bool) string {
	if creating {
		if sp.Image == "" {
			return "spec.image is required"
		}
		if sp.SSHKey == "" && sp.UserPassword == "" {
			return "at least one of spec.sshKey or spec.userPassword is required"
		}
	}
	if sp.Image != "" && len(p.Spec.Image) > 0 && !contains(p.Spec.Image, sp.Image) {
		return fmt.Sprintf("image %q is not allowed by bm profile %q (allowed: %s)", sp.Image, p.Metadata.Name, strings.Join(p.Spec.Image, ", "))
	}
	if sp.SSHKey != "" && s.opts.SSHKeys != nil && !contains(s.opts.SSHKeys, sp.SSHKey) {
		return fmt.Sprintf("ssh key %q not found", sp.SSHKey)
	}
	if sp.Datacenter != "" && len(p.Spec.Datacenter) > 0 && !contains(p.Spec.Datacenter, sp.Datacenter) {
		return fmt.Sprintf("datacenter %q is not allowed by bm profile %q (allowed: %s)", sp.Datacenter, p.Metadata.Name, strings.Join(p.Spec.Datacenter, ", "))
	}
	if sp.AssignPublicIP && strings.EqualFold(p.Spec.AssignPublicIP, "Disabled") {
		return fmt.Sprintf("bm profile %q does not allow a public IP", p.Metadata.Name)
	}
	if sp.ContractTerm != 0 && len(p.Spec.MRCContractTerm) > 0 {
		ok := false
		for _, t := range p.Spec.MRCContractTerm {
			if t == sp.ContractTerm {
				ok = true
			}
		}
		if !ok {
			return fmt.Sprintf("contract term %d is not offered by bm profile %q", sp.ContractTerm, p.Metadata.Name)
		}
	}
	return ""
}

// derive adds the fields the backend fills in from the SKU.
func derive(sp apiv1.BareMetalSpec, p *apiv1.BmProfile) apiv1.BareMetalSpec {
	out := sp
	if out.Datacenter == "" && len(p.Spec.Datacenter) == 1 {
		out.Datacenter = p.Spec.Datacenter[0]
	}
	if out.GPUName == "" {
		out.GPUName = p.Spec.GPUName
	}
	out.ProvisionerType = p.Spec.ProvisionerType
	if out.Sharing == nil {
		out.Sharing = &apiv1.DevSharingSpec{ShareMode: "None"}
	}
	return out
}

// computeInstances serves the compute instances behind failed and deleted bare-metal instances.
func (s *Server) computeInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, paasPrefix), "/")
	project, rest, workspace := parts[0], parts[1:], ""
	if len(rest) >= 2 && rest[0] == "workspaces" {
		workspace, rest = rest[1], rest[2:]
	}
	if len(rest) == 0 || rest[0] != "computeinstances" {
		writeErr(w, http.StatusNotFound, "404 page not found")
		return
	}
	var items []convert.PaaSComputeInstance
	if r.URL.Query().Get("include_first_class") == "true" {
		mk := func(name string, st *convert.PaaSComputeInstanceStatus) convert.PaaSComputeInstance {
			return convert.PaaSComputeInstance{APIVersion: convert.PaaSAPIVersion, Kind: convert.PaaSComputeInstanceKind,
				Metadata: convert.DevMetadata{Name: name, Project: project, Workspace: workspace}, Status: st}
		}
		for k, in := range s.instances {
			ks := strings.SplitN(k, "/", 3)
			if ks[0] != project || ks[1] != workspace || s.FailureCause == nil {
				continue
			}
			if _, status := s.view(in); status != apiv1.BareMetalPhaseFailed {
				continue
			}
			if msg := s.FailureCause(ks[2]); msg != "" {
				items = append(items, mk(ks[2], &convert.PaaSComputeInstanceStatus{
					Status:         "failed",
					Reason:         "activity failed: " + msg,
					WorkflowStatus: map[string]any{"resources": []any{map[string]any{"errors": []any{map[string]any{"raw": msg}}}}},
				}))
			}
		}
		for k, until := range s.mirrors {
			ks := strings.SplitN(k, "/", 3)
			if ks[0] == project && ks[1] == workspace && time.Now().Before(until) {
				items = append(items, mk(ks[2], &convert.PaaSComputeInstanceStatus{Status: "destroying", Action: "destroy"}))
			}
		}
	}
	if len(rest) == 2 {
		for _, it := range items {
			if it.Metadata.Name == rest[1] {
				writeJSON(w, http.StatusOK, it)
				return
			}
		}
		writeErr(w, http.StatusNotFound, "workspace compute instance not found")
		return
	}
	writeJSON(w, http.StatusOK, convert.PaaSComputeInstanceList{APIVersion: convert.PaaSAPIVersion, Kind: convert.PaaSComputeInstanceListKind, Items: append([]convert.PaaSComputeInstance{}, items...)})
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"internal": msg, "code": 1, "external": http.StatusText(status)})
}
