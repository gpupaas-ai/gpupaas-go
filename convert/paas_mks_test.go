package convert_test

import (
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func TestPaaSMKSClusterRoundTrip(t *testing.T) {
	ha := true
	dedicated := false
	tls := true
	publicIP := true
	in := &apiv1.MKSCluster{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSCluster},
		Metadata: apiv1.ObjectMeta{
			Name:    "c1",
			Project: "demo",
			Labels:  map[string]string{"team": "ops"},
		},
		Spec: apiv1.MKSClusterSpec{
			KubernetesVersion:     "1.31",
			PlatformVersion:       "p1",
			CNI:                   "calico",
			CNIVersion:            "v3",
			OS:                    "ubuntu22.04",
			HAEnabled:             &ha,
			DedicatedControlPlane: &dedicated,
			Location:              "dc1",
			Tags:                  map[string]string{"env": "prod"},
			Blueprint:             &apiv1.MKSBlueprint{Name: "minimal", Version: "v1"},
			Networking: &apiv1.MKSNetworking{
				VPC:            "vpc-1",
				Subnet:         "subnet-1",
				PodCIDR:        "192.168.0.0/16",
				ServiceCIDR:    "10.96.0.0/12",
				IPFamily:       "IPv4",
				SecurityGroups: []string{"sg-1", "sg-2"},
			},
			Proxy: &apiv1.MKSProxy{
				HTTPProxy:    "http://proxy:8080",
				HTTPSProxy:   "https://proxy:8443",
				NoProxy:      "localhost",
				ProxyRootCA:  "ca-data",
				TLSTerminate: &tls,
			},
			Storage: &apiv1.MKSStorage{
				Block:               &apiv1.MKSStorageBackend{Type: "ceph", AccessMode: "RWO", ReclaimPolicy: "Delete", Config: map[string]string{"pool": "rbd"}},
				DefaultStorageClass: "block",
			},
			ControlPlaneNodeGroup: &apiv1.MKSNodeGroup{
				ID:           "cp",
				SKU:          "large",
				NodeCount:    3,
				MinNodes:     1,
				MaxNodes:     5,
				DesiredNodes: 3,
				PublicIP:     &publicIP,
				NodeLabels:   map[string]string{"role": "cp"},
				KubeletConfig: &apiv1.MKSKubeletConfig{
					KeyValue: map[string]string{"maxPods": "110"},
					YAML:     "foo: bar",
				},
			},
			WorkerNodeGroups: []apiv1.MKSNodeGroup{
				{ID: "wng-1", SKU: "medium", NodeCount: 2},
			},
			Nodes: []apiv1.MKSNodeSpec{
				{
					Hostname:        "master-1",
					Roles:           []string{"master", "worker"},
					SSHUserName:     "ubuntu",
					SSHKey:          "key-data",
					SSHPort:         22,
					PrivateIP:       "10.0.0.10",
					Arch:            "amd64",
					OperatingSystem: "ubuntu22.04",
				},
			},
		},
		Status: apiv1.MKSClusterStatus{
			Condition: "MKS_CLUSTER_STATUS_RUNNING",
		},
	}

	wire := convert.ToPaaSMKSCluster(in, "demo")
	if wire == nil {
		t.Fatal("wire is nil")
	}
	if wire.APIVersion != convert.PaaSMKSAPIVersion || wire.Kind != convert.PaaSMKSClusterKind {
		t.Fatalf("wire envelope: %s/%s", wire.APIVersion, wire.Kind)
	}
	if wire.Metadata.Name != "c1" || wire.Metadata.Project != "demo" {
		t.Fatalf("wire metadata: %+v", wire.Metadata)
	}
	if wire.Spec.KubernetesVersion != "1.31" || wire.Spec.CNI != "calico" {
		t.Fatalf("wire spec scalars: %+v", wire.Spec)
	}
	if wire.Spec.HAEnabled == nil || !*wire.Spec.HAEnabled {
		t.Fatalf("wire haEnabled: %+v", wire.Spec.HAEnabled)
	}
	if wire.Spec.Blueprint == nil || wire.Spec.Blueprint.Name != "minimal" {
		t.Fatalf("wire blueprint: %+v", wire.Spec.Blueprint)
	}
	if wire.Spec.Networking == nil || len(wire.Spec.Networking.SecurityGroups) != 2 {
		t.Fatalf("wire networking: %+v", wire.Spec.Networking)
	}
	if wire.Spec.Proxy == nil || wire.Spec.Proxy.TLSTerminate == nil || !*wire.Spec.Proxy.TLSTerminate {
		t.Fatalf("wire proxy: %+v", wire.Spec.Proxy)
	}
	if wire.Spec.Storage == nil || wire.Spec.Storage.Block == nil || wire.Spec.Storage.Block.Type != "ceph" {
		t.Fatalf("wire storage: %+v", wire.Spec.Storage)
	}
	if wire.Spec.ControlPlaneNodeGroup == nil || wire.Spec.ControlPlaneNodeGroup.KubeletConfig == nil {
		t.Fatalf("wire control plane node group: %+v", wire.Spec.ControlPlaneNodeGroup)
	}
	if len(wire.Spec.WorkerNodeGroups) != 1 || wire.Spec.WorkerNodeGroups[0].ID != "wng-1" {
		t.Fatalf("wire worker node groups: %+v", wire.Spec.WorkerNodeGroups)
	}
	if len(wire.Spec.Nodes) != 1 || len(wire.Spec.Nodes[0].Roles) != 2 {
		t.Fatalf("wire nodes: %+v", wire.Spec.Nodes)
	}

	// Simulate a server response with status, then round-trip back.
	wire.Status = convert.PaaSMKSClusterStatus{
		Condition:       "MKS_CLUSTER_STATUS_RUNNING",
		ConditionReason: "all good",
		Output:          &convert.PaaSMKSClusterOutput{APIServerEndpoint: "https://api:6443", ClusterIDEdgesrv: "edge-1"},
		Action:          "none",
	}
	out := convert.FromPaaSMKSCluster(wire)
	if out == nil {
		t.Fatal("out is nil")
	}
	if out.Metadata.Name != "c1" || out.Metadata.Project != "demo" {
		t.Fatalf("round-trip metadata: %+v", out.Metadata)
	}
	if out.Spec.KubernetesVersion != "1.31" || out.Spec.Networking.PodCIDR != "192.168.0.0/16" {
		t.Fatalf("round-trip spec: %+v", out.Spec)
	}
	if out.Spec.HAEnabled == nil || !*out.Spec.HAEnabled {
		t.Fatalf("round-trip haEnabled: %+v", out.Spec.HAEnabled)
	}
	if out.Spec.Storage == nil || out.Spec.Storage.Block.Config["pool"] != "rbd" {
		t.Fatalf("round-trip storage: %+v", out.Spec.Storage)
	}
	if out.Status.Condition != "MKS_CLUSTER_STATUS_RUNNING" || out.Status.Output == nil || out.Status.Output.APIServerEndpoint != "https://api:6443" {
		t.Fatalf("round-trip status: %+v", out.Status)
	}
}

func TestPaaSMKSClusterListConvert(t *testing.T) {
	wire := &convert.PaaSMKSClusterList{
		APIVersion: convert.PaaSMKSAPIVersion,
		Kind:       convert.PaaSMKSClusterListKind,
		Items: []convert.PaaSMKSCluster{
			{Metadata: convert.PaaSMKSMetadata{Name: "c1", Project: "demo"}},
			{Metadata: convert.PaaSMKSMetadata{Name: "c2", Project: "demo"}},
		},
	}
	out := convert.FromPaaSMKSClusterList(wire)
	if out == nil || len(out.Items) != 2 {
		t.Fatalf("list items: %+v", out)
	}
	if out.Kind != apiv1.KindMKSCluster+"List" {
		t.Fatalf("list kind: %s", out.Kind)
	}
	if out.Items[0].Metadata.Name != "c1" || out.Items[1].Metadata.Name != "c2" {
		t.Fatalf("list names: %+v", out.Items)
	}

	empty := convert.FromPaaSMKSClusterList(nil)
	if empty == nil || empty.Kind != apiv1.KindMKSCluster+"List" {
		t.Fatalf("empty: %+v", empty)
	}
}

func TestPaaSMKSNodeConvert(t *testing.T) {
	wire := &convert.PaaSMKSNode{
		APIVersion: convert.PaaSMKSAPIVersion,
		Kind:       convert.PaaSMKSNodeKind,
		Metadata:   convert.PaaSMKSMetadata{Name: "master-1", Project: "demo"},
		Spec:       convert.PaaSMKSNodeSpec{Hostname: "master-1", Roles: []string{"master"}, PrivateIP: "10.0.0.10"},
		Status:     convert.PaaSMKSNodeStatus{Phase: "MKS_NODE_PHASE_RUNNING", Reason: "ok"},
	}
	out := convert.FromPaaSMKSNode(wire)
	if out == nil || out.Metadata.Name != "master-1" {
		t.Fatalf("node: %+v", out)
	}
	if out.Spec.PrivateIP != "10.0.0.10" || len(out.Spec.Roles) != 1 {
		t.Fatalf("node spec: %+v", out.Spec)
	}
	if out.Status.Phase != "MKS_NODE_PHASE_RUNNING" {
		t.Fatalf("node status: %+v", out.Status)
	}

	list := convert.FromPaaSMKSNodeList(&convert.PaaSMKSNodeList{
		Items: []convert.PaaSMKSNode{*wire},
	})
	if list == nil || len(list.Items) != 1 || list.Kind != apiv1.KindMKSNode+"List" {
		t.Fatalf("node list: %+v", list)
	}
}

func TestPaaSMKSWorkerNodeGroupRoundTrip(t *testing.T) {
	in := &apiv1.MKSWorkerNodeGroup{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSWorkerNodeGroup},
		Metadata: apiv1.ObjectMeta{Name: "wng-1", Project: "demo"},
		Spec: apiv1.MKSWorkerNodeGroupSpec{
			ClusterName: "c1",
			NodeGroup:   &apiv1.MKSNodeGroup{ID: "wng-1", SKU: "medium", NodeCount: 2},
		},
	}
	wire := convert.ToPaaSMKSWorkerNodeGroup(in, "demo")
	if wire == nil || wire.Spec.ClusterName != "c1" || wire.Spec.NodeGroup == nil {
		t.Fatalf("wire: %+v", wire)
	}
	out := convert.FromPaaSMKSWorkerNodeGroup(wire)
	if out == nil || out.Spec.ClusterName != "c1" || out.Spec.NodeGroup.SKU != "medium" {
		t.Fatalf("round-trip: %+v", out)
	}

	list := convert.FromPaaSMKSWorkerNodeGroupList(&convert.PaaSMKSWorkerNodeGroupList{
		Items: []convert.PaaSMKSWorkerNodeGroup{*wire},
	})
	if list == nil || len(list.Items) != 1 || list.Kind != apiv1.KindMKSWorkerNodeGroup+"List" {
		t.Fatalf("wng list: %+v", list)
	}
}

func TestPaaSMKSAuditEventConvert(t *testing.T) {
	wire := &convert.PaaSMKSAuditEvent{
		ID:           "evt-1",
		ResourceType: "MKSCluster",
		ResourceName: "c1",
		Action:       "create",
		Actor:        "alice",
		Message:      "created cluster",
		Timestamp:    1234567890,
		Metadata:     convert.PaaSMKSMetadata{Name: "c1", Project: "demo"},
	}
	out := convert.FromPaaSMKSAuditEvent(wire)
	if out == nil || out.ID != "evt-1" || out.Action != "create" || out.Timestamp != 1234567890 {
		t.Fatalf("audit event: %+v", out)
	}

	list := convert.FromPaaSMKSAuditEventList(&convert.PaaSMKSAuditEventList{
		Items: []convert.PaaSMKSAuditEvent{*wire},
	})
	if list == nil || len(list.Items) != 1 || list.Items[0].Actor != "alice" {
		t.Fatalf("audit list: %+v", list)
	}

	if convert.FromPaaSMKSAuditEvent(nil) != nil {
		t.Fatal("expected nil for nil audit event")
	}
}

func TestPaaSMKSSubActionRequestConvert(t *testing.T) {
	upgrade := convert.ToPaaSMKSUpgradeRequest(&apiv1.MKSUpgradeRequest{K8sVersion: "1.32", PlatformVersion: "p2"})
	if upgrade == nil || upgrade.K8sVersion != "1.32" || upgrade.PlatformVersion != "p2" {
		t.Fatalf("upgrade: %+v", upgrade)
	}

	desired := int32(5)
	scale := convert.ToPaaSMKSScaleNodeGroupRequest(&apiv1.MKSScaleNodeGroupRequest{
		NodeGroupName: "wng-1",
		DesiredCount:  &desired,
	})
	if scale == nil || scale.NodeGroupName != "wng-1" || scale.DesiredCount == nil || *scale.DesiredCount != 5 {
		t.Fatalf("scale: %+v", scale)
	}

	add := convert.ToPaaSMKSAddNodeGroupRequest(&apiv1.MKSNodeGroup{ID: "wng-2", SKU: "small"})
	if add == nil || add.NodeGroup == nil || add.NodeGroup.ID != "wng-2" {
		t.Fatalf("add: %+v", add)
	}

	force := true
	drain := convert.ToPaaSMKSDrainRequest(&apiv1.MKSDrainRequest{Force: &force})
	if drain == nil || drain.Force == nil || !*drain.Force {
		t.Fatalf("drain: %+v", drain)
	}

	if convert.ToPaaSMKSUpgradeRequest(nil) != nil {
		t.Fatal("expected nil upgrade for nil input")
	}
}

func TestMKSPaths(t *testing.T) {
	collection, item, subroute := convert.MKSClusterPaths(convert.PaaSMKSScope{Project: "demo"})
	if got, want := collection(), "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters"; got != want {
		t.Fatalf("cluster collection: %s", got)
	}
	if got, want := item("c1"), "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1"; got != want {
		t.Fatalf("cluster item: %s", got)
	}
	if got, want := subroute("c1", "upgrade"), "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/upgrade"; got != want {
		t.Fatalf("cluster subroute: %s", got)
	}

	nodeColl, nodeItem, nodeSub := convert.MKSNodePaths(convert.PaaSMKSClusterScope{Project: "demo", Cluster: "c1"})
	if got, want := nodeColl(), "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/mksnodes"; got != want {
		t.Fatalf("node collection: %s", got)
	}
	if got, want := nodeItem("n1"), "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/mksnodes/n1"; got != want {
		t.Fatalf("node item: %s", got)
	}
	if got, want := nodeSub("n1", "drain"), "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/mksnodes/n1/drain"; got != want {
		t.Fatalf("node subroute: %s", got)
	}

	wngColl, wngItem := convert.MKSWorkerNodeGroupPaths(convert.PaaSMKSClusterScope{Project: "demo", Cluster: "c1"})
	if got, want := wngColl(), "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/workernodegroups"; got != want {
		t.Fatalf("wng collection: %s", got)
	}
	if got, want := wngItem("wng-1"), "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/workernodegroups/wng-1"; got != want {
		t.Fatalf("wng item: %s", got)
	}

	aeColl, aeItem := convert.MKSAuditEventPaths(convert.PaaSMKSClusterScope{Project: "demo", Cluster: "c1"})
	if got, want := aeColl(), "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/auditevents"; got != want {
		t.Fatalf("audit collection: %s", got)
	}
	if got, want := aeItem("evt-1"), "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/auditevents/evt-1"; got != want {
		t.Fatalf("audit item: %s", got)
	}
}
