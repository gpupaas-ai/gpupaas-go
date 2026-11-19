// Package devapi is an in-process mock of the workspace, SSH key, security group and storage APIs for tests.
// It reproduces what was observed on live controllers: workspaces live on paas.envmgmt.io (and keep
// their metadata only: a spec sent with one is not stored) and SSH
// keys and security groups on dev.envmgmt.io, all with project scope and (for the dev kinds)
// workspace scope (a project-level list also shows the entries of the project's workspaces); POST on a collection is an upsert by name; only create, get, list and delete
// exist for SSH keys and security groups; a record is "success" as soon as it is stored; and a
// missing record answers 404.
//
// Storage is asynchronous like the live API: a write stores the record "in_progress" and it settles to
// "success" (with an illustrative output; the live keys of an object store are not known) after a number of
// reads; a delete shows "deleting" for a number of reads, then 404. The list takes the undocumented
// serviceType filter. How a failure, a parked delete or an ignored resize looks is what was observed on live
// controllers for VMs and bare metal; the knobs below switch them on.
package devapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"

	"github.com/gpupaas-ai/gpupaas-go/convert"
)

const (
	devPrefix  = "/apis/dev.envmgmt.io/v1/projects/"
	paasPrefix = "/apis/paas.envmgmt.io/v1/projects/"
)

// Server is the mock API. Create one with New and close it with Close.
type Server struct {
	*httptest.Server

	mu         sync.Mutex
	workspaces map[string]*convert.PaaSWorkspace // project/name
	sshKeys    map[string]*convert.DevSshKey     // project/workspace/name
	groups     map[string]*convert.DevSecurityGroup
	requests   []string
	orgs       map[string]bool

	profiles       map[string]string      // storage profile name -> service type
	storages       map[string]*storageRec // project/workspace/name
	storagePolls   int                    // reads that still show a changing record
	failStorage    map[string]bool
	failDelete     map[string]bool
	parkDeletes    bool
	ignoreSizeEdit bool
	failGets       int // storage item reads that still answer 500
}

// New starts a mock server.
func New() *Server {
	s := &Server{
		workspaces: map[string]*convert.PaaSWorkspace{},
		sshKeys:    map[string]*convert.DevSshKey{},
		groups:     map[string]*convert.DevSecurityGroup{},
		orgs:       map[string]bool{},

		profiles:     map[string]string{},
		storages:     map[string]*storageRec{},
		storagePolls: 1,
		failStorage:  map[string]bool{},
		failDelete:   map[string]bool{},
	}
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
	sort.Strings(out)
	return out
}

// Workspace returns the stored workspace, or nil.
func (s *Server) Workspace(project, name string) *convert.PaaSWorkspace {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.workspaces[project+"/"+name]
}

// SshKey returns the stored SSH key, or nil. workspace is "" for project scope.
func (s *Server) SshKey(project, workspace, name string) *convert.DevSshKey {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sshKeys[project+"/"+workspace+"/"+name]
}

// SecurityGroup returns the stored security group, or nil. workspace is "" for project scope.
func (s *Server) SecurityGroup(project, workspace, name string) *convert.DevSecurityGroup {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.groups[project+"/"+workspace+"/"+name]
}

// SeedSshKey stores a key the way something outside the test would have created it. workspace is "" for project scope.
func (s *Server) SeedSshKey(project, workspace, name, publicKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sshKeys[project+"/"+workspace+"/"+name] = &convert.DevSshKey{
		APIVersion: convert.DevAPIVersion, Kind: convert.DevSshKeyKind,
		Metadata: convert.DevMetadata{Name: name, Project: project, Workspace: workspace},
		Spec:     convert.DevSshKeySpec{PublicKey: publicKey, Sharing: &convert.DevSharingSpec{ShareMode: "None"}},
		Status:   convert.DevSshKeyStatus{Status: "success"},
	}
}

// DeleteSshKey removes a key behind the API's back and reports whether it existed.
func (s *Server) DeleteSshKey(project, workspace, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := project + "/" + workspace + "/" + name
	_, ok := s.sshKeys[k]
	delete(s.sshKeys, k)
	return ok
}

type route struct {
	project, workspace, kind, name string
}

// parse splits ".../projects/{p}[/workspaces/{w}]/{kind}[/{name}]" below the given prefix.
func parse(path, prefix string) (route, bool) {
	rest := strings.TrimPrefix(path, prefix)
	if rest == path {
		return route{}, false
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) < 2 {
		return route{}, false
	}
	rt := route{project: parts[0]}
	parts = parts[1:]
	if parts[0] == "workspaces" && len(parts) >= 3 {
		rt.workspace = parts[1]
		parts = parts[2:]
	}
	rt.kind = parts[0]
	if len(parts) > 1 {
		rt.name = parts[1]
	}
	if len(parts) > 2 {
		return route{}, false
	}
	return rt, true
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.requests = append(s.requests, r.Method+" "+r.URL.RequestURI())
	if o := r.Header.Get("X-Organization-ID"); o != "" {
		s.orgs[o] = true
	}
	s.mu.Unlock()

	if rt, ok := parse(r.URL.Path, paasPrefix); ok && rt.kind == "workspaces" && rt.workspace == "" {
		s.workspaceRoute(w, r, rt)
		return
	}
	if rt, ok := parse(r.URL.Path, devPrefix); ok {
		switch rt.kind {
		case "sshkeys":
			s.sshKeyRoute(w, r, rt)
			return
		case "securitygroups":
			s.groupRoute(w, r, rt)
			return
		case "storages":
			s.storageRoute(w, r, rt)
			return
		}
	}
	writeErr(w, http.StatusNotFound, "not found: "+r.URL.Path)
}

func (s *Server) workspaceRoute(w http.ResponseWriter, r *http.Request, rt route) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case rt.name == "" && r.Method == http.MethodGet:
		var items []convert.PaaSWorkspace
		for k, v := range s.workspaces {
			if strings.HasPrefix(k, rt.project+"/") {
				items = append(items, *v)
			}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Metadata.Name < items[j].Metadata.Name })
		writeJSON(w, http.StatusOK, convert.PaaSWorkspaceList{APIVersion: convert.PaaSWorkspaceAPIVer, Kind: convert.PaaSWorkspaceListKind, Items: items})
	case rt.name == "" && r.Method == http.MethodPost:
		var in convert.PaaSWorkspace
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Metadata.Name == "" {
			writeErr(w, http.StatusBadRequest, "invalid workspace")
			return
		}
		in.Metadata.Project = rt.project
		k := rt.project + "/" + in.Metadata.Name
		out := in
		out.APIVersion, out.Kind = convert.PaaSWorkspaceAPIVer, convert.PaaSWorkspaceKind
		if out.Metadata.DisplayName == "" {
			out.Metadata.DisplayName = out.Metadata.Name
		}
		s.workspaces[k] = &out
		writeJSON(w, http.StatusOK, out)
	case rt.name != "" && r.Method == http.MethodGet:
		if v, ok := s.workspaces[rt.project+"/"+rt.name]; ok {
			writeJSON(w, http.StatusOK, v)
			return
		}
		writeErr(w, http.StatusNotFound, fmt.Sprintf("workspace %q not found", rt.name))
	case rt.name != "" && r.Method == http.MethodDelete:
		k := rt.project + "/" + rt.name
		if _, ok := s.workspaces[k]; !ok {
			writeErr(w, http.StatusNotFound, fmt.Sprintf("workspace %q not found", rt.name))
			return
		}
		for _, k2 := range s.holding(rt.project, rt.name) {
			writeErr(w, http.StatusPreconditionFailed, "workspace still holds "+k2)
			return
		}
		delete(s.workspaces, k)
		writeJSON(w, http.StatusOK, map[string]any{})
	default:
		writeErr(w, http.StatusMethodNotAllowed, r.Method+" is not supported")
	}
}

// holding returns what a workspace still holds; a workspace cannot be deleted while it holds anything.
func (s *Server) holding(project, workspace string) []string {
	var out []string
	for k := range s.sshKeys {
		if strings.HasPrefix(k, project+"/"+workspace+"/") {
			out = append(out, "ssh key "+k)
		}
	}
	for k := range s.groups {
		if strings.HasPrefix(k, project+"/"+workspace+"/") {
			out = append(out, "security group "+k)
		}
	}
	for k := range s.storages {
		if strings.HasPrefix(k, project+"/"+workspace+"/") {
			out = append(out, "storage "+k)
		}
	}
	return out
}

func (s *Server) sshKeyRoute(w http.ResponseWriter, r *http.Request, rt route) {
	s.mu.Lock()
	defer s.mu.Unlock()
	scope := rt.project + "/" + rt.workspace + "/"
	switch {
	case rt.name == "" && r.Method == http.MethodGet:
		items := []convert.DevSshKey{}
		for k, v := range s.sshKeys {
			if inListScope(k, rt) {
				items = append(items, *v)
			}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Metadata.Name < items[j].Metadata.Name })
		writeJSON(w, http.StatusOK, convert.DevSshKeyList{APIVersion: convert.DevAPIVersion, Kind: convert.DevSshKeyListKind, Items: items})
	case rt.name == "" && r.Method == http.MethodPost:
		var in convert.DevSshKey
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Metadata.Name == "" {
			writeErr(w, http.StatusBadRequest, "invalid ssh key")
			return
		}
		if strings.TrimSpace(in.Spec.PublicKey) == "" {
			writeErr(w, http.StatusBadRequest, "public_key is required")
			return
		}
		in.APIVersion, in.Kind = convert.DevAPIVersion, convert.DevSshKeyKind
		in.Metadata.Project, in.Metadata.Workspace = rt.project, rt.workspace
		in.Status = convert.DevSshKeyStatus{Status: "success"}
		if in.Spec.Sharing == nil {
			in.Spec.Sharing = &convert.DevSharingSpec{ShareMode: "None"}
		}
		s.sshKeys[scope+in.Metadata.Name] = &in
		writeJSON(w, http.StatusOK, in)
	case rt.name != "" && r.Method == http.MethodGet:
		if v, ok := s.sshKeys[scope+rt.name]; ok {
			writeJSON(w, http.StatusOK, v)
			return
		}
		writeErr(w, http.StatusNotFound, fmt.Sprintf("ssh key %q not found", rt.name))
	case rt.name != "" && r.Method == http.MethodDelete:
		if _, ok := s.sshKeys[scope+rt.name]; !ok {
			writeErr(w, http.StatusNotFound, fmt.Sprintf("ssh key %q not found", rt.name))
			return
		}
		delete(s.sshKeys, scope+rt.name)
		writeJSON(w, http.StatusOK, map[string]any{})
	default:
		writeErr(w, http.StatusMethodNotAllowed, r.Method+" is not supported")
	}
}

func (s *Server) groupRoute(w http.ResponseWriter, r *http.Request, rt route) {
	s.mu.Lock()
	defer s.mu.Unlock()
	scope := rt.project + "/" + rt.workspace + "/"
	switch {
	case rt.name == "" && r.Method == http.MethodGet:
		items := []convert.DevSecurityGroup{}
		for k, v := range s.groups {
			if inListScope(k, rt) {
				items = append(items, *v)
			}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Metadata.Name < items[j].Metadata.Name })
		writeJSON(w, http.StatusOK, convert.DevSecurityGroupList{APIVersion: convert.DevAPIVersion, Kind: convert.DevSecurityGroupListKind, Items: items})
	case rt.name == "" && r.Method == http.MethodPost:
		var in convert.DevSecurityGroup
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Metadata.Name == "" {
			writeErr(w, http.StatusBadRequest, "invalid security group")
			return
		}
		for _, rule := range in.Spec.Rules {
			if rule.Action != "allow" && rule.Action != "deny" {
				writeErr(w, http.StatusBadRequest, fmt.Sprintf("rule action %q must be allow or deny", rule.Action))
				return
			}
		}
		in.APIVersion, in.Kind = convert.DevAPIVersion, convert.DevSecurityGroupKind
		in.Metadata.Project, in.Metadata.Workspace = rt.project, rt.workspace
		in.Status = convert.DevSecurityGroupStatus{Status: "success"}
		if in.Spec.Sharing == nil {
			in.Spec.Sharing = &convert.DevSharingSpec{ShareMode: "None"}
		}
		s.groups[scope+in.Metadata.Name] = &in
		writeJSON(w, http.StatusOK, in)
	case rt.name != "" && r.Method == http.MethodGet:
		if v, ok := s.groups[scope+rt.name]; ok {
			writeJSON(w, http.StatusOK, v)
			return
		}
		writeErr(w, http.StatusNotFound, fmt.Sprintf("security group %q not found", rt.name))
	case rt.name != "" && r.Method == http.MethodDelete:
		if _, ok := s.groups[scope+rt.name]; !ok {
			writeErr(w, http.StatusNotFound, fmt.Sprintf("security group %q not found", rt.name))
			return
		}
		delete(s.groups, scope+rt.name)
		writeJSON(w, http.StatusOK, map[string]any{})
	default:
		writeErr(w, http.StatusMethodNotAllowed, r.Method+" is not supported")
	}
}

// inListScope reports whether a stored key ("project/workspace/name") belongs in the list of a route. A project-level
// list shows the entries of its workspaces too, as the live API does; a workspace-level list shows only that workspace.
func inListScope(key string, rt route) bool {
	if rt.workspace == "" {
		return strings.HasPrefix(key, rt.project+"/")
	}
	return strings.HasPrefix(key, rt.project+"/"+rt.workspace+"/")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"internal": msg, "code": 1, "external": http.StatusText(status)})
}
