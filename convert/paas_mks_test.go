package convert

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

func loadFixture[T any](t *testing.T, name string) T {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return out
}

func TestMKSClusterPaths(t *testing.T) {
	coll, item := MKSClusterPaths(DevScope{Project: "p"})
	if coll() != "/apis/paas.k8smgmt.io/v3/projects/p/clusters" || item("c") != "/apis/paas.k8smgmt.io/v3/projects/p/clusters/c" {
		t.Fatalf("project paths: %s %s", coll(), item("c"))
	}
	coll, item = MKSClusterPaths(DevScope{Project: "p", Workspace: "w s"})
	if coll() != "/apis/paas.k8smgmt.io/v3/projects/p/workspaces/w%20s/clusters" || item("c") != "/apis/paas.k8smgmt.io/v3/projects/p/workspaces/w%20s/clusters/c" {
		t.Fatalf("workspace paths: %s %s", coll(), item("c"))
	}
}

func TestFromMKSClusterReadsLiveResponse(t *testing.T) {
	w := loadFixture[MKSClusterObject](t, "mks_cluster.json")
	c := FromMKSCluster(&w, "system-catalog", "data-test")
	if c.Metadata.Name != "demo-cluster" || c.Metadata.Project != "system-catalog" || c.Metadata.Workspace != "data-test" {
		t.Fatalf("metadata: %#v", c.Metadata)
	}
	s := c.Spec
	if s.Profile == nil || s.Profile.Name != "mks-oneclick-1" || s.KubernetesVersion != "v1.36.4" || s.Location != "default-dc" {
		t.Fatalf("spec basics: %#v", s)
	}
	if s.Networking == nil || s.Networking.CNI == nil || s.Networking.CNI.Name != "Calico" || s.Networking.PodSubnet == "" {
		t.Fatalf("networking: %#v", s.Networking)
	}
	cp := s.ControlPlane
	if cp == nil || cp.SKUName != "bmaas-test-1" || cp.SKUType != "BMAAS" || cp.OSImage != "ubuntu-26" || cp.Scaling == nil || cp.Scaling.Count != 1 || cp.Scaling.Mode != "manual" {
		t.Fatalf("control plane: %#v", cp)
	}
	if cp.Shape == nil {
		t.Fatal("derived shape was not kept")
	}
	if c.Status.Phase() != "running" || !c.Status.Running() || c.Status.Reason != "Cluster is ready" || c.Status.Output.ClusterIDEdgesrv != "demo-cluster" {
		t.Fatalf("status: %#v", c.Status)
	}
	if c.Version != 5 {
		t.Fatalf("version = %d", c.Version)
	}
	if c.Spec.Billing == nil || c.Spec.Billing.Variables["ContractTerm"] != "0" || c.Spec.Sharing == nil || c.Spec.Sharing.ShareMode != "None" {
		t.Fatalf("billing/sharing: %#v %#v", c.Spec.Billing, c.Spec.Sharing)
	}
}

func TestToMKSClusterRoundTripsAReadSpec(t *testing.T) {
	// Applying a spec read from the API must send every field back, derived ones included.
	w := loadFixture[MKSClusterObject](t, "mks_cluster.json")
	c := FromMKSCluster(&w, "system-catalog", "data-test")
	c.Spec.KubernetesVersion = "v1.37.0"
	out := ToMKSCluster(c, "system-catalog", "data-test")
	if out.Status != nil {
		t.Fatal("status must not be sent")
	}
	if out.Version != 5 {
		t.Fatalf("version = %d, want the read revision echoed back", out.Version)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"kubernetesVersion":"v1.37.0"`, `"skuDisplayName":"custom"`, `"shape":{"gpu"`, `"workerPools":[]`, `"apiVersion":"paas.k8smgmt.io/v3"`, `"kind":"Cluster"`, `"workspace":"data-test"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("missing %s in %s", want, raw)
		}
	}
}

func TestFromMKSClusterProfileReadsLiveResponse(t *testing.T) {
	w := loadFixture[MKSClusterProfileObject](t, "mks_clusterprofile.json")
	p := FromMKSClusterProfile(&w, "system-catalog")
	if p.Metadata.Name != "mks-oneclick-1" || p.Metadata.Project != "system-catalog" {
		t.Fatalf("metadata: %#v", p.Metadata)
	}
	s := p.Spec
	if s.ControlPlane == nil || s.ControlPlane.DefaultSKUName != "bmaas-test-1" || len(s.ControlPlane.SKUTypes) != 1 || s.ControlPlane.SKUTypes[0] != "BMAAS" {
		t.Fatalf("control plane: %#v", s.ControlPlane)
	}
	if s.Workers == nil || len(s.Workers.SKUNames) == 0 || len(s.Datacenters) != 1 || s.Datacenters[0] != "default-dc" {
		t.Fatalf("workers/datacenters: %#v %#v", s.Workers, s.Datacenters)
	}
	if s.DefaultCNI == nil || s.DefaultCNI.Version != "3.32.2" || len(s.SupportedKubernetesVersions) != 1 || s.DefaultKubernetesVersion != "1.36" {
		t.Fatalf("cni/versions: %#v", s)
	}
}

func TestFromMKSNodeListReadsLiveResponse(t *testing.T) {
	w := loadFixture[MKSNodeObjectList](t, "mks_clusternodes.json")
	l := FromMKSNodeList(&w, "system-catalog")
	if len(l.Items) != 1 {
		t.Fatalf("items = %d", len(l.Items))
	}
	n := l.Items[0]
	if n.NodePool() != "master" || n.Spec.KubeVersion != "v1.36.4" || n.Metadata.Labels["cluster_name"] != "demo-cluster" || n.Metadata.Project != "system-catalog" {
		t.Fatalf("node: %#v", n)
	}
}

var _ = apiv1.KindMKSCluster
