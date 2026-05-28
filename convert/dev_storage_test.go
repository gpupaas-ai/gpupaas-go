package convert_test

import (
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func TestDevStorageRoundTrip(t *testing.T) {
	in := &apiv1.Storage{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindStorage},
		Metadata: apiv1.ObjectMeta{Name: "my-storage", Project: "demo", Workspace: "dev"},
		Spec: apiv1.StorageSpec{
			Storage: apiv1.ResourceRef{Name: "block-vol", SystemCatalog: true},
			Type:    "standard",
			Size:    "10Gi",
			Sharing: &apiv1.DevSharingSpec{ShareMode: "Custom", Workspaces: []string{"dev"}},
		},
		Status: apiv1.StorageStatus{Status: "success"},
	}

	wire := convert.ToDevStorage(in, "demo", "dev")
	if wire.Spec.Storage.Name != "block-vol" {
		t.Fatalf("wire storage.name: %q", wire.Spec.Storage.Name)
	}
	if wire.Spec.Size != "10Gi" {
		t.Fatalf("wire size: %q", wire.Spec.Size)
	}

	wire.Status = convert.DevStorageStatus{Status: "success"}
	out := convert.FromDevStorage(wire, "dev")
	if out.Spec.Storage.Name != "block-vol" || out.Metadata.Workspace != "dev" {
		t.Fatalf("round-trip failed: %+v", out)
	}
}

func TestStoragePaths(t *testing.T) {
	collection, item := convert.StoragePaths(convert.DevScope{Project: "demo"})
	if collection() != "/apis/dev.envmgmt.io/v1/projects/demo/storages" {
		t.Fatalf("project collection: %s", collection())
	}
	if item("s1") != "/apis/dev.envmgmt.io/v1/projects/demo/storages/s1" {
		t.Fatalf("project item: %s", item("s1"))
	}

	collection, item = convert.StoragePaths(convert.DevScope{Project: "demo", Workspace: "dev"})
	if collection() != "/apis/dev.envmgmt.io/v1/projects/demo/workspaces/dev/storages" {
		t.Fatalf("workspace collection: %s", collection())
	}
}

func TestDevSecurityGroupRoundTrip(t *testing.T) {
	in := &apiv1.SecurityGroup{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindSecurityGroup},
		Metadata: apiv1.ObjectMeta{Name: "default-sg", Project: "demo"},
		Spec: apiv1.SecurityGroupSpec{
			SecurityGroup: apiv1.ResourceRef{Name: "default-sg"},
			IPRules: []apiv1.IpRule{
				{SourceCIDR: "0.0.0.0/0", Application: "ssh", Action: "allow"},
			},
		},
	}

	wire := convert.ToDevSecurityGroup(in, "demo", "")
	if wire.Spec.SecurityGroup.Name != "default-sg" {
		t.Fatalf("wire security_group: %+v", wire.Spec.SecurityGroup)
	}
	if len(wire.Spec.IPRules) != 1 || wire.Spec.IPRules[0].SourceCIDR != "0.0.0.0/0" {
		t.Fatalf("wire ip_rules: %+v", wire.Spec.IPRules)
	}

	out := convert.FromDevSecurityGroup(wire, "")
	if out.Spec.IPRules[0].SourceCIDR != "0.0.0.0/0" {
		t.Fatalf("round-trip ip rule: %+v", out.Spec.IPRules)
	}
}

func TestDevSshKeyRoundTrip(t *testing.T) {
	in := &apiv1.SshKey{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindSshKey},
		Metadata: apiv1.ObjectMeta{Name: "my-key", Project: "demo", Workspace: "dev"},
		Spec: apiv1.SshKeySpec{
			SSHKey:    apiv1.ResourceRef{Name: "my-key"},
			PublicKey: "ssh-rsa AAAA...",
		},
	}

	wire := convert.ToDevSshKey(in, "demo", "dev")
	if wire.Spec.SSHKey.Name != "my-key" {
		t.Fatalf("wire ssh_key: %+v", wire.Spec.SSHKey)
	}
	if wire.Spec.PublicKey != "ssh-rsa AAAA..." {
		t.Fatalf("wire public_key: %q", wire.Spec.PublicKey)
	}

	out := convert.FromDevSshKey(wire, "dev")
	if out.Spec.PublicKey != "ssh-rsa AAAA..." {
		t.Fatalf("round-trip public key: %q", out.Spec.PublicKey)
	}
}

func TestSshKeyPathsWorkspaceScope(t *testing.T) {
	collection, item := convert.SshKeyPaths(convert.DevScope{Project: "demo", Workspace: "dev"})
	if item("key1") != "/apis/dev.envmgmt.io/v1/projects/demo/workspaces/dev/sshkeys/key1" {
		t.Fatalf("item: %s", item("key1"))
	}
	if collection() == "" {
		t.Fatal("expected collection path")
	}
}
