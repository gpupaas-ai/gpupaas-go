// Package clusterapi is an in-process mock of the first-class MKS API (paas.k8smgmt.io/v3)
// for tests. It reproduces what was observed on a live controller and in the paasctl
// regression script: apply is an upsert by name; provisioning and updates are asynchronous
// (PROVISIONING/UPDATING, then RUNNING); the status can flap after a write; the backend adds
// derived fields (computeType, shape, defaults from the profile) and a revision; DELETE starts
// an asynchronous teardown (DESTROYPENDING, DESTROYING, gone).
//
// Phase changes follow a timeline measured from the write, so tests need no goroutines and run
// in milliseconds (Options.Step).
package clusterapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

const prefix = "/apis/paas.k8smgmt.io/v3/projects/"

// Options configures a Server.
type Options struct {
	// Profiles are the cluster profiles (Kubernetes SKUs) known to the mock.
	Profiles []apiv1.MKSClusterProfile
	// Step is the delay between phase changes. Defaults to 20ms.
	Step time.Duration
	// Flap makes a cluster go RUNNING -> UPDATING -> RUNNING again after an update, like the
	// live backend does while it finishes a background re-render.
	Flap bool
	// StaleReads is how many reads after an update still return the previous spec and phase.
	StaleReads int
	// SSHKeys are the names of the SSH keys that exist in every workspace and project. When set,
	// a cluster that names another key is stored in ERROR and the create call answers 404
	// `trigger wci failed: not found: SSH key "x" doesn't exist.`, as the live API does (the
	// record is kept although the call fails). The keys are also served under dev.envmgmt.io.
	SSHKeys []string
	// MirrorLinger keeps the compute instance behind a deleted cluster visible, as "destroying" (with
	// include_first_class=true), for this long after the cluster record is gone. The live platform
	// refuses to delete a workspace until that instance is gone.
	MirrorLinger time.Duration
	// NotFoundStatus is the HTTP status for a missing cluster. Defaults to 404; the compute
	// API answers 502 with an "ent: ... not found" body, which the SDK also accepts.
	NotFoundStatus int
	// PoolOpFailedNodes makes the first N nodes of a pool report a failure in every pool operation.
	PoolOpFailedNodes int
	// PoolOpBusy makes the first N pool operations (cordon, uncordon, drain) answer 412 "cluster is
	// updating", as the live platform does while a cluster operation is still running.
	PoolOpBusy int
	// NodePageSize caps how many nodes one node-list response holds, whatever limit is asked for.
	// 0 means no cap beyond the requested limit.
	NodePageSize int
	// PerPoolNetwork makes the mock follow the rules of controller builds that take the network per node
	// pool (the v3.1.41 API reference): every new pool names a VPC and a subnet, the cluster-level
	// networking.vpc and subnet are cleared, a pool's network and boot disk are fixed once it exists,
	// only virtual-machine pools take a boot disk, all worker pools carry the same assignPublicIp, and
	// scaling and taints are checked. The control plane has 1 or 3 nodes and keeps its size, a pool's
	// datacenter is the location, a new cluster needs a password or an SSH key when the profile asks for one.
	PerPoolNetwork bool
	// PlatformAdditions makes the mock store pools the way a live platform shows them: it adds a label (also
	// on the cluster itself) and a taint of its own, lists taints in another order, leaves out an empty taint value, and omits
	// assignPublicIp when it is false. A client must not take any of that for a change.
	PlatformAdditions bool
	// VMProfiles are the VM SKUs served under dev.envmgmt.io (GET .../vmprofiles/{name}), for a client that checks a
	// pool's boot disk against its SKU.
	VMProfiles []apiv1.VMProfile
	// NoMetadataRoute makes PATCH .../clusters/{c}/metadata answer 404, like a controller that predates it.
	NoMetadataRoute bool
	// ReconcileAfterMetadata makes a metadata change (PATCH .../metadata) start a node reconcile like the live
	// controller does: the cluster reads UPDATING ("Compute allocation in progress", action reconcile-nodes) and
	// returns to RUNNING ("Node reconciliation complete") a few steps later.
	ReconcileAfterMetadata bool
	// NoCapacity makes GET .../clusterprofiles/{p}/availablecapacity answer 400 "not supported for this compute type",
	// like the live controller does for a bare metal profile.
	NoCapacity bool
	// IgnoreFailOnExists makes a create that carries ?fail-on-exists=true update an existing cluster
	// instead of answering 409, like a controller that does not know the parameter.
	IgnoreFailOnExists bool
}

// maskedPassword is what the backend shows instead of a node password.
const maskedPassword = "******"

// Server is the mock API. Create one with New and close it with Close.
type Server struct {
	*httptest.Server
	opts Options

	// poolOps and cordoned record the worker pool operations received (see PoolOps, Cordoned).
	poolOps  []string
	cordoned map[string]bool
	poolBusy int

	// FailProvision and FailDestroy inject failures: a non-empty reason makes the run end in
	// ERROR (provision/update) or DESTROYFAILED (delete) with that reason.
	FailProvision func(cluster string) string
	FailDestroy   func(cluster string) string
	// NodeFailure returns why a node of a cluster that ended in ERROR failed to provision. When
	// it returns a message, the mock serves a failed node compute instance named
	// "<cluster>-node-m0" with that workflow error, but only when the request carries
	// include_first_class=true: the live API hides the compute instances behind first-class
	// objects otherwise (GET by name answers 404, the list leaves them out).
	NodeFailure func(cluster string) string
	// ClusterFailure returns why the cluster's own workflow failed (served as a failed compute
	// instance named like the cluster, again only with include_first_class=true).
	ClusterFailure func(cluster string) string
	// RejectAfterStore makes a create answer 404 with the returned message after the cluster
	// was stored in ERROR, like a failed provisioning trigger on the live API.
	RejectAfterStore func(cluster string) string
	// UpdateFailure makes an update fail the way the live platform does: the cluster stays RUNNING
	// with a reason that says the changes could not be applied, while the cluster's own workflow
	// (the compute instance named like the cluster, served with include_first_class=true) is
	// in progress for a moment and then failed with the returned message.
	UpdateFailure func(cluster string) string

	mu       sync.Mutex
	audit    map[string][]apiv1.MKSAuditEvent // project/workspace/name -> audit trail, oldest first
	nodeOps  []string                         // single-node operations received, see NodeOps
	mirrors  map[string]time.Time             // project/workspace/name -> when the backing instance disappears
	clusters map[string]*cluster              // project/workspace/name
	requests []string
	orgs     map[string]bool
	seq      int
}

type phaseAt struct {
	at     time.Time
	phase  string // MKS_CLUSTER_STATUS_* or "GONE"
	reason string
	action string
}

type cluster struct {
	obj      convert.MKSClusterObject
	timeline []phaseAt
	// noInfra marks a cluster stored after its provisioning trigger failed: nothing exists to
	// tear down, and the live API refuses a plain DELETE of it (force=true removes the record).
	noInfra bool
	// previous is returned for staleLeft reads after an update.
	previous  *convert.MKSClusterObject
	prevPhase string
	staleLeft int
	// busyUntil is when the workflow of the last update ends; until then the cluster's compute
	// instance reads in progress. updateFailed is its failure message, shown afterwards.
	busyUntil    time.Time
	updateFailed string
}

// New starts a mock server.
func New(opts Options) *Server {
	if opts.Step <= 0 {
		opts.Step = 20 * time.Millisecond
	}
	if opts.NotFoundStatus == 0 {
		opts.NotFoundStatus = http.StatusNotFound
	}
	s := &Server{opts: opts, clusters: map[string]*cluster{}, orgs: map[string]bool{}, mirrors: map[string]time.Time{}}
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

// Cluster returns a snapshot of the stored cluster at its current phase, or nil.
func (s *Server) Cluster(project, workspace, name string) *apiv1.MKSCluster {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.clusters[key(project, workspace, name)]
	if c == nil {
		return nil
	}
	w, _ := s.view(c, false)
	return convert.FromMKSCluster(w, project, workspace)
}

// MirrorExists reports whether the compute instance behind a deleted cluster is still around.
func (s *Server) MirrorExists(project, workspace, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Now().Before(s.mirrors[key(project, workspace, name)])
}

// Exists reports whether the cluster is stored (it may be in a destroying phase).
func (s *Server) Exists(project, workspace, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.clusters[key(project, workspace, name)]
	return ok
}

func key(project, workspace, name string) string { return project + "/" + workspace + "/" + name }

func (s *Server) profile(name string) *apiv1.MKSClusterProfile {
	for i := range s.opts.Profiles {
		if s.opts.Profiles[i].Metadata.Name == name {
			return &s.opts.Profiles[i]
		}
	}
	return nil
}

// route is a parsed request path.
type route struct {
	project, workspace string
	kind               string // clusters | clusterprofiles | clusternodes
	name               string
}

func parse(path string) (route, bool) {
	if !strings.HasPrefix(path, prefix) {
		return route{}, false
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
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
	if len(rest) > 1 {
		r.name = rest[1]
	}
	switch r.kind {
	case "clusters", "clusterprofiles", "clusternodes":
		return r, len(rest) <= 2
	}
	return route{}, false
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
	if strings.HasPrefix(r.URL.Path, "/apis/dev.envmgmt.io/v1/projects/") {
		s.sshKeys(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/apis/paas.envmgmt.io/v1/projects/") {
		s.computeInstances(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/v2/sentry/kubeconfig/") {
		s.kubeconfig(w, r)
		return
	}
	if s.day2(w, r) {
		return
	}
	if project, pool, verb, ok := parsePoolOp(r.URL.Path); ok {
		s.poolOp(w, r, project, pool, verb)
		return
	}
	rt, ok := parse(r.URL.Path)
	if !ok {
		writeErr(w, http.StatusNotFound, "404 page not found")
		return
	}
	switch rt.kind {
	case "clusterprofiles":
		s.profiles(w, r, rt)
	case "clusternodes":
		s.nodes(w, r, rt)
	default:
		s.clustersRoute(w, r, rt)
	}
}

func (s *Server) profiles(w http.ResponseWriter, r *http.Request, rt route) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if rt.name != "" {
		p := s.profile(rt.name)
		if p == nil {
			writeErr(w, http.StatusNotFound, "profile not found")
			return
		}
		writeJSON(w, http.StatusOK, profileWire(p, rt.project))
		return
	}
	list := convert.MKSClusterProfileObjectList{APIVersion: convert.MKSAPIVersion, Kind: convert.MKSClusterProfileListKind}
	list.Items = []convert.MKSClusterProfileObject{}
	for i := range s.opts.Profiles {
		list.Items = append(list.Items, *profileWire(&s.opts.Profiles[i], rt.project))
	}
	list.Metadata.Count = int64(len(list.Items))
	writeJSON(w, http.StatusOK, list)
}

func profileWire(p *apiv1.MKSClusterProfile, project string) *convert.MKSClusterProfileObject {
	return &convert.MKSClusterProfileObject{
		APIVersion: convert.MKSAPIVersion,
		Kind:       convert.MKSClusterProfileKind,
		Metadata:   convert.DevMetadata{Name: p.Metadata.Name, Project: project, Labels: p.Metadata.Labels},
		Spec:       p.Spec,
		Status:     &apiv1.MKSClusterProfileStatus{Status: "ready"},
	}
}

func (s *Server) nodes(w http.ResponseWriter, r *http.Request, rt route) {
	name := r.URL.Query().Get("cluster_name")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "cluster_name is required")
		return
	}
	list := convert.MKSNodeObjectList{APIVersion: convert.MKSAPIVersion, Kind: convert.MKSNodeListKind, Items: []convert.MKSNodeObject{}}
	for k, c := range s.clusters {
		if !strings.HasPrefix(k, rt.project+"/") || c.obj.Metadata.Name != name {
			continue
		}
		list.Items = append(list.Items, s.clusterNodes(rt.project, c)...)
	}
	total := len(list.Items)
	list.Metadata.Count = int64(total)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if s.opts.NodePageSize > 0 && (limit <= 0 || limit > s.opts.NodePageSize) {
		limit = s.opts.NodePageSize
	}
	if offset < 0 || offset > total {
		offset = total
	}
	end := total
	if limit > 0 && offset+limit < total {
		end = offset + limit
	}
	list.Items = list.Items[offset:end]
	if limit > 0 {
		list.Metadata.Limit = int64(limit)
	}
	if offset > 0 {
		list.Metadata.Offset = int64(offset)
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) clustersRoute(w http.ResponseWriter, r *http.Request, rt route) {
	switch r.Method {
	case http.MethodGet:
		if rt.name == "" {
			s.list(w, rt)
			return
		}
		c := s.clusters[key(rt.project, rt.workspace, rt.name)]
		if c == nil {
			writeErr(w, s.opts.NotFoundStatus, "ent: cluster not found")
			return
		}
		view, phase := s.view(c, true)
		if phase == "GONE" {
			s.mirrors[key(rt.project, rt.workspace, rt.name)] = time.Now().Add(s.opts.MirrorLinger)
			delete(s.clusters, key(rt.project, rt.workspace, rt.name))
			writeErr(w, s.opts.NotFoundStatus, "ent: cluster not found")
			return
		}
		writeJSON(w, http.StatusOK, view)
	case http.MethodPost:
		if rt.name != "" {
			writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.apply(w, r, rt)
	case http.MethodDelete:
		s.remove(w, r, rt)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) list(w http.ResponseWriter, rt route) {
	list := convert.MKSClusterObjectList{APIVersion: convert.MKSAPIVersion, Kind: convert.MKSClusterListKind, Items: []convert.MKSClusterObject{}}
	for k, c := range s.clusters {
		ws := strings.SplitN(k, "/", 3)
		if ws[0] != rt.project || (rt.workspace != "" && ws[1] != rt.workspace) {
			continue
		}
		if view, phase := s.view(c, true); phase != "GONE" {
			list.Items = append(list.Items, *view)
		}
	}
	list.Metadata.Count = int64(len(list.Items))
	writeJSON(w, http.StatusOK, list)
}

// apply upserts a cluster, validating it against its profile like the backend does.
func (s *Server) apply(w http.ResponseWriter, r *http.Request, rt route) {
	var in convert.MKSClusterObject
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	name := in.Metadata.Name
	if name == "" {
		writeErr(w, http.StatusBadRequest, "metadata.name is required")
		return
	}
	if in.Spec.Profile == nil || in.Spec.Profile.Name == "" {
		writeErr(w, http.StatusBadRequest, "spec.profile.name is required")
		return
	}
	prof := s.profile(in.Spec.Profile.Name)
	if prof == nil {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("cluster profile %q not found", in.Spec.Profile.Name))
		return
	}
	k := key(rt.project, rt.workspace, name)
	existing := s.clusters[k]
	if existing != nil && r.URL.Query().Get("fail-on-exists") == "true" && !s.opts.IgnoreFailOnExists {
		writeErr(w, http.StatusConflict, fmt.Sprintf("A cluster named %q already exists. Choose a different name.", name))
		return
	}
	if msg := validate(in.Spec, prof); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	var held *apiv1.MKSClusterSpec
	if existing != nil {
		held = &existing.obj.Spec
	}
	restorePasswords(&in.Spec, held)
	if s.opts.PerPoolNetwork {
		if msg := validatePools(in.Spec, held); msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
		if msg := validateStrict(in.Spec, prof, held); msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
	}
	if existing != nil && in.Version != 0 && in.Version != existing.obj.Version {
		writeErr(w, http.StatusConflict, fmt.Sprintf("version conflict: have %d, got %d", existing.obj.Version, in.Version))
		return
	}

	now := time.Now()
	next := convert.MKSClusterObject{APIVersion: convert.MKSAPIVersion, Kind: convert.MKSClusterKind}
	next.Metadata = convert.DevMetadata{Name: name, Project: rt.project, Workspace: rt.workspace, Labels: in.Metadata.Labels, Annotations: in.Metadata.Annotations}
	next.Spec = derive(in.Spec, prof)
	if s.opts.PlatformAdditions {
		next.Spec = withPlatformAdditions(next.Spec)
		next.Metadata = withPlatformLabel(next.Metadata)
	}
	if s.opts.PerPoolNetwork && next.Spec.Networking != nil {
		// The pool network replaces the cluster-level one.
		next.Spec.Networking.VPC, next.Spec.Networking.Subnet = "", ""
	}
	next.Version = 1
	var tl []phaseAt
	reason := ""
	if s.FailProvision != nil {
		reason = s.FailProvision(name)
	}
	if existing == nil {
		// A trigger failure keeps the record, in ERROR, and fails the call.
		if msg := s.triggerFailure(name, in.Spec); msg != "" {
			next.Status = &apiv1.MKSClusterStatus{}
			s.clusters[k] = &cluster{noInfra: true, obj: next, timeline: []phaseAt{{now, "MKS_CLUSTER_STATUS_ERROR", "Couldn't provision cluster " + name + " due to an internal error. Please contact your administrator.", ""}}}
			writeErr(w, http.StatusNotFound, "unable to apply workspace mks cluster: trigger wci failed: "+msg)
			return
		}
		tl = []phaseAt{{now, "MKS_CLUSTER_STATUS_PROVISIONING", "Cluster provisioning", "deploy"}}
		if reason != "" {
			tl = append(tl, phaseAt{now.Add(s.opts.Step), "MKS_CLUSTER_STATUS_ERROR", reason, "deploy"})
		} else {
			tl = append(tl, phaseAt{now.Add(s.opts.Step), "MKS_CLUSTER_STATUS_RUNNING", "Cluster is ready", "deploy"})
		}
		s.seq++
		next.Status = &apiv1.MKSClusterStatus{Output: &apiv1.MKSClusterOutput{ClusterIDEdgesrv: name}}
	} else {
		prev, prevPhase := s.view(existing, false)
		next.Version = existing.obj.Version + 1
		next.Status = &apiv1.MKSClusterStatus{Output: existing.obj.Status.Output}
		tl = []phaseAt{{now, "MKS_CLUSTER_STATUS_UPDATING", "Cluster update in progress", "redeploy"}}
		updateFailed := ""
		if s.UpdateFailure != nil {
			updateFailed = s.UpdateFailure(name)
		}
		if updateFailed != "" {
			// The live platform keeps the cluster RUNNING and only changes the reason.
			tl = append(tl, phaseAt{now.Add(s.opts.Step), "MKS_CLUSTER_STATUS_RUNNING", "Couldn't apply the configuration changes to cluster " + name + " because a node failed to provision. Please contact your administrator.", "deploy"})
		} else if reason != "" {
			tl = append(tl, phaseAt{now.Add(s.opts.Step), "MKS_CLUSTER_STATUS_ERROR", reason, "redeploy"})
		} else {
			tl = append(tl, phaseAt{now.Add(s.opts.Step), "MKS_CLUSTER_STATUS_RUNNING", "Cluster is ready", "redeploy"})
			if s.opts.Flap {
				tl = append(tl,
					phaseAt{now.Add(2 * s.opts.Step), "MKS_CLUSTER_STATUS_UPDATING", "Cluster update in progress", "redeploy"},
					phaseAt{now.Add(3 * s.opts.Step), "MKS_CLUSTER_STATUS_RUNNING", "Cluster is ready", "redeploy"})
			}
		}
		c := &cluster{obj: next, timeline: tl, busyUntil: now.Add(4 * s.opts.Step), updateFailed: updateFailed}
		if s.opts.StaleReads > 0 {
			c.previous, c.prevPhase, c.staleLeft = prev, prevPhase, s.opts.StaleReads
		}
		s.clusters[k] = c
		s.record(k, "cluster.apply", "cluster", name, apiv1.MKSAuditAccepted, "")
		view, _ := s.view(c, false)
		writeJSON(w, http.StatusOK, view)
		return
	}
	c := &cluster{obj: next, timeline: tl}
	s.clusters[k] = c
	s.record(k, "cluster.apply", "cluster", name, apiv1.MKSAuditAccepted, "")
	view, _ := s.view(c, false)
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) remove(w http.ResponseWriter, r *http.Request, rt route) {
	k := key(rt.project, rt.workspace, rt.name)
	c := s.clusters[k]
	if c == nil {
		writeErr(w, s.opts.NotFoundStatus, "ent: cluster not found")
		return
	}
	if r.URL.Query().Get("force") == "true" {
		s.record(k, "cluster.delete.force", "cluster", rt.name, apiv1.MKSAuditAccepted, "")
	} else {
		s.record(k, "cluster.delete", "cluster", rt.name, apiv1.MKSAuditAccepted, "")
	}
	if c.noInfra {
		if r.URL.Query().Get("force") == "true" {
			s.mirrors[k] = time.Now().Add(s.opts.MirrorLinger)
			delete(s.clusters, k)
			writeJSON(w, http.StatusOK, map[string]any{})
			return
		}
		writeErr(w, http.StatusBadGateway, fmt.Sprintf("unable to delete mks cluster: teardown of cluster %s could not be started; retry the delete, or delete with force=true to remove the cluster record even if its infrastructure cannot be torn down", rt.name))
		return
	}
	_, phase := s.view(c, false)
	if phase == "MKS_CLUSTER_STATUS_DESTROYPENDING" || phase == "MKS_CLUSTER_STATUS_DESTROYING" {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	now := time.Now()
	tl := []phaseAt{
		{now, "MKS_CLUSTER_STATUS_DESTROYPENDING", "Cluster destroy requested", "destroy"},
		{now.Add(s.opts.Step), "MKS_CLUSTER_STATUS_DESTROYING", "Cluster is being destroyed", "destroy"},
	}
	if s.FailDestroy != nil {
		if reason := s.FailDestroy(rt.name); reason != "" {
			tl = append(tl, phaseAt{now.Add(2 * s.opts.Step), "MKS_CLUSTER_STATUS_DESTROYFAILED", reason, "destroy"})
			c.timeline, c.previous, c.staleLeft = tl, nil, 0
			writeJSON(w, http.StatusOK, map[string]any{})
			return
		}
	}
	tl = append(tl, phaseAt{now.Add(2 * s.opts.Step), "GONE", "", "destroy"})
	c.timeline, c.previous, c.staleLeft = tl, nil, 0
	writeJSON(w, http.StatusOK, map[string]any{})
}

// view returns the cluster as the API shows it now. consume counts a stale read.
func (s *Server) view(c *cluster, consume bool) (*convert.MKSClusterObject, string) {
	if c.staleLeft > 0 && c.previous != nil {
		if consume {
			c.staleLeft--
		}
		return c.previous, c.prevPhase
	}
	cur := c.timeline[0]
	for _, p := range c.timeline {
		if !time.Now().Before(p.at) {
			cur = p
		}
	}
	out := c.obj
	out.Spec = maskPasswords(out.Spec)
	st := apiv1.MKSClusterStatus{Status: cur.phase, Reason: cur.reason, Action: cur.action}
	if c.obj.Status != nil {
		st.Output = c.obj.Status.Output
	}
	if out.Spec.Profile != nil {
		st.AvailableKubernetesVersions = upgradeTargets(out.Spec.KubernetesVersion, s.profile(out.Spec.Profile.Name))
	}
	out.Status = &st
	return &out, cur.phase
}

// withPlatformAdditions returns the spec with each pool changed as PlatformAdditions describes. The
// input is not modified.
func withPlatformAdditions(spec apiv1.MKSClusterSpec) apiv1.MKSClusterSpec {
	shape := func(g apiv1.MKSNodeGroup) apiv1.MKSNodeGroup {
		labels := map[string]string{"platform/owned": "yes"}
		for k, v := range g.Labels {
			labels[k] = v
		}
		g.Labels = labels
		if g.AssignPublicIP != nil && !*g.AssignPublicIP {
			g.AssignPublicIP = nil
		}
		// Idempotent: a client sends back what it read, which already holds the platform's taint.
		taints := []any{map[string]any{"key": "platform/reserved", "effect": apiv1.MKSTaintPreferNoSchedule}}
		for i := len(g.Taints) - 1; i >= 0; i-- { // reversed
			if m, ok := g.Taints[i].(map[string]any); ok {
				if m["key"] == "platform/reserved" {
					continue
				}
				c := map[string]any{}
				for k, v := range m {
					if str, isString := v.(string); k == "value" && isString && str == "" {
						continue
					}
					c[k] = v
				}
				taints = append(taints, c)
			}
		}
		g.Taints = taints
		return g
	}
	if cp := spec.ControlPlane; cp != nil {
		c := shape(*cp)
		spec.ControlPlane = &c
	}
	if len(spec.WorkerPools) > 0 {
		pools := make([]apiv1.MKSNodeGroup, len(spec.WorkerPools))
		for i, wp := range spec.WorkerPools {
			pools[i] = shape(wp)
		}
		spec.WorkerPools = pools
	}
	return spec
}

// upgradeTargets lists the profile's versions one minor version above the current one: the platform
// upgrades one minor version at a time and never downgrades.
func upgradeTargets(current string, p *apiv1.MKSClusterProfile) []string {
	if p == nil {
		return nil
	}
	cur := strings.Split(strings.TrimPrefix(current, "v"), ".")
	if len(cur) < 2 {
		return nil
	}
	curMinor, err := strconv.Atoi(cur[1])
	if err != nil {
		return nil
	}
	var out []string
	for _, v := range p.Spec.SupportedKubernetesVersions {
		parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
		if len(parts) < 2 || parts[0] != cur[0] {
			continue
		}
		if minor, err := strconv.Atoi(parts[1]); err == nil && minor == curMinor+1 {
			out = append(out, v)
		}
	}
	return out
}

// maskPasswords returns the spec with every node password replaced by maskedPassword, without
// touching the stored one.
func maskPasswords(spec apiv1.MKSClusterSpec) apiv1.MKSClusterSpec {
	if cp := spec.ControlPlane; cp != nil && cp.UserPassword != "" {
		c := *cp
		c.UserPassword = maskedPassword
		spec.ControlPlane = &c
	}
	if len(spec.WorkerPools) > 0 {
		pools := make([]apiv1.MKSNodeGroup, len(spec.WorkerPools))
		copy(pools, spec.WorkerPools)
		for i := range pools {
			if pools[i].UserPassword != "" {
				pools[i].UserPassword = maskedPassword
			}
		}
		spec.WorkerPools = pools
	}
	return spec
}

// restorePasswords replaces a masked password in an update with the one the cluster holds, as the
// backend does: sending back what a read returned keeps the stored password.
func restorePasswords(in *apiv1.MKSClusterSpec, held *apiv1.MKSClusterSpec) {
	if held == nil {
		return
	}
	if cp := in.ControlPlane; cp != nil && cp.UserPassword == maskedPassword && held.ControlPlane != nil {
		c := *cp
		c.UserPassword = held.ControlPlane.UserPassword
		in.ControlPlane = &c
	}
	for i := range in.WorkerPools {
		if in.WorkerPools[i].UserPassword != maskedPassword {
			continue
		}
		for _, h := range held.WorkerPools {
			if h.Name == in.WorkerPools[i].Name {
				in.WorkerPools[i].UserPassword = h.UserPassword
			}
		}
	}
}

// PoolPassword returns the node password the mock stored for a pool ("" names the control plane).
func (s *Server) PoolPassword(project, workspace, cluster, pool string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.clusters[key(project, workspace, cluster)]
	if c == nil {
		return ""
	}
	if pool == "" {
		if c.obj.Spec.ControlPlane != nil {
			return c.obj.Spec.ControlPlane.UserPassword
		}
		return ""
	}
	for _, wp := range c.obj.Spec.WorkerPools {
		if wp.Name == pool {
			return wp.UserPassword
		}
	}
	return ""
}

// validatePools applies the per-pool rules of the v3.1.41 API reference. held is the spec of the
// cluster that exists (nil on create): a pool it already has keeps its network and boot disk.
func validatePools(spec apiv1.MKSClusterSpec, held *apiv1.MKSClusterSpec) string {
	heldGroup := func(name string, control bool) *apiv1.MKSNodeGroup {
		if held == nil {
			return nil
		}
		if control {
			return held.ControlPlane
		}
		for i := range held.WorkerPools {
			if held.WorkerPools[i].Name == name {
				return &held.WorkerPools[i]
			}
		}
		return nil
	}
	check := func(label string, g *apiv1.MKSNodeGroup, control bool) string {
		if g == nil {
			return ""
		}
		old := heldGroup(g.Name, control)
		nw := g.Network
		switch {
		case old == nil && (nw == nil || nw.VPC == "" && nw.Subnet == ""):
			return label + ".network: Select a VPC and a subnet for this pool."
		case old == nil && nw.Subnet == "":
			return fmt.Sprintf("%s.network.subnet: choose a subnet in VPC %q for this pool", label, nw.VPC)
		case old == nil && nw.VPC == "":
			return fmt.Sprintf("%s.network.vpc: choose the VPC that subnet %q belongs to", label, nw.Subnet)
		case old != nil && !sameNetwork(old.Network, nw):
			return label + ".network: a node pool's network can't be changed after the pool is created (its nodes are already placed in it); add a new node pool to use another network"
		}
		if g.BootDiskSizeGb != 0 && strings.EqualFold(g.SKUType, "BMAAS") {
			return label + ".bootDiskSizeGb: only virtual machine pools take a boot disk size"
		}
		if old != nil && old.BootDiskSizeGb != g.BootDiskSizeGb {
			return label + ".bootDiskSizeGb: a node pool's boot disk size can't be changed after the pool is created; add a new node pool for another size"
		}
		if sc := g.Scaling; sc != nil {
			if strings.EqualFold(sc.Mode, apiv1.MKSScalingAutomatic) {
				if sc.Min < 0 || sc.Max < 1 || sc.Min > sc.Desired || sc.Desired > sc.Max {
					return label + ".scaling: automatic scaling needs min >= 0, max >= 1 and min <= desired <= max"
				}
			} else if sc.Count < 0 {
				return label + ".scaling: count can't be negative"
			}
		}
		for _, t := range g.Taints {
			if m, ok := t.(map[string]any); ok {
				switch m["effect"] {
				case apiv1.MKSTaintNoSchedule, apiv1.MKSTaintPreferNoSchedule, apiv1.MKSTaintNoExecute:
				default:
					return fmt.Sprintf("pool %q: taint effect %v must be NoSchedule, PreferNoSchedule or NoExecute", g.Name, m["effect"])
				}
			}
		}
		return ""
	}
	if msg := check("controlPlane", spec.ControlPlane, true); msg != "" {
		return msg
	}
	var first *apiv1.MKSNodeGroup
	for i := range spec.WorkerPools {
		wp := &spec.WorkerPools[i]
		if msg := check(wp.Name, wp, false); msg != "" {
			return msg
		}
		if wp.AssignPublicIP == nil {
			continue
		}
		if first == nil {
			first = wp
		} else if *first.AssignPublicIP != *wp.AssignPublicIP {
			return fmt.Sprintf("assignPublicIp: worker pools %q (%t) and %q (%t) disagree; the driver assigns public IPs to all worker nodes or none, so every worker pool must carry the same value",
				first.Name, *first.AssignPublicIP, wp.Name, *wp.AssignPublicIP)
		}
	}
	return ""
}

func sameNetwork(a, b *apiv1.MKSNodePoolNetwork) bool {
	var x, y apiv1.MKSNodePoolNetwork
	if a != nil {
		x = *a
	}
	if b != nil {
		y = *b
	}
	return x == y
}

// validate checks the spec against the profile like the backend does.
func validate(spec apiv1.MKSClusterSpec, p *apiv1.MKSClusterProfile) string {
	if v := strings.TrimPrefix(spec.KubernetesVersion, "v"); v != "" && len(p.Spec.SupportedKubernetesVersions) > 0 {
		ok := false
		for _, sv := range p.Spec.SupportedKubernetesVersions {
			if v == sv || strings.HasPrefix(v, sv+".") {
				ok = true
			}
		}
		if !ok {
			return fmt.Sprintf("kubernetes version %q is not supported by profile %q (supported: %s)", spec.KubernetesVersion, p.Metadata.Name, strings.Join(p.Spec.SupportedKubernetesVersions, ", "))
		}
	}
	// The live API does not apply profile defaults: it rejects a create that lacks these.
	if spec.Networking == nil || spec.Networking.CNI == nil || spec.Networking.CNI.Name == "" {
		return "unable to apply workspace mks cluster: spec.networking.cni.name is required"
	}
	if spec.ControlPlane == nil || spec.ControlPlane.Scaling == nil || spec.ControlPlane.Scaling.Count < 1 {
		return "unable to apply workspace mks cluster: spec.controlPlane.scaling.count is required"
	}
	check := func(label string, g *apiv1.MKSNodeGroup, allowed *apiv1.MKSGroupProfile) string {
		if g == nil || allowed == nil {
			return ""
		}
		if g.SKUName != "" && len(allowed.SKUNames) > 0 && !contains(allowed.SKUNames, g.SKUName) {
			return fmt.Sprintf("%s sku %q is not allowed by profile %q", label, g.SKUName, p.Metadata.Name)
		}
		if g.OSImage != "" && len(allowed.Images) > 0 && !contains(allowed.Images, g.OSImage) {
			return fmt.Sprintf("%s image %q is not allowed by profile %q", label, g.OSImage, p.Metadata.Name)
		}
		return ""
	}
	if msg := check("control plane", spec.ControlPlane, p.Spec.ControlPlane); msg != "" {
		return msg
	}
	for _, wp := range spec.WorkerPools {
		wp := wp
		if msg := check("worker pool "+wp.Name, &wp, p.Spec.Workers); msg != "" {
			return msg
		}
	}
	return ""
}

// derive adds the fields the backend fills in: profile defaults and derived node group fields.
func derive(spec apiv1.MKSClusterSpec, p *apiv1.MKSClusterProfile) apiv1.MKSClusterSpec {
	out := spec
	if out.KubernetesVersion == "" {
		out.KubernetesVersion = "v" + p.Spec.DefaultKubernetesVersion
	} else if !strings.HasPrefix(out.KubernetesVersion, "v") {
		out.KubernetesVersion = "v" + out.KubernetesVersion + ".4"
	}
	if out.Location == "" && len(p.Spec.Datacenters) > 0 {
		out.Location = p.Spec.Datacenters[0]
	}
	if out.Blueprint == nil && p.Spec.DefaultBlueprint != nil {
		bp := *p.Spec.DefaultBlueprint
		out.Blueprint = &bp
	}
	nw := apiv1.MKSNetworking{VPC: "default", Subnet: "default", PodSubnet: p.Spec.DefaultPodSubnet, ServiceSubnet: p.Spec.DefaultServiceSubnet}
	if out.Networking != nil {
		in := *out.Networking
		if in.VPC != "" {
			nw.VPC = in.VPC
		}
		if in.Subnet != "" {
			nw.Subnet = in.Subnet
		}
		if in.PodSubnet != "" {
			nw.PodSubnet = in.PodSubnet
		}
		if in.ServiceSubnet != "" {
			nw.ServiceSubnet = in.ServiceSubnet
		}
		nw.CNI = in.CNI
	}
	if nw.CNI == nil && p.Spec.DefaultCNI != nil {
		cni := *p.Spec.DefaultCNI
		nw.CNI = &cni
	}
	out.Networking = &nw
	if out.Sharing == nil {
		out.Sharing = &apiv1.DevSharingSpec{ShareMode: "None"}
	}
	if out.Billing == nil {
		out.Billing = &apiv1.MKSBilling{Variables: map[string]string{"ContractTerm": "0"}}
	}
	decorate := func(g apiv1.MKSNodeGroup, def *apiv1.MKSGroupProfile) apiv1.MKSNodeGroup {
		g.DisplayName = g.Name
		g.ComputeType = "GPU"
		g.SKUDisplayName, g.SKUDisplayText = "custom", "custom"
		g.Shape = map[string]any{"gpu": map[string]any{"count": 1, "model": "custom"}}
		if def != nil {
			if g.OSImage == "" {
				g.OSImage = def.DefaultImage
			}
			if g.SKUName == "" {
				g.SKUName = def.DefaultSKUName
			}
		}
		if g.Scaling != nil && g.Scaling.Mode == "" {
			sc := *g.Scaling
			sc.Mode = "manual"
			g.Scaling = &sc
		}
		return g
	}
	if out.ControlPlane != nil {
		cp := decorate(*out.ControlPlane, p.Spec.ControlPlane)
		out.ControlPlane = &cp
	}
	if len(out.WorkerPools) > 0 {
		pools := make([]apiv1.MKSNodeGroup, len(out.WorkerPools))
		for i, wp := range out.WorkerPools {
			pools[i] = decorate(wp, p.Spec.Workers)
		}
		out.WorkerPools = pools
	}
	return out
}

// nodeCount is how many nodes a scaling setting stands for: the count of a manual group, the current
// size of an automatic one.
func nodeCount(sc *apiv1.MKSScaling) int32 {
	if strings.EqualFold(sc.Mode, apiv1.MKSScalingAutomatic) {
		return sc.Desired
	}
	return sc.Count
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

// triggerFailure returns why provisioning cannot even start, or "".
func (s *Server) triggerFailure(name string, spec apiv1.MKSClusterSpec) string {
	if s.RejectAfterStore != nil {
		if msg := s.RejectAfterStore(name); msg != "" {
			return msg
		}
	}
	if s.opts.SSHKeys != nil {
		groups := append([]apiv1.MKSNodeGroup{}, spec.WorkerPools...)
		if spec.ControlPlane != nil {
			groups = append(groups, *spec.ControlPlane)
		}
		for _, g := range groups {
			if g.SSHKey != "" && !contains(s.opts.SSHKeys, g.SSHKey) {
				return fmt.Sprintf("not found: SSH key %q doesn't exist.", g.SSHKey)
			}
		}
	}
	return ""
}

// sshKeys serves GET .../[workspaces/{ws}/]sshkeys[/{name}] from Options.SSHKeys.
func (s *Server) sshKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/apis/dev.envmgmt.io/v1/projects/"), "/")
	project, rest, workspace := parts[0], parts[1:], ""
	if len(rest) >= 2 && rest[0] == "workspaces" {
		workspace, rest = rest[1], rest[2:]
	}
	if len(rest) == 2 && rest[0] == "vmprofiles" {
		for i := range s.opts.VMProfiles {
			if p := s.opts.VMProfiles[i]; p.Metadata.Name == rest[1] {
				writeJSON(w, http.StatusOK, convert.VMProfileObject{APIVersion: convert.DevAPIVersion, Kind: "VmProfile",
					Metadata: convert.DevMetadata{Name: p.Metadata.Name, Project: project}, Spec: p.Spec})
				return
			}
		}
		writeErr(w, s.opts.NotFoundStatus, "ent: vm profile not found")
		return
	}
	if len(rest) == 0 || rest[0] != "sshkeys" {
		writeErr(w, http.StatusNotFound, "404 page not found")
		return
	}
	key := func(name string) map[string]any {
		return map[string]any{"apiVersion": "dev.envmgmt.io/v1", "kind": "SshKey", "metadata": map[string]any{"name": name, "project": project, "workspace": workspace}, "spec": map[string]any{}}
	}
	if len(rest) == 2 {
		if !contains(s.opts.SSHKeys, rest[1]) {
			writeErr(w, s.opts.NotFoundStatus, "ent: ssh key not found")
			return
		}
		writeJSON(w, http.StatusOK, key(rest[1]))
		return
	}
	items := []map[string]any{}
	for _, n := range s.opts.SSHKeys {
		items = append(items, key(n))
	}
	writeJSON(w, http.StatusOK, map[string]any{"apiVersion": "dev.envmgmt.io/v1", "kind": "SshKeyList", "metadata": map[string]any{"count": len(items)}, "items": items})
}

// computeInstances serves the failed node compute instances described by NodeFailure.
func (s *Server) computeInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/apis/paas.envmgmt.io/v1/projects/"), "/")
	project, rest, workspace := parts[0], parts[1:], ""
	if len(rest) >= 2 && rest[0] == "workspaces" {
		workspace, rest = rest[1], rest[2:]
	}
	if len(rest) == 0 || rest[0] != "computeinstances" {
		writeErr(w, http.StatusNotFound, "404 page not found")
		return
	}
	visible := r.URL.Query().Get("include_first_class") == "true"
	var items []convert.PaaSComputeInstance
	if visible {
		// The workflow of the last update: in progress, then failed when the update failed.
		for k, c := range s.clusters {
			ks := strings.SplitN(k, "/", 3)
			if ks[0] != project || ks[1] != workspace || c.busyUntil.IsZero() {
				continue
			}
			it := convert.PaaSComputeInstance{
				APIVersion: convert.PaaSAPIVersion, Kind: convert.PaaSComputeInstanceKind,
				Metadata: convert.DevMetadata{Name: c.obj.Metadata.Name, Project: project, Workspace: workspace},
			}
			switch {
			case time.Now().Before(c.busyUntil):
				it.Status = &convert.PaaSComputeInstanceStatus{Status: "inprogress", Reason: "activity in progress: group.res-mks-oneclick.custom.mks-oneclick"}
			case c.updateFailed != "":
				it.Status = &convert.PaaSComputeInstanceStatus{
					Status:         "failed",
					Reason:         "activity failed: " + c.updateFailed,
					WorkflowStatus: map[string]any{"resources": []any{map[string]any{"errors": []any{map[string]any{"raw": c.updateFailed}}}}},
				}
			default:
				continue
			}
			items = append(items, it)
		}
	}
	if visible && (s.NodeFailure != nil || s.ClusterFailure != nil) {
		for k, c := range s.clusters {
			ks := strings.SplitN(k, "/", 3)
			if ks[0] != project || ks[1] != workspace {
				continue
			}
			_, phase := s.view(c, false)
			if phase != "MKS_CLUSTER_STATUS_ERROR" {
				continue
			}
			failed := func(name, msg string) convert.PaaSComputeInstance {
				return convert.PaaSComputeInstance{
					APIVersion: convert.PaaSAPIVersion,
					Kind:       convert.PaaSComputeInstanceKind,
					Metadata:   convert.DevMetadata{Name: name, Project: project, Workspace: workspace},
					Status: &convert.PaaSComputeInstanceStatus{
						Status:         "failed",
						Reason:         "activity failed: " + msg,
						WorkflowStatus: map[string]any{"resources": []any{map[string]any{"errors": []any{map[string]any{"raw": msg}}}}},
					},
				}
			}
			if s.ClusterFailure != nil {
				if msg := s.ClusterFailure(c.obj.Metadata.Name); msg != "" {
					items = append(items, failed(c.obj.Metadata.Name, msg))
				}
			}
			if s.NodeFailure == nil {
				continue
			}
			msg := s.NodeFailure(c.obj.Metadata.Name)
			if msg == "" {
				continue
			}
			items = append(items, convert.PaaSComputeInstance{
				APIVersion: convert.PaaSAPIVersion,
				Kind:       convert.PaaSComputeInstanceKind,
				Metadata:   convert.DevMetadata{Name: c.obj.Metadata.Name + "-node-m0", Project: project, Workspace: workspace},
				Status: &convert.PaaSComputeInstanceStatus{
					Status:         "failed",
					Reason:         "activity failed: group.res-bmaas.custom.node-provision: " + msg,
					WorkflowStatus: map[string]any{"resources": []any{map[string]any{"errors": []any{map[string]any{"raw": msg}}}}},
				},
			})
		}
	}
	if visible {
		for k, until := range s.mirrors {
			ks := strings.SplitN(k, "/", 3)
			if ks[0] == project && ks[1] == workspace && time.Now().Before(until) {
				items = append(items, convert.PaaSComputeInstance{
					APIVersion: convert.PaaSAPIVersion,
					Kind:       convert.PaaSComputeInstanceKind,
					Metadata:   convert.DevMetadata{Name: ks[2], Project: project, Workspace: workspace},
					Status:     &convert.PaaSComputeInstanceStatus{Status: "destroying", Action: "destroy"},
				})
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

// parsePoolOp splits /apis/paas.k8smgmt.io/v3/projects/{project}/nodepools/{pool}/{verb}.
func parsePoolOp(path string) (project, pool, verb string, ok bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) != 4 || parts[1] != "nodepools" {
		return "", "", "", false
	}
	return parts[0], parts[2], parts[3], true
}

// poolOp serves cordon, uncordon and drain of a worker pool. It remembers what was asked, so tests
// can check it with PoolOps and Cordoned.
func (s *Server) poolOp(w http.ResponseWriter, r *http.Request, project, pool, verb string) {
	if r.Method != http.MethodPost || (verb != "cordon" && verb != "uncordon" && verb != "drain") {
		writeErr(w, http.StatusNotFound, "404 page not found")
		return
	}
	if s.poolBusy < s.opts.PoolOpBusy {
		s.poolBusy++
		writeJSON(w, http.StatusPreconditionFailed, map[string]any{
			"internal": "unable to " + verb + " mks node pool: cluster is updating (Compute allocation in progress); wait for the current operation to finish before starting another",
			"code":     1, "external": "Precondition Failed",
		})
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	clusterName := r.URL.Query().Get("cluster_name")
	if clusterName == "" {
		clusterName, _ = body["clusterName"].(string)
	}
	var found *cluster
	for k, c := range s.clusters {
		if strings.HasPrefix(k, project+"/") && c.obj.Metadata.Name == clusterName {
			found = c
		}
	}
	if found == nil {
		writeErr(w, http.StatusNotFound, "cluster not found")
		return
	}
	view, phase := s.view(found, false)
	if phase != "MKS_CLUSTER_STATUS_RUNNING" {
		writeErr(w, http.StatusConflict, "cluster is not running")
		return
	}
	known := false
	for _, p := range view.Spec.WorkerPools {
		known = known || p.Name == pool
	}
	if !known {
		writeErr(w, http.StatusNotFound, "node pool not found")
		return
	}
	if s.cordoned == nil {
		s.cordoned = map[string]bool{}
	}
	key := project + "/" + clusterName + "/" + pool
	switch verb {
	case "cordon", "drain":
		s.cordoned[key] = true
	case "uncordon":
		delete(s.cordoned, key)
	}
	s.poolOps = append(s.poolOps, verb+" "+key)
	// Like the live platform: one result per node of the pool, with the node's own message.
	var count int32
	for _, p := range view.Spec.WorkerPools {
		if p.Name == pool && p.Scaling != nil {
			count = nodeCount(p.Scaling)
		}
	}
	results := []map[string]any{}
	failed := 0
	for i := int32(0); i < count; i++ {
		res := map[string]any{"hostname": fmt.Sprintf("%s-%s-%d", clusterName, pool, i), "succeeded": true, "message": "node/" + pool + " " + verb + "ed"}
		if i < int32(s.opts.PoolOpFailedNodes) {
			res["succeeded"], res["message"] = false, "error: unable to "+verb+" the node"
			failed++
		}
		results = append(results, res)
	}
	writeJSON(w, http.StatusOK, map[string]any{"clusterName": clusterName, "nodePool": pool, "action": strings.ToUpper(verb[:1]) + verb[1:],
		"requested": count, "succeeded": int(count) - failed, "failed": failed, "results": results})
}

// PoolOps lists the pool operations received, as "verb project/cluster/pool".
func (s *Server) PoolOps() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.poolOps...)
}

// Cordoned reports whether the pool was cordoned (or drained) and not uncordoned since.
func (s *Server) Cordoned(project, cluster, pool string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cordoned[project+"/"+cluster+"/"+pool]
}

// kubeconfig serves GET /v2/sentry/kubeconfig/user?opts.selector=rafay.dev/clusterName=<name> for a
// running cluster. For any other cluster it answers 500 with an rpc error, as sentry does.
func (s *Server) kubeconfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || r.URL.Path != "/v2/sentry/kubeconfig/user" {
		writeErr(w, http.StatusNotFound, "404 page not found")
		return
	}
	name := strings.TrimPrefix(r.URL.Query().Get("opts.selector"), "rafay.dev/clusterName=")
	for _, c := range s.clusters {
		if c.obj.Metadata.Name != name {
			continue
		}
		if _, phase := s.view(c, false); phase == "MKS_CLUSTER_STATUS_RUNNING" {
			yaml := ("apiVersion: v1\nkind: Config\nclusters:\n- name: " + name + "\n  cluster:\n    server: https://mock.invalid:6443\ncontexts:\n- name: " + name + "\n  context:\n    cluster: " + name + "\n    user: mock\ncurrent-context: " + name + "\nusers:\n- name: mock\n  user:\n    token: mock-token\n")
			if strings.Contains(r.Header.Get("Accept"), "application/json") {
				writeJSON(w, http.StatusOK, map[string]string{"content_type": "application/yaml", "data": base64.StdEncoding.EncodeToString([]byte(yaml))})
				return
			}
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write([]byte(yaml))
			return
		}
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "rpc error: code = Unknown desc = pg: no rows in result set"})
}
