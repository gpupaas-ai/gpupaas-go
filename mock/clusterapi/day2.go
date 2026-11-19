package clusterapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

// The dedicated day-2, status, audit and discovery routes of the v3.1.41 API reference:
//
//	POST   clusters/{c}/nodepools         DELETE clusters/{c}/nodepools/{pool}
//	POST   clusters/{c}/upgrade           PATCH  clusters/{c}/metadata
//	POST   clusters/{c}/redeploy          GET    clusters/{c}/status
//	GET    clusters/{c}/auditevents
//	GET    clusterprofiles/{p}/createoptions | availablecapacity
//	GET    clustermatchmakers/{call}
//	POST   clusternodes/{host}/{cordon|uncordon|drain}   PUT clusternodes/{host}/{metadata|taints}
//	DELETE clusternodes/{host}
//
// Every route addresses the project, whatever the scope of the cluster.

// subRoute is a parsed day-2 path.
type subRoute struct {
	project string
	kind    string // clusters | clusterprofiles | clusternodes | clustermatchmakers
	name    string
	tail    string
	pool    string
}

func parseSub(path string) (subRoute, bool) {
	if !strings.HasPrefix(path, prefix) {
		return subRoute{}, false
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) < 3 {
		return subRoute{}, false
	}
	r := subRoute{project: parts[0], kind: parts[1], name: parts[2]}
	switch r.kind {
	case "clusters":
		if len(parts) == 4 || len(parts) == 5 {
			r.tail = parts[3]
			if len(parts) == 5 {
				r.pool = parts[4]
			}
			return r, true
		}
	case "clusterprofiles":
		if len(parts) == 4 {
			r.tail = parts[3]
			return r, true
		}
	case "clusternodes":
		if len(parts) == 4 {
			r.tail = parts[3]
			return r, true
		}
		return r, len(parts) == 3 // DELETE clusternodes/{host}; a GET of the list never has a name
	case "clustermatchmakers":
		return r, len(parts) == 3
	}
	return subRoute{}, false
}

// day2 serves the routes above. It reports whether the request was one of them.
func (s *Server) day2(w http.ResponseWriter, r *http.Request) bool {
	rt, ok := parseSub(r.URL.Path)
	if !ok {
		return false
	}
	switch rt.kind {
	case "clustermatchmakers":
		if r.Method != http.MethodGet {
			return false
		}
		s.matchmaker(w, r, rt)
		return true
	case "clusterprofiles":
		if r.Method != http.MethodGet {
			return false
		}
		s.profileDiscovery(w, rt)
		return true
	case "clusternodes":
		if rt.tail == "" && r.Method != http.MethodDelete {
			return false
		}
		s.nodeOp(w, r, rt)
		return true
	}
	k, c := s.findByName(rt.project, rt.name)
	if c == nil {
		writeErr(w, s.opts.NotFoundStatus, "ent: cluster not found")
		return true
	}
	switch {
	case rt.tail == "nodepools" && rt.pool == "" && r.Method == http.MethodPost:
		s.addPool(w, r, k, c)
	case rt.tail == "nodepools" && rt.pool != "" && r.Method == http.MethodDelete:
		s.removePool(w, k, c, rt.pool)
	case rt.tail == "upgrade" && r.Method == http.MethodPost:
		s.upgrade(w, r, k, c)
	case rt.tail == "metadata" && r.Method == http.MethodPatch && s.opts.NoMetadataRoute:
		writeErr(w, http.StatusNotFound, "404 page not found")
	case rt.tail == "metadata" && r.Method == http.MethodPatch:
		s.patchMetadata(w, r, k, c)
	case rt.tail == "redeploy" && r.Method == http.MethodPost:
		s.redeploy(w, k, c)
	case rt.tail == "status" && r.Method == http.MethodGet:
		s.status(w, c)
	case rt.tail == "auditevents" && r.Method == http.MethodGet:
		s.auditEvents(w, r, k)
	default:
		writeErr(w, http.StatusNotFound, "404 page not found")
	}
	return true
}

// findByName returns the cluster of a project by name, whatever its workspace.
func (s *Server) findByName(project, name string) (string, *cluster) {
	for k, c := range s.clusters {
		if strings.HasPrefix(k, project+"/") && c.obj.Metadata.Name == name {
			return k, c
		}
	}
	return "", nil
}

func copySpec(spec apiv1.MKSClusterSpec) apiv1.MKSClusterSpec {
	b, _ := json.Marshal(spec)
	var out apiv1.MKSClusterSpec
	_ = json.Unmarshal(b, &out)
	return out
}

// record appends an audit event of a cluster.
func (s *Server) record(k, action, resourceType, resourceName, result, message string) {
	if s.audit == nil {
		s.audit = map[string][]apiv1.MKSAuditEvent{}
	}
	s.seq++
	s.audit[k] = append(s.audit[k], apiv1.MKSAuditEvent{
		ID:           "evt-" + strconv.Itoa(s.seq),
		ResourceType: resourceType, ResourceName: resourceName, Action: action, Actor: "tester",
		Message: message, Result: result,
		Timestamp:   apiv1.MKSInt64(1700000000 + int64(s.seq)),
		ClusterName: strings.SplitN(k, "/", 3)[2],
	})
}

// AuditActions returns the actions recorded for a cluster, oldest first.
func (s *Server) AuditActions(project, workspace, name string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, e := range s.audit[key(project, workspace, name)] {
		out = append(out, e.Action)
	}
	return out
}

// busy answers 412 while another operation runs, like the live platform. It reports whether it did.
func (s *Server) busy(w http.ResponseWriter, c *cluster) bool {
	_, phase := s.view(c, false)
	if phase == "MKS_CLUSTER_STATUS_RUNNING" {
		return false
	}
	writeErr(w, http.StatusPreconditionFailed, "the cluster is "+strings.ToLower(strings.TrimPrefix(phase, "MKS_CLUSTER_STATUS_"))+"; wait for the current operation to finish before starting another")
	return true
}

// day2Update applies a spec change as a day-2 operation: validated like an apply, versioned, UPDATING (UPGRADING
// for an upgrade) and then RUNNING again. change edits a copy of the spec and returns an error message to refuse it.
func (s *Server) day2Update(w http.ResponseWriter, k string, c *cluster, action, phase string, change func(spec *apiv1.MKSClusterSpec) string) {
	if s.busy(w, c) {
		return
	}
	spec := copySpec(c.obj.Spec)
	if msg := change(&spec); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	prof := s.profile(spec.Profile.Name)
	if prof == nil {
		writeErr(w, http.StatusBadRequest, "cluster profile not found")
		return
	}
	if msg := validate(spec, prof); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	if s.opts.PerPoolNetwork {
		if msg := validatePools(spec, &c.obj.Spec); msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
		if msg := validateStrict(spec, prof, &c.obj.Spec); msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
	}
	now := time.Now()
	next := c.obj
	next.Version++
	next.Spec = derive(spec, prof)
	if s.opts.PlatformAdditions {
		next.Spec = withPlatformAdditions(next.Spec)
		next.Metadata = withPlatformLabel(next.Metadata)
	}
	if s.opts.PerPoolNetwork && next.Spec.Networking != nil {
		next.Spec.Networking.VPC, next.Spec.Networking.Subnet = "", ""
	}
	tl := []phaseAt{
		{now, phase, "Cluster update in progress", action},
		{now.Add(s.opts.Step), "MKS_CLUSTER_STATUS_RUNNING", "Cluster is ready", action},
	}
	s.clusters[k] = &cluster{obj: next, timeline: tl, busyUntil: now.Add(4 * s.opts.Step)}
	view, _ := s.view(s.clusters[k], false)
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) addPool(w http.ResponseWriter, r *http.Request, k string, c *cluster) {
	var pool apiv1.MKSNodeGroup
	if err := json.NewDecoder(r.Body).Decode(&pool); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if pool.Name == "" || pool.Name == "control-plane" {
		writeErr(w, http.StatusBadRequest, "node pool name is required and control-plane is reserved")
		return
	}
	s.day2Update(w, k, c, "nodepool-add", "MKS_CLUSTER_STATUS_UPDATING", func(spec *apiv1.MKSClusterSpec) string {
		for _, p := range spec.WorkerPools {
			if p.Name == pool.Name {
				return fmt.Sprintf("a node pool named %q already exists", pool.Name)
			}
		}
		if pool.AssignPublicIP == nil { // an unset flag takes the existing workers' value
			for _, p := range spec.WorkerPools {
				if p.AssignPublicIP != nil {
					v := *p.AssignPublicIP
					pool.AssignPublicIP = &v
					break
				}
			}
		}
		spec.WorkerPools = append(spec.WorkerPools, pool)
		return ""
	})
	s.record(k, "cluster.nodepool.create", "nodePool", pool.Name, apiv1.MKSAuditAccepted, "")
}

func (s *Server) removePool(w http.ResponseWriter, k string, c *cluster, pool string) {
	s.day2Update(w, k, c, "nodepool-remove", "MKS_CLUSTER_STATUS_UPDATING", func(spec *apiv1.MKSClusterSpec) string {
		for i, p := range spec.WorkerPools {
			if p.Name == pool {
				spec.WorkerPools = append(spec.WorkerPools[:i:i], spec.WorkerPools[i+1:]...)
				return ""
			}
		}
		return fmt.Sprintf("node pool %q not found", pool)
	})
	s.record(k, "cluster.nodepool.delete", "nodePool", pool, apiv1.MKSAuditAccepted, "")
}

func (s *Server) upgrade(w http.ResponseWriter, r *http.Request, k string, c *cluster) {
	var req apiv1.MKSUpgradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	s.day2Update(w, k, c, "upgrade", "MKS_CLUSTER_STATUS_UPGRADING", func(spec *apiv1.MKSClusterSpec) string {
		if req.K8sVersion != "" {
			want := strings.TrimPrefix(req.K8sVersion, "v")
			ok := false
			for _, t := range upgradeTargets(spec.KubernetesVersion, s.profile(spec.Profile.Name)) {
				if want == t || strings.HasPrefix(want, t+".") {
					ok = true
				}
			}
			if !ok {
				return fmt.Sprintf("kubernetes version %q is not an upgrade target of this cluster (valid: %s)", req.K8sVersion,
					strings.Join(upgradeTargets(spec.KubernetesVersion, s.profile(spec.Profile.Name)), ", "))
			}
			spec.KubernetesVersion = req.K8sVersion
		}
		if req.Blueprint != "" {
			spec.Blueprint = &apiv1.MKSBlueprint{Name: req.Blueprint, Version: req.BlueprintVersion}
		} else if req.BlueprintVersion != "" && spec.Blueprint != nil {
			spec.Blueprint = &apiv1.MKSBlueprint{Name: spec.Blueprint.Name, Version: req.BlueprintVersion}
		}
		if req.CNI != nil && spec.Networking != nil {
			cni := *req.CNI
			spec.Networking.CNI = &cni
		}
		return ""
	})
	s.record(k, "cluster.upgrade", "cluster", c.obj.Metadata.Name, apiv1.MKSAuditAccepted, "")
}

// patchMetadata edits labels, annotations and tags in place: no phase change, no redeploy.
func (s *Server) patchMetadata(w http.ResponseWriter, r *http.Request, k string, c *cluster) {
	var body struct {
		Labels             map[string]string `json:"labels"`
		Annotations        map[string]string `json:"annotations"`
		Tags               map[string]string `json:"tags"`
		ReplaceLabels      bool              `json:"replaceLabels"`
		ReplaceAnnotations bool              `json:"replaceAnnotations"`
		ReplaceTags        bool              `json:"replaceTags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	_, phase := s.view(c, false)
	if phase == "MKS_CLUSTER_STATUS_DESTROYPENDING" || phase == "MKS_CLUSTER_STATUS_DESTROYING" {
		writeErr(w, http.StatusPreconditionFailed, "the cluster is being deleted")
		return
	}
	merge := func(held, in map[string]string, replace bool) map[string]string {
		if in == nil {
			return held
		}
		out := map[string]string{}
		if !replace {
			for kk, v := range held {
				out[kk] = v
			}
		}
		for kk, v := range in {
			out[kk] = v
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	c.obj.Metadata.Labels = merge(c.obj.Metadata.Labels, body.Labels, body.ReplaceLabels)
	c.obj.Metadata.Annotations = merge(c.obj.Metadata.Annotations, body.Annotations, body.ReplaceAnnotations)
	c.obj.Spec.Tags = merge(c.obj.Spec.Tags, body.Tags, body.ReplaceTags)
	c.obj.Version++
	if s.opts.ReconcileAfterMetadata {
		now := time.Now()
		c.timeline = append(c.timeline,
			phaseAt{now, "MKS_CLUSTER_STATUS_UPDATING", "Compute allocation in progress", "reconcile-nodes"},
			phaseAt{now.Add(3 * s.opts.Step), "MKS_CLUSTER_STATUS_RUNNING", "Node reconciliation complete", "reconcile-nodes"})
	}
	s.record(k, "cluster.metadata.update", "cluster", c.obj.Metadata.Name, apiv1.MKSAuditCompleted, "")
	view, _ := s.view(c, false)
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) redeploy(w http.ResponseWriter, k string, c *cluster) {
	_, phase := s.view(c, false)
	if phase != "MKS_CLUSTER_STATUS_ERROR" && phase != "MKS_CLUSTER_STATUS_DEGRADED" {
		writeErr(w, http.StatusPreconditionFailed, "redeploy is allowed only from ERROR or DEGRADED, the cluster is "+strings.TrimPrefix(phase, "MKS_CLUSTER_STATUS_"))
		return
	}
	now := time.Now()
	next := c.obj
	next.Version++
	c.obj = next
	c.timeline = []phaseAt{
		{now, "MKS_CLUSTER_STATUS_PROVISIONING", "Cluster provisioning", "redeploy"},
		{now.Add(s.opts.Step), "MKS_CLUSTER_STATUS_RUNNING", "Cluster is ready", "redeploy"},
	}
	c.previous, c.staleLeft = nil, 0
	s.record(k, "cluster.redeploy", "cluster", c.obj.Metadata.Name, apiv1.MKSAuditAccepted, "")
	view, _ := s.view(c, false)
	writeJSON(w, http.StatusOK, view)
}

// ---------------------------------------------------------------------------
// Status document and audit events
// ---------------------------------------------------------------------------

func (s *Server) status(w http.ResponseWriter, c *cluster) {
	view, phase := s.view(c, false)
	out := apiv1.MKSClusterStatusView{
		Metadata: apiv1.MKSStatusViewMetadata{Name: view.Metadata.Name, Project: view.Metadata.Project, Workspace: view.Metadata.Workspace, Labels: view.Metadata.Labels, Annotations: view.Metadata.Annotations},
		Spec: apiv1.MKSStatusViewSpec{
			KubernetesVersion: view.Spec.KubernetesVersion, Location: view.Spec.Location, Blueprint: view.Spec.Blueprint, Tags: view.Spec.Tags,
		},
		Status: apiv1.MKSStatusViewStatus{Status: phase, Reason: view.Status.Reason, Action: view.Status.Action,
			AvailableKubernetesVersions: view.Status.AvailableKubernetesVersions,
			Health:                      &apiv1.MKSClusterHealth{ControlPlane: "MKS_HEALTH_STATE_HEALTHY", Overall: "MKS_HEALTH_STATE_HEALTHY", NodePools: map[string]string{}}},
	}
	lifecycle := "ACTIVE"
	switch phase {
	case "MKS_CLUSTER_STATUS_PROVISIONING":
		lifecycle = "CREATING"
	case "MKS_CLUSTER_STATUS_UPDATING", "MKS_CLUSTER_STATUS_UPGRADING":
		lifecycle = "UPDATING"
	}
	up := phase == "MKS_CLUSTER_STATUS_RUNNING" || phase == "MKS_CLUSTER_STATUS_UPDATING" || phase == "MKS_CLUSTER_STATUS_UPGRADING"
	pool := func(g apiv1.MKSNodeGroup, hostPool, role string) apiv1.MKSPoolStatus {
		ps := apiv1.MKSPoolStatus{Name: g.Name, SKUName: g.SKUName, SKUType: g.SKUType, ComputeType: g.ComputeType, SKUDisplayName: g.SKUDisplayName,
			Datacenter: g.Datacenter, OSImage: g.OSImage, Scaling: g.Scaling, Labels: g.Labels, Annotations: g.Annotations, Tags: g.Tags, Status: lifecycle}
		for _, t := range g.Taints {
			if m, ok := t.(map[string]any); ok {
				key, _ := m["key"].(string)
				val, _ := m["value"].(string)
				eff, _ := m["effect"].(string)
				ps.Taints = append(ps.Taints, apiv1.MKSTaint{Key: key, Value: val, Effect: eff})
			}
		}
		if g.Scaling != nil && up {
			for i := int32(0); i < nodeCount(g.Scaling); i++ {
				ps.Nodes = append(ps.Nodes, apiv1.MKSNodeStatusView{
					Hostname: fmt.Sprintf("%s-%s-%d", view.Metadata.Name, hostPool, i), PrivateIP: fmt.Sprintf("10.0.0.%d", 10+int(i)),
					KubeVersion: view.Spec.KubernetesVersion, OS: "ubuntu", Arch: "amd64", Roles: []string{role}, Status: "READY",
					State: map[string]string{"DiskPressure": "false", "MemoryPressure": "false", "PIDPressure": "false", "ClusterNodeReady": "true"},
				})
			}
		}
		return ps
	}
	if cp := view.Spec.ControlPlane; cp != nil {
		ps := pool(*cp, "master", "Master")
		ps.Name = ""
		out.Spec.ControlPlane = &ps
	}
	for _, wp := range view.Spec.WorkerPools {
		out.Spec.WorkerPools = append(out.Spec.WorkerPools, pool(wp, wp.Name, "Worker"))
		out.Status.Health.NodePools[wp.Name] = "MKS_HEALTH_STATE_HEALTHY"
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) auditEvents(w http.ResponseWriter, r *http.Request, k string) {
	q := r.URL.Query()
	var match []apiv1.MKSAuditEvent
	// Newest first, like the console's list.
	events := s.audit[k]
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if v := q.Get("action"); v != "" && e.Action != v {
			continue
		}
		if v := q.Get("result"); v != "" && e.Result != v {
			continue
		}
		if v := q.Get("resourceType"); v != "" && e.ResourceType != v {
			continue
		}
		if v := q.Get("actor"); v != "" && e.Actor != v {
			continue
		}
		if v := q.Get("from"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err != nil || int64(e.Timestamp) < t.Unix() {
				continue
			}
		}
		if v := q.Get("to"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err != nil || int64(e.Timestamp) > t.Unix() {
				continue
			}
		}
		match = append(match, e)
	}
	total := len(match)
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if offset > total {
		offset = total
	}
	match = match[offset:]
	if limit > 0 && limit < len(match) {
		match = match[:limit]
	}
	items := make([]map[string]any, 0, len(match))
	for _, e := range match {
		// The platform writes 64-bit integers as strings.
		items = append(items, map[string]any{"id": e.ID, "resourceType": e.ResourceType, "resourceName": e.ResourceName, "action": e.Action,
			"actor": e.Actor, "message": e.Message, "timestamp": strconv.FormatInt(int64(e.Timestamp), 10), "result": e.Result, "clusterName": e.ClusterName})
	}
	writeJSON(w, http.StatusOK, map[string]any{"auditEvents": items, "metadata": map[string]any{"count": strconv.Itoa(total)}})
}

// ---------------------------------------------------------------------------
// Node operations
// ---------------------------------------------------------------------------

// NodeOps lists the single-node operations received, as "verb project/cluster/hostname".
func (s *Server) NodeOps() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.nodeOps...)
}

func (s *Server) nodeOp(w http.ResponseWriter, r *http.Request, rt subRoute) {
	clusterName := r.URL.Query().Get("cluster_name")
	if clusterName == "" {
		writeErr(w, http.StatusBadRequest, "cluster_name is required")
		return
	}
	_, c := s.findByName(rt.project, clusterName)
	if c == nil {
		writeErr(w, http.StatusNotFound, "cluster not found")
		return
	}
	known := false
	for _, n := range s.clusterNodes(rt.project, c) {
		known = known || n.Metadata.Name == rt.name
	}
	if !known {
		writeErr(w, http.StatusNotFound, fmt.Sprintf("node %q not found in cluster %q", rt.name, clusterName))
		return
	}
	var body map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	verb := rt.tail
	switch {
	case r.Method == http.MethodDelete && verb == "":
		verb = "delete"
	case r.Method == http.MethodPost && (verb == "cordon" || verb == "uncordon" || verb == "drain"):
	case r.Method == http.MethodPut && verb == "metadata":
		if len(body) == 0 {
			writeErr(w, http.StatusBadRequest, "a node metadata change must set or remove something")
			return
		}
		for _, l := range toStrings(body["removeLabels"]) {
			if _, set := mapOf(body["setLabels"])[l]; set {
				writeErr(w, http.StatusBadRequest, fmt.Sprintf("label %q is both set and removed", l))
				return
			}
		}
	case r.Method == http.MethodPut && verb == "taints":
		if _, ok := body["taints"].([]any); !ok {
			writeErr(w, http.StatusBadRequest, "taints is required (an empty list clears them)")
			return
		}
	default:
		writeErr(w, http.StatusNotFound, "404 page not found")
		return
	}
	s.nodeOps = append(s.nodeOps, verb+" "+rt.project+"/"+clusterName+"/"+rt.name)
	writeJSON(w, http.StatusOK, map[string]any{})
}

func mapOf(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func toStrings(v any) []string {
	var out []string
	if l, ok := v.([]any); ok {
		for _, x := range l {
			if str, ok := x.(string); ok {
				out = append(out, str)
			}
		}
	}
	return out
}

// clusterNodes lists the nodes behind a running cluster, control plane first.
func (s *Server) clusterNodes(project string, c *cluster) []convert.MKSNodeObject {
	view, phase := s.view(c, false)
	if phase != "MKS_CLUSTER_STATUS_RUNNING" && phase != "MKS_CLUSTER_STATUS_UPDATING" && phase != "MKS_CLUSTER_STATUS_UPGRADING" {
		return nil
	}
	name := c.obj.Metadata.Name
	var out []convert.MKSNodeObject
	add := func(pool string, count int32, role string, sku string) {
		for i := int32(0); i < count; i++ {
			host := fmt.Sprintf("%s-%s-%d", name, pool, i)
			out = append(out, convert.MKSNodeObject{
				APIVersion: convert.MKSAPIVersion,
				Kind:       convert.MKSNodeKind,
				Metadata:   convert.DevMetadata{Name: host, Labels: map[string]string{"cluster_name": name, "nodepoolname": pool}},
				Spec: apiv1.MKSNodeSpec{Hostname: host, KubeVersion: view.Spec.KubernetesVersion, Arch: "amd64", Roles: []string{role},
					PrivateIP: fmt.Sprintf("10.0.0.%d", 10+len(out)), OperatingSystem: "ubuntu", NodePool: pool, SKU: sku},
				Status: apiv1.MKSNodeStatus{Phase: "MKS_NODE_PHASE_RUNNING", EdgeStatus: "READY"},
			})
		}
	}
	if cp := view.Spec.ControlPlane; cp != nil && cp.Scaling != nil {
		add("master", cp.Scaling.Count, "Master", cp.SKUName)
	}
	for _, p := range view.Spec.WorkerPools {
		if p.Scaling != nil {
			add(p.Name, nodeCount(p.Scaling), "Worker", p.SKUName)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// SKU discovery and the matchmaker
// ---------------------------------------------------------------------------

func roleOptions(role string, g *apiv1.MKSGroupProfile, datacenters []string) map[string]any {
	if g == nil {
		return nil
	}
	return map[string]any{"name": role, "skuKind": lowerAll(g.SKUTypes), "computeType": lowerAll(g.ComputeTypes), "datacenters": datacenters,
		"skuNames": g.SKUNames, "images": g.Images}
}

func lowerAll(in []string) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = strings.ToLower(v)
	}
	return out
}

func (s *Server) profileDiscovery(w http.ResponseWriter, rt subRoute) {
	p := s.profile(rt.name)
	if p == nil {
		writeErr(w, http.StatusNotFound, "profile not found")
		return
	}
	switch rt.tail {
	case "createoptions":
		writeJSON(w, http.StatusOK, map[string]any{
			"controlPlane":                roleOptions("control-plane", p.Spec.ControlPlane, p.Spec.Datacenters),
			"workerNode":                  roleOptions("workers", p.Spec.Workers, p.Spec.Datacenters),
			"dataCenterRegions":           p.Spec.Datacenters,
			"defaultKubernetesVersion":    p.Spec.DefaultKubernetesVersion,
			"supportedKubernetesVersions": p.Spec.SupportedKubernetesVersions,
		})
	case "availablecapacity":
		if s.opts.NoCapacity {
			writeErr(w, http.StatusBadRequest, "unable to get available capacity: rpc error: code = InvalidArgument desc = available-capacity is not supported for this compute type")
			return
		}
		skus := []map[string]any{}
		for _, g := range []*apiv1.MKSGroupProfile{p.Spec.ControlPlane, p.Spec.Workers} {
			if g == nil {
				continue
			}
			for _, n := range g.SKUNames {
				skus = append(skus, map[string]any{"skuName": n, "available": 3})
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"profile": p.Metadata.Name, "skus": skus})
	default:
		writeErr(w, http.StatusNotFound, "404 page not found")
	}
}

// matchmaker answers the discovery calls from the mock's profiles, so that the SDK's queries and decoding can be
// tested. The query rules are those of the API reference: datacenter is required, sku_kind and compute_type
// are joined with "+".
func (s *Server) matchmaker(w http.ResponseWriter, r *http.Request, rt subRoute) {
	q := r.URL.Query()
	need := func(keys ...string) bool {
		for _, k := range keys {
			if len(q[k]) == 0 {
				writeErr(w, http.StatusBadRequest, k+" is required")
				return false
			}
		}
		return true
	}
	// The "+" of "bmaas+vmaas" reaches the server as a space or as a plus, whichever the client wrote.
	split := func(key string) []string {
		var out []string
		for _, v := range q[key] {
			out = append(out, strings.FieldsFunc(v, func(r rune) bool { return r == '+' || r == ' ' || r == ',' })...)
		}
		return out
	}
	var skuNames, images []string
	seenSKU, seenImg := map[string]bool{}, map[string]bool{}
	for _, p := range s.opts.Profiles {
		for _, g := range []*apiv1.MKSGroupProfile{p.Spec.ControlPlane, p.Spec.Workers} {
			if g == nil {
				continue
			}
			for _, n := range g.SKUNames {
				if !seenSKU[n] {
					seenSKU[n] = true
					skuNames = append(skuNames, n)
				}
			}
			for _, i := range g.Images {
				if !seenImg[i] {
					seenImg[i] = true
					images = append(images, i)
				}
			}
		}
	}
	sort.Strings(skuNames)
	sort.Strings(images)
	switch rt.name {
	case "kubernetesversions":
		seen := map[string]bool{}
		var minors []string
		def := ""
		for _, p := range s.opts.Profiles {
			if def == "" {
				def = p.Spec.DefaultKubernetesVersion
			}
			for _, v := range p.Spec.SupportedKubernetesVersions {
				if m := strings.Join(strings.SplitN(strings.TrimPrefix(v, "v"), ".", 3)[:2], "."); !seen[m] {
					seen[m] = true
					minors = append(minors, m)
				}
			}
		}
		sort.Sort(sort.Reverse(sort.StringSlice(minors)))
		versions := []map[string]any{}
		for _, m := range minors {
			versions = append(versions, map[string]any{"minor": m, "latestPatch": "v" + m + ".4", "isDefault": m == def})
		}
		writeJSON(w, http.StatusOK, map[string]any{"minors": minors, "versions": versions})
	case "availableimages":
		if !need("datacenter", "sku_kind", "compute_type") {
			return
		}
		list := []map[string]any{}
		for i, n := range images {
			list = append(list, map[string]any{"name": n, "default": i == 0})
		}
		writeJSON(w, http.StatusOK, map[string]any{"images": list})
	case "availableos":
		if !need("datacenter", "sku_kind", "compute_type") {
			return
		}
		groups := []map[string]any{{"name": "Ubuntu", "versions": []map[string]any{}}}
		for _, n := range images {
			groups[0]["versions"] = append(groups[0]["versions"].([]map[string]any), map[string]any{"version": n, "imageName": n})
		}
		writeJSON(w, http.StatusOK, map[string]any{"osImages": groups, "gpus": []string{}})
	case "availablespecs":
		if !need("datacenter", "sku_kind", "compute_type", "image") {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"specs": []map[string]any{{
			"skuKind": strings.Join(split("sku_kind"), "+"), "computeType": strings.Join(split("compute_type"), "+"), "cpu": 8, "ramGb": 32, "storageGb": 100,
			"bootDiskRange": map[string]any{"minGb": 50, "maxGb": 500, "incrementGb": 10, "unit": "GB"},
			"datacenters":   q["datacenter"], "skuCount": len(skuNames), "skuNames": skuNames, "displayName": firstOr(skuNames), "displayText": "8 vCPU · 32 GB Memory",
		}}})
	case "matchskus":
		if !need("datacenter", "sku_kind", "compute_type") {
			return
		}
		for _, t := range split("compute_type") {
			if t == "gpu" && q.Get("gpu_type") == "" {
				writeErr(w, http.StatusBadRequest, "gpu_type is required when compute_type includes gpu")
				return
			}
		}
		list := []map[string]any{}
		for _, n := range skuNames {
			list = append(list, map[string]any{"name": n, "id": "id-" + n, "kind": firstOr(lowerAll(split("sku_kind"))), "cpu": 8, "ramGb": 32, "storageGb": 100, "datacenters": q["datacenter"]})
		}
		writeJSON(w, http.StatusOK, map[string]any{"skus": list})
	case "availablecomputeskus":
		if !need("datacenter", "sku_kind", "compute_type", "image") {
			return
		}
		list := []map[string]any{}
		for _, n := range skuNames {
			list = append(list, map[string]any{"name": n, "kind": firstOr(lowerAll(split("sku_kind"))), "computeType": firstOr(split("compute_type")), "images": q["image"], "datacenters": q["datacenter"]})
		}
		writeJSON(w, http.StatusOK, map[string]any{"skus": list})
	default:
		writeErr(w, http.StatusNotFound, "404 page not found")
	}
}

func firstOr(list []string) string {
	if len(list) == 0 {
		return ""
	}
	return list[0]
}

// validateStrict applies the remaining validation rules of the v3.1.41 API reference (Options.PerPoolNetwork): the
// control plane has 1 or 3 nodes and keeps its size, a pool's datacenter is the cluster's location, a new pool needs
// a node password or an SSH key on its control plane or a pool when the profile asks for one and has no default, and
// ssh_key names a key resource.
// held is the spec of the cluster that exists (nil on create).
func validateStrict(spec apiv1.MKSClusterSpec, p *apiv1.MKSClusterProfile, held *apiv1.MKSClusterSpec) string {
	if cp := spec.ControlPlane; cp != nil && cp.Scaling != nil {
		if n := cp.Scaling.Count; n != 1 && n != 3 {
			return fmt.Sprintf("controlPlane.scaling.count: the control plane has 1 or 3 nodes, got %d", n)
		}
		if held != nil && held.ControlPlane != nil && held.ControlPlane.Scaling != nil && held.ControlPlane.Scaling.Count != cp.Scaling.Count {
			return "controlPlane.scaling.count: the control plane size can't be changed (convert to HA is not implemented); create the cluster with three control-plane nodes instead"
		}
	}
	// The platform installs one key per cluster: a new cluster needs a node password or an SSH key on its control plane
	// or on one of its pools when the profile asks the user to choose and has no default.
	asks := held == nil && p.Spec.ExposeSSHKeyChoice && p.Spec.DefaultSSHKey == "" && p.Spec.DefaultUserPassword == ""
	access := spec.ControlPlane != nil && (spec.ControlPlane.SSHKey != "" || spec.ControlPlane.UserPassword != "")
	for _, wp := range spec.WorkerPools {
		access = access || wp.SSHKey != "" || wp.UserPassword != ""
	}
	if asks && !access {
		return "Either a node password or an SSH key is required."
	}
	check := func(label string, g *apiv1.MKSNodeGroup) string {
		if g == nil {
			return ""
		}
		if g.Datacenter != "" && spec.Location != "" && g.Datacenter != spec.Location {
			return fmt.Sprintf("%s.datacenter: nodes are placed in spec.location (%q); a per-pool datacenter (%q) is not supported by the driver", label, spec.Location, g.Datacenter)
		}
		if strings.ContainsAny(g.SSHKey, " \t\n") || strings.HasPrefix(g.SSHKey, "-----BEGIN") {
			return label + ".sshKey must be the name of an SSH key resource; inline public keys are not accepted"
		}
		return ""
	}
	if msg := check("controlPlane", spec.ControlPlane); msg != "" {
		return msg
	}
	for i := range spec.WorkerPools {
		if msg := check(spec.WorkerPools[i].Name, &spec.WorkerPools[i]); msg != "" {
			return msg
		}
	}
	return ""
}

// withPlatformLabel adds the label the platform puts on every cluster it stores (Options.PlatformAdditions).
func withPlatformLabel(m convert.DevMetadata) convert.DevMetadata {
	labels := map[string]string{"paas.envmgmt.io/compute_type": "mks"}
	for k, v := range m.Labels {
		labels[k] = v
	}
	m.Labels = labels
	return m
}

// SetTags replaces the tags of a stored cluster, like a user editing them in the console. It reports whether the
// cluster exists.
func (s *Server) SetTags(project, workspace, name string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.clusters[key(project, workspace, name)]
	if c == nil {
		return false
	}
	c.obj.Spec.Tags = map[string]string{}
	for k, v := range tags {
		c.obj.Spec.Tags[k] = v
	}
	c.obj.Version++
	return true
}
