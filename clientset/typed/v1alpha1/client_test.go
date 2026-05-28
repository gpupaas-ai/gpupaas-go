package v1alpha1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func TestProjectClientGetByNameResolvesID(t *testing.T) {
	const projectID = "gkjnz20"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/auth/v1/projects/":
			_ = json.NewEncoder(w).Encode(convert.AuthProjectList{
				Count: 2,
				Results: []convert.AuthProject{
					{ID: "dk331kn", Name: "batch-d-mon-3-4-4550s"},
					{ID: projectID, Name: "test"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/auth/v1/projects/"+projectID+"/":
			_ = json.NewEncoder(w).Encode(convert.AuthProject{
				ID:          projectID,
				Name:        "test",
				Description: "Demo project",
				Default:     false,
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	got, err := cs.V1alpha1().Projects().Get(ctx, "test", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Name != "test" {
		t.Fatalf("name: %q", got.Metadata.Name)
	}
	if got.Spec.Description != "Demo project" {
		t.Fatalf("description: %q", got.Spec.Description)
	}
	if got.Metadata.Annotations["gpupaas.ai/project-id"] != projectID {
		t.Fatalf("annotations: %+v", got.Metadata.Annotations)
	}
	if got.APIVersion != apiv1.APIVersion {
		t.Fatalf("expected k8s apiVersion, got %q", got.APIVersion)
	}
}

func TestProjectClientGetNotFoundWhenNameMissingFromList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/auth/v1/projects/" {
			_ = json.NewEncoder(w).Encode(convert.AuthProjectList{
				Count:   1,
				Results: []convert.AuthProject{{ID: "abc", Name: "other"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = cs.V1alpha1().Projects().Get(context.Background(), "missing", gpupaas.GetOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !gpupaas.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestProjectClientCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/v1/projects/":
			var in convert.AuthProject
			_ = json.NewDecoder(r.Body).Decode(&in)
			in.ID = "new-id"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(in)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	created, err := cs.V1alpha1().Projects().Create(ctx, &apiv1.Project{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindProject},
		Metadata: apiv1.ObjectMeta{Name: "demo"},
		Spec:     apiv1.ProjectSpec{Description: "Demo project"},
	}, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if created.Metadata.Name != "demo" {
		t.Fatalf("unexpected create result: %+v", created)
	}
}

func TestWorkspaceClientPaaSAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/apis/paas.envmgmt.io/v1/projects/demo/workspaces":
			var in convert.PaaSWorkspace
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in.APIVersion != convert.PaaSWorkspaceAPIVer {
				t.Errorf("wire apiVersion: %q", in.APIVersion)
			}
			in.Status = convert.PaaSWorkspaceStatus{
				CommonStatus: convert.PaaSStatus{ConditionStatus: "StatusOK"},
			}
			_ = json.NewEncoder(w).Encode(in)
		case r.Method == http.MethodGet && r.URL.Path == "/apis/paas.envmgmt.io/v1/projects/demo/workspaces/ws1":
			_ = json.NewEncoder(w).Encode(convert.PaaSWorkspace{
				APIVersion: convert.PaaSWorkspaceAPIVer,
				Kind:       convert.PaaSWorkspaceKind,
				Metadata: convert.PaaSMetadata{
					Name:        "ws1",
					Project:     "demo",
					Description: "dev workspace",
				},
				Status: convert.PaaSWorkspaceStatus{
					CommonStatus: convert.PaaSStatus{ConditionStatus: "StatusOK"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	created, err := cs.V1alpha1().Workspaces("demo").Create(ctx, &apiv1.Workspace{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindWorkspace},
		Metadata: apiv1.ObjectMeta{Name: "ws1", Project: "demo"},
		Spec:     apiv1.WorkspaceSpec{Description: "dev workspace"},
	}, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if created.Metadata.Name != "ws1" || created.Status.Phase != "StatusOK" {
		t.Fatalf("create: %+v", created)
	}

	got, err := cs.V1alpha1().Workspaces("demo").Get(ctx, "ws1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.Description != "dev workspace" {
		t.Fatalf("get: %+v", got)
	}
}

func TestWorkspaceCollaboratorClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/apis/paas.envmgmt.io/v1/projects/demo/workspaces/ws1/assigncollaborators":
			if r.URL.Query().Get("ssoUsers") != "true" {
				t.Errorf("expected ssoUsers=true query, got %q", r.URL.RawQuery)
			}
			var in convert.PaaSWorkspaceAddCollaborators
			_ = json.NewDecoder(r.Body).Decode(&in)
			if len(in.Spec.Usernames) != 1 || in.Spec.Usernames[0] != "alice" {
				t.Errorf("assign usernames: %+v", in.Spec.Usernames)
			}
			in.Status = convert.PaaSStatus{ConditionStatus: "StatusOK"}
			_ = json.NewEncoder(w).Encode(in)
		case r.Method == http.MethodPost && r.URL.Path == "/apis/paas.envmgmt.io/v1/projects/demo/workspaces/ws1/collaborators":
			var in convert.PaaSWorkspaceCollaborator
			_ = json.NewDecoder(r.Body).Decode(&in)
			in.Status = convert.PaaSStatus{ConditionStatus: "StatusOK"}
			_ = json.NewEncoder(w).Encode(in)
		case r.Method == http.MethodGet && r.URL.Path == "/apis/paas.envmgmt.io/v1/projects/demo/workspaces/ws1/collaborators":
			_ = json.NewEncoder(w).Encode(convert.PaaSWorkspaceCollaboratorList{
				APIVersion: convert.PaaSWorkspaceAPIVer,
				Kind:       convert.PaaSWorkspaceCollaboratorListKind,
				Items: []convert.PaaSWorkspaceCollaborator{
					{
						Metadata: convert.PaaSMetadata{Name: "alice", Project: "demo"},
						Spec:     convert.PaaSWorkspaceCollaboratorSpec{Role: apiv1.WorkspaceRoleCollaboratorReadOnly},
					},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/apis/paas.envmgmt.io/v1/projects/demo/workspaces/ws1/unassigncollaborators":
			if r.URL.Query().Get("ssoUsers") != "true" {
				t.Errorf("expected ssoUsers=true on unassign, got %q", r.URL.RawQuery)
			}
			var in convert.PaaSWorkspaceDeleteCollaborators
			_ = json.NewDecoder(r.Body).Decode(&in)
			if len(in.Spec.Usernames) != 1 || in.Spec.Usernames[0] != "alice" {
				t.Errorf("unassign usernames: %+v", in.Spec.Usernames)
			}
			_ = json.NewEncoder(w).Encode(in)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	collabClient := cs.V1alpha1().Workspaces("demo").Collaborators("ws1")

	assigned, err := collabClient.Create(ctx, &apiv1.WorkspaceCollaborator{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindWorkspaceCollaborator},
		Metadata: apiv1.ObjectMeta{Name: "alice", Project: "demo", Workspace: "ws1"},
		Spec: apiv1.WorkspaceCollaboratorSpec{
			Username:  "alice",
			Role:      apiv1.WorkspaceRoleCollaboratorReadOnly,
			IsSSOUser: true,
		},
	}, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if assigned.Metadata.Name != "alice" {
		t.Fatalf("assign: %+v", assigned)
	}

	invited, err := collabClient.Create(ctx, &apiv1.WorkspaceCollaborator{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindWorkspaceCollaborator},
		Metadata: apiv1.ObjectMeta{Name: "guest@example.com", Project: "demo", Workspace: "ws1"},
		Spec: apiv1.WorkspaceCollaboratorSpec{
			Email: "guest@example.com",
			Role:  apiv1.WorkspaceRoleCollaborator,
		},
	}, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if invited.Spec.Email != "guest@example.com" {
		t.Fatalf("invite: %+v", invited)
	}

	list, err := collabClient.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Metadata.Name != "alice" {
		t.Fatalf("list: %+v", list.Items)
	}

	got, err := collabClient.Get(ctx, "alice", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.Role != apiv1.WorkspaceRoleCollaboratorReadOnly {
		t.Fatalf("get: %+v", got)
	}
	if got.Status.Role != apiv1.WorkspaceRoleCollaboratorReadOnly {
		t.Fatalf("status role: %q", got.Status.Role)
	}

	ssoTrue := true
	if err := collabClient.Delete(ctx, "alice", gpupaas.DeleteOptions{SSOUser: &ssoTrue}); err != nil {
		t.Fatal(err)
	}
}

func sampleDevVM(name string) convert.DevVirtualMachine {
	return convert.DevVirtualMachine{
		APIVersion: convert.DevAPIVersion,
		Kind:       convert.DevVirtualMachineKind,
		Metadata: convert.DevMetadata{
			Name:    name,
			Project: "demo",
		},
		Spec: convert.DevVirtualMachineSpec{
			VirtualMachine: convert.DevResourceRef{Name: "ubuntu-profile"},
			CPUCount:       "2",
			Memory:         "4Gi",
			Image:          "ubuntu-22.04",
		},
		Status: convert.DevVirtualMachineStatus{Status: "success"},
	}
}

func TestVirtualMachineClientProjectScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/apis/dev.envmgmt.io/v1/projects/demo/virtualmachines":
			_ = json.NewEncoder(w).Encode(convert.DevVirtualMachineList{
				APIVersion: convert.DevAPIVersion,
				Kind:       convert.DevVirtualMachineListKind,
				Items:      []convert.DevVirtualMachine{sampleDevVM("vm1")},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/apis/dev.envmgmt.io/v1/projects/demo/virtualmachines/vm1":
			_ = json.NewEncoder(w).Encode(sampleDevVM("vm1"))
		case r.Method == http.MethodPost && r.URL.Path == "/apis/dev.envmgmt.io/v1/projects/demo/virtualmachines":
			var in convert.DevVirtualMachine
			_ = json.NewDecoder(r.Body).Decode(&in)
			in.Status = convert.DevVirtualMachineStatus{Status: "pending"}
			_ = json.NewEncoder(w).Encode(in)
		case r.Method == http.MethodDelete && r.URL.Path == "/apis/dev.envmgmt.io/v1/projects/demo/virtualmachines/vm1":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/apis/dev.envmgmt.io/v1/projects/demo/virtualmachines/vm1/status":
			vm := sampleDevVM("vm1")
			vm.Status = convert.DevVirtualMachineStatus{Status: "pending", Reason: "provisioning"}
			_ = json.NewEncoder(w).Encode(vm)
		case r.Method == http.MethodPost && r.URL.Path == "/apis/dev.envmgmt.io/v1/projects/demo/virtualmachines/vm1/action/start":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	vms := cs.V1alpha1().VirtualMachines("demo")

	list, err := vms.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Metadata.Name != "vm1" {
		t.Fatalf("list: %+v", list.Items)
	}
	if list.Items[0].APIVersion != apiv1.APIVersion {
		t.Fatalf("list apiVersion: %q", list.Items[0].APIVersion)
	}
	if list.Items[0].Metadata.Workspace != "" {
		t.Fatalf("project-scoped workspace should be empty, got %q", list.Items[0].Metadata.Workspace)
	}

	got, err := vms.Get(ctx, "vm1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.CPUCount != "2" {
		t.Fatalf("get spec: %+v", got.Spec)
	}

	created, err := vms.Create(ctx, &apiv1.VirtualMachine{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindVirtualMachine},
		Metadata: apiv1.ObjectMeta{Name: "vm-new", Project: "demo"},
		Spec: apiv1.VirtualMachineSpec{
			VirtualMachine: apiv1.ResourceRef{Name: "ubuntu-profile"},
			CPUCount:       "2",
			Memory:         "4Gi",
		},
	}, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status.Status != "pending" {
		t.Fatalf("create status: %+v", created.Status)
	}

	status, err := vms.GetStatus(ctx, "vm1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.Status.Reason != "provisioning" {
		t.Fatalf("status: %+v", status.Status)
	}

	if _, err := vms.Start(ctx, "vm1", gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := vms.Delete(ctx, "vm1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestVirtualMachineClientWorkspaceScope(t *testing.T) {
	const base = "/apis/dev.envmgmt.io/v1/projects/demo/workspaces/dev/virtualmachines"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(convert.DevVirtualMachineList{
				APIVersion: convert.DevAPIVersion,
				Kind:       convert.DevVirtualMachineListKind,
				Items: []convert.DevVirtualMachine{
					{
						APIVersion: convert.DevAPIVersion,
						Kind:       convert.DevVirtualMachineKind,
						Metadata:   convert.DevMetadata{Name: "ws-vm", Project: "demo", Workspace: "dev"},
						Spec:       convert.DevVirtualMachineSpec{VirtualMachine: convert.DevResourceRef{Name: "profile"}},
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == base+"/ws-vm":
			_ = json.NewEncoder(w).Encode(convert.DevVirtualMachine{
				APIVersion: convert.DevAPIVersion,
				Kind:       convert.DevVirtualMachineKind,
				Metadata:   convert.DevMetadata{Name: "ws-vm", Project: "demo", Workspace: "dev"},
				Spec:       convert.DevVirtualMachineSpec{VirtualMachine: convert.DevResourceRef{Name: "profile"}},
			})
		case r.Method == http.MethodPost && r.URL.Path == base+"/ws-vm/action/stop":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	vms := cs.V1alpha1().Workspaces("demo").VirtualMachines("dev")

	list, err := vms.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Metadata.Workspace != "dev" {
		t.Fatalf("list: %+v", list.Items)
	}

	if _, err := vms.Stop(ctx, "ws-vm", gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
}
