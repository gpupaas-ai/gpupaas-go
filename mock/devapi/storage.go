package devapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gpupaas-ai/gpupaas-go/convert"
)

// storageRec is one stored volume and where it is in its asynchronous run.
type storageRec struct {
	obj    *convert.DevStorage
	state  string // "deploying", "settled", "deleting", "parked"
	reads  int    // reads since the last write
	action string // the action that is running or ran last
}

// SeedStorageProfile publishes a storage profile (SKU) of the given service type, "high-speed-storage" or
// "object-storage".
func (s *Server) SeedStorageProfile(name, serviceType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profiles[name] = serviceType
}

// SetStoragePolls sets how many reads still show a changing record (default 1) before a write or a delete settles.
func (s *Server) SetStoragePolls(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storagePolls = n
}

// FailStorage makes the next provisioning of the named volume end failed.
func (s *Server) FailStorage(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failStorage[name] = true
}

// FailNextStorageDelete makes the next delete of the named volume end failed on destroy, as a live controller does
// once for a failed volume ("Please try again"); the delete after it works.
func (s *Server) FailNextStorageDelete(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failDelete[name] = true
}

// ParkStorageDeletes makes every delete end delete_pending, the state the platform keeps for an administrator;
// a second delete from the end user then answers 412.
func (s *Server) ParkStorageDeletes() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parkDeletes = true
}

// IgnoreStorageSizeChange makes an update accept a new size and report success without changing it.
func (s *Server) IgnoreStorageSizeChange() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ignoreSizeEdit = true
}

// FailStorageReads makes the next n reads of a single volume answer 500, as an outage would.
func (s *Server) FailStorageReads(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failGets = n
}

// SeedStorage stores a volume the way something outside the test created it: provisioned and successful.
// workspace is "" for project scope.
func (s *Server) SeedStorage(project, workspace, name, profile, size string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := &convert.DevStorage{
		APIVersion: convert.DevAPIVersion, Kind: convert.DevStorageKind,
		Metadata: convert.DevMetadata{Name: name, Project: project, Workspace: workspace},
		Spec:     convert.DevStorageSpec{Type: profile, Size: size, Sharing: &convert.DevSharingSpec{ShareMode: "None"}},
	}
	rec := &storageRec{obj: o, state: "settled", action: "deploy"}
	s.settle(rec)
	s.storages[project+"/"+workspace+"/"+name] = rec
}

// Storage returns the stored volume as the API would show it now (without advancing it), or nil.
// workspace is "" for project scope.
func (s *Server) Storage(project, workspace, name string) *convert.DevStorage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.storages[project+"/"+workspace+"/"+name]; ok {
		cp := *rec.obj
		return &cp
	}
	return nil
}

// DeleteStorage removes a volume behind the API's back and reports whether it existed.
func (s *Server) DeleteStorage(project, workspace, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := project + "/" + workspace + "/" + name
	_, ok := s.storages[k]
	delete(s.storages, k)
	return ok
}

func (s *Server) storageRoute(w http.ResponseWriter, r *http.Request, rt route) {
	s.mu.Lock()
	defer s.mu.Unlock()
	scope := rt.project + "/" + rt.workspace + "/"
	switch {
	case rt.name == "" && r.Method == http.MethodGet:
		want := r.URL.Query().Get("serviceType")
		items := []convert.DevStorage{}
		for k, rec := range s.storages {
			// a project-level list shows the volumes of its workspaces too, as the live API does
			if !inListScope(k, rt) {
				continue
			}
			if want != "" && s.profiles[rec.obj.Spec.Type] != want {
				continue
			}
			items = append(items, *rec.obj)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Metadata.Name < items[j].Metadata.Name })
		writeJSON(w, http.StatusOK, convert.DevStorageList{APIVersion: convert.DevAPIVersion, Kind: convert.DevStorageListKind, Items: items})
	case rt.name == "" && r.Method == http.MethodPost:
		s.storageUpsert(w, r, rt, scope)
	case rt.name != "" && r.Method == http.MethodGet:
		if s.failGets > 0 {
			s.failGets--
			writeErr(w, http.StatusInternalServerError, "temporarily unavailable")
			return
		}
		rec, ok := s.storages[scope+rt.name]
		if !ok {
			writeErr(w, http.StatusNotFound, fmt.Sprintf("storage %q not found", rt.name))
			return
		}
		s.advance(scope+rt.name, rec)
		if _, still := s.storages[scope+rt.name]; !still {
			writeErr(w, http.StatusNotFound, fmt.Sprintf("storage %q not found", rt.name))
			return
		}
		writeJSON(w, http.StatusOK, rec.obj)
	case rt.name != "" && r.Method == http.MethodDelete:
		rec, ok := s.storages[scope+rt.name]
		if !ok {
			writeErr(w, http.StatusNotFound, fmt.Sprintf("storage %q not found", rt.name))
			return
		}
		if rec.state == "parked" {
			writeErr(w, http.StatusPreconditionFailed, "This instance's deletion is taking longer than expected to finalize. Please contact your administrator.")
			return
		}
		rec.state, rec.reads, rec.action = "deleting", 0, "destroy"
		rec.obj.Status = convert.DevStorageStatus{Status: "deleting", Action: "destroy", Reason: "Deleting storage " + rt.name + "."}
		if s.failDelete[rt.name] {
			delete(s.failDelete, rt.name)
			rec.state = "settled"
			rec.obj.Status = convert.DevStorageStatus{Status: "failed", Action: "destroy",
				Reason: "Couldn't delete storage " + rt.name + ". Please try again; if the problem persists, contact your administrator."}
		}
		writeJSON(w, http.StatusOK, map[string]any{})
	default:
		writeErr(w, http.StatusMethodNotAllowed, r.Method+" is not supported")
	}
}

func (s *Server) storageUpsert(w http.ResponseWriter, r *http.Request, rt route, scope string) {
	var in convert.DevStorage
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Metadata.Name == "" {
		writeErr(w, http.StatusBadRequest, "invalid storage")
		return
	}
	if in.Spec.Type == "" {
		writeErr(w, http.StatusBadRequest, "type (the storage profile) is required")
		return
	}
	if _, ok := s.profiles[in.Spec.Type]; !ok {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("storage profile %q not found", in.Spec.Type))
		return
	}
	if in.Spec.Size != "" {
		if n, err := strconv.ParseInt(in.Spec.Size, 10, 64); err != nil || n <= 0 {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("size %q must be a positive number", in.Spec.Size))
			return
		}
	}
	for _, v := range []string{in.Spec.EnableEncryptionAtRest, in.Spec.EnableEncryptionInTransit} {
		if v != "" && v != "true" && v != "false" {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("encryption flag %q must be true or false", v))
			return
		}
	}
	in.APIVersion, in.Kind = convert.DevAPIVersion, convert.DevStorageKind
	in.Metadata.Project, in.Metadata.Workspace = rt.project, rt.workspace
	in.Spec.Storage = nil // the reference is accepted and not stored
	if in.Spec.Sharing == nil {
		in.Spec.Sharing = &convert.DevSharingSpec{ShareMode: "None"}
	}
	key := scope + in.Metadata.Name
	action := "deploy"
	if old, ok := s.storages[key]; ok {
		if old.obj.Spec.Type != in.Spec.Type {
			writeErr(w, http.StatusBadRequest, "the storage profile of an existing volume cannot change")
			return
		}
		action = "update"
		if s.ignoreSizeEdit {
			in.Spec.Size = old.obj.Spec.Size
		}
	}
	rec := &storageRec{obj: &in, state: "deploying", action: action}
	in.Status = convert.DevStorageStatus{Status: "in_progress", Action: action, Reason: "Deploying storage " + in.Metadata.Name + "."}
	if old, ok := s.storages[key]; ok && s.ignoreSizeEdit && old.obj.Spec.Size == in.Spec.Size {
		rec.state = "settled"
		s.settle(rec)
	}
	s.storages[key] = rec
	writeJSON(w, http.StatusOK, in)
}

// advance moves a record along its run: each read counts, and after storagePolls reads it settles.
func (s *Server) advance(key string, rec *storageRec) {
	switch rec.state {
	case "deploying":
		if rec.reads++; rec.reads > s.storagePolls {
			s.settle(rec)
		}
	case "deleting":
		if rec.reads++; rec.reads > s.storagePolls {
			if s.parkDeletes {
				rec.state = "parked"
				rec.obj.Status = convert.DevStorageStatus{Status: "delete_pending", Action: "destroy",
					Reason: "Instance " + rec.obj.Metadata.Name + "'s deletion is taking longer than expected to finalize. Please contact your administrator."}
				return
			}
			delete(s.storages, key)
		}
	}
}

// settle ends a deploy or an update: success with an output, or failed when the volume was set to fail.
func (s *Server) settle(rec *storageRec) {
	name := rec.obj.Metadata.Name
	if s.failStorage[name] {
		delete(s.failStorage, name)
		rec.state = "settled"
		rec.obj.Status = convert.DevStorageStatus{Status: "failed", Action: rec.action,
			Reason: "Couldn't provision storage " + name + "; provisioning is temporarily unavailable. Please contact your administrator."}
		return
	}
	rec.state = "settled"
	rec.obj.Status = convert.DevStorageStatus{Status: "success", Action: rec.action, Reason: "Storage Deployment Successful", Output: rawOutput(s.storageOutput(rec.obj))}
}

// storageOutput is an illustrative output: the keys of a filesystem follow what live controllers return; the keys
// of an object store are guessed, because no live object store has been seen.
func (s *Server) storageOutput(o *convert.DevStorage) map[string]any {
	size, _ := strconv.ParseInt(o.Spec.Size, 10, 64)
	if s.profiles[o.Spec.Type] == "object-storage" {
		return map[string]any{"bucket_name": o.Metadata.Name, "bucket_quota_gb": float64(size), "access_key_id": "AKIAMOCKMOCK", "secret_access_key": "mock-secret-access-key"}
	}
	return map[string]any{"name": "vol-" + o.Metadata.Name, "uid": "fs-0001", "filesystem_servers": "10.0.0.7",
		"mount_username": "mount_user", "mount_password": "mock-mount-password", "volume_size": float64(size) * 1e9}
}

// rawOutput encodes each value as the JSON the API would send.
func rawOutput(in map[string]any) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(in))
	for k, v := range in {
		raw, _ := json.Marshal(v)
		out[k] = raw
	}
	return out
}
