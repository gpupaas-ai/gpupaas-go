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

func TestStorageClientProjectScope(t *testing.T) {
	const base = "/apis/dev.envmgmt.io/v1/projects/demo/storages"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(convert.DevStorageList{
				APIVersion: convert.DevAPIVersion,
				Kind:       convert.DevStorageListKind,
				Items: []convert.DevStorage{
					{
						APIVersion: convert.DevAPIVersion,
						Kind:       convert.DevStorageKind,
						Metadata:   convert.DevMetadata{Name: "vol1", Project: "demo"},
						Spec:       convert.DevStorageSpec{Storage: convert.DevResourceRef{Name: "vol1"}, Size: "10Gi"},
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == base+"/vol1":
			_ = json.NewEncoder(w).Encode(convert.DevStorage{
				APIVersion: convert.DevAPIVersion,
				Kind:       convert.DevStorageKind,
				Metadata:   convert.DevMetadata{Name: "vol1", Project: "demo"},
				Spec:       convert.DevStorageSpec{Storage: convert.DevResourceRef{Name: "vol1"}, Size: "10Gi"},
			})
		case r.Method == http.MethodPost && r.URL.Path == base:
			var in convert.DevStorage
			_ = json.NewDecoder(r.Body).Decode(&in)
			in.Status = convert.DevStorageStatus{Status: "pending"}
			_ = json.NewEncoder(w).Encode(in)
		case r.Method == http.MethodDelete && r.URL.Path == base+"/vol1":
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
	st := cs.V1alpha1().Storages("demo")

	list, err := st.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Spec.Size != "10Gi" {
		t.Fatalf("list: %+v", list.Items)
	}

	got, err := st.Get(ctx, "vol1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Name != "vol1" {
		t.Fatalf("get: %+v", got.Metadata)
	}

	created, err := st.Create(ctx, &apiv1.Storage{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindStorage},
		Metadata: apiv1.ObjectMeta{Name: "vol-new", Project: "demo"},
		Spec:     apiv1.StorageSpec{Storage: apiv1.ResourceRef{Name: "vol-new"}, Size: "20Gi"},
	}, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status.Status != "pending" {
		t.Fatalf("create status: %+v", created.Status)
	}

	if err := st.Delete(ctx, "vol1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestSecurityGroupClientWorkspaceScope(t *testing.T) {
	const base = "/apis/dev.envmgmt.io/v1/projects/demo/workspaces/dev/securitygroups"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(convert.DevSecurityGroupList{
				APIVersion: convert.DevAPIVersion,
				Kind:       convert.DevSecurityGroupListKind,
				Items: []convert.DevSecurityGroup{
					{
						APIVersion: convert.DevAPIVersion,
						Kind:       convert.DevSecurityGroupKind,
						Metadata:   convert.DevMetadata{Name: "default-sg", Project: "demo", Workspace: "dev"},
						Spec:       convert.DevSecurityGroupSpec{SecurityGroup: convert.DevResourceRef{Name: "default-sg"}},
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == base+"/default-sg":
			_ = json.NewEncoder(w).Encode(convert.DevSecurityGroup{
				APIVersion: convert.DevAPIVersion,
				Kind:       convert.DevSecurityGroupKind,
				Metadata:   convert.DevMetadata{Name: "default-sg", Project: "demo", Workspace: "dev"},
				Spec:       convert.DevSecurityGroupSpec{SecurityGroup: convert.DevResourceRef{Name: "default-sg"}},
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
	sg := cs.V1alpha1().Workspaces("demo").SecurityGroups("dev")

	list, err := sg.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Metadata.Workspace != "dev" {
		t.Fatalf("list: %+v", list.Items)
	}

	got, err := sg.Get(ctx, "default-sg", gpupaas.GetOptions{})
	if err != nil || got.Spec.SecurityGroup.Name != "default-sg" {
		t.Fatalf("get: %+v err=%v", got, err)
	}
}

func TestSshKeyClientProjectScope(t *testing.T) {
	const base = "/apis/dev.envmgmt.io/v1/projects/demo/sshkeys"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == base:
			var in convert.DevSshKey
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in.Spec.SSHKey.Name == "" {
				t.Fatal("expected ssh_key in wire body")
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
	keys := cs.V1alpha1().SshKeys("demo")

	_, err = keys.Create(ctx, &apiv1.SshKey{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindSshKey},
		Metadata: apiv1.ObjectMeta{Name: "my-key", Project: "demo"},
		Spec: apiv1.SshKeySpec{
			SSHKey:    apiv1.ResourceRef{Name: "my-key"},
			PublicKey: "ssh-rsa AAAA",
		},
	}, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
}
