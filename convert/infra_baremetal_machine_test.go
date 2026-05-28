package convert_test

import (
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func TestInfraBaremetalMachineRoundTrip(t *testing.T) {
	online := true
	rotational := false
	in := &apiv1.BaremetalMachine{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindBaremetalMachine},
		Metadata: apiv1.ObjectMeta{
			Name:    "bm1",
			Project: "demo",
			Labels:  map[string]string{"team": "ops"},
		},
		Spec: apiv1.BaremetalMachineSpec{
			Architecture:             "x86_64",
			BaremetalProvisionerName: "rack1-provisioner",
			BootMode:                 "UEFI",
			Datacenter:               "dc1",
			DeviceID:                 "dev-123",
			Hostname:                 "bm1.example.com",
			MACAddress:               "aa:bb:cc:dd:ee:ff",
			SSHKey:                   "my-key",
			UserData:                 "#cloud-config\nhostname: bm1\n",
			Online:                   &online,
			Image: &apiv1.BaremetalImage{
				Checksum:     "abc",
				ChecksumType: "sha256",
				Format:       "qcow2",
				URL:          "http://images/example.qcow2",
			},
			RootDeviceHints: &apiv1.BaremetalRootDeviceHints{
				DeviceName: "/dev/sda",
				Rotational: &rotational,
			},
			Raid: &apiv1.BaremetalRaid{
				HardwareRAIDVolumes: []apiv1.BaremetalHardwareRAIDVolumes{
					{
						Controller:            "ctrl-1",
						Level:                 "1",
						Name:                  "root",
						NumberOfPhysicalDisks: 2,
						PhysicalDisks:         []string{"disk1", "disk2"},
						SizeGibibytes:         100,
					},
				},
				SoftwareRAIDVolumes: []apiv1.BaremetalSoftwareRAIDVolumes{
					{
						Level: "1+0",
						PhysicalDisks: []apiv1.BaremetalRootDeviceHints{
							{DeviceName: "/dev/sdb"},
							{DeviceName: "/dev/sdc"},
						},
						SizeGibibytes: 200,
					},
				},
			},
		},
		Status: apiv1.BaremetalMachineStatus{
			Conditions: []apiv1.BaremetalMachineCondition{
				{Type: "Provisioned", Status: "Success"},
			},
		},
	}

	wire := convert.ToInfraBaremetalMachine(in, "demo")
	if wire == nil {
		t.Fatal("wire is nil")
	}
	if wire.APIVersion != convert.InfraAPIVersion || wire.Kind != convert.InfraBaremetalMachineKind {
		t.Fatalf("wire envelope: %s/%s", wire.APIVersion, wire.Kind)
	}
	if wire.Metadata.Name != "bm1" || wire.Metadata.Project != "demo" {
		t.Fatalf("wire metadata: %+v", wire.Metadata)
	}
	if wire.Spec.Architecture != "x86_64" || wire.Spec.Hostname != "bm1.example.com" {
		t.Fatalf("wire spec scalars: %+v", wire.Spec)
	}
	if wire.Spec.Online == nil || *wire.Spec.Online != true {
		t.Fatalf("wire online: %+v", wire.Spec.Online)
	}
	if wire.Spec.Image == nil || wire.Spec.Image.URL != "http://images/example.qcow2" {
		t.Fatalf("wire image: %+v", wire.Spec.Image)
	}
	if wire.Spec.RootDeviceHints == nil || wire.Spec.RootDeviceHints.Rotational == nil || *wire.Spec.RootDeviceHints.Rotational {
		t.Fatalf("wire root device hints rotational: %+v", wire.Spec.RootDeviceHints)
	}
	if wire.Spec.Raid == nil || len(wire.Spec.Raid.HardwareRAIDVolumes) != 1 || wire.Spec.Raid.HardwareRAIDVolumes[0].Level != "1" {
		t.Fatalf("wire hardware raid: %+v", wire.Spec.Raid)
	}
	if len(wire.Spec.Raid.SoftwareRAIDVolumes) != 1 || len(wire.Spec.Raid.SoftwareRAIDVolumes[0].PhysicalDisks) != 2 {
		t.Fatalf("wire software raid: %+v", wire.Spec.Raid.SoftwareRAIDVolumes)
	}
	// Status is observed-only and should not appear on the apply payload.
	if len(wire.Status.Conditions) != 0 {
		t.Fatalf("wire status leaked: %+v", wire.Status)
	}

	// Add a status to simulate a server response, then round-trip back.
	wire.Status = convert.InfraBaremetalMachineStatus{
		Conditions: []convert.InfraBaremetalMachineCondition{{Type: "Provisioned", Status: "Success"}},
	}
	out := convert.FromInfraBaremetalMachine(wire)
	if out == nil {
		t.Fatal("out is nil")
	}
	if out.Metadata.Name != "bm1" || out.Metadata.Project != "demo" {
		t.Fatalf("round-trip metadata: %+v", out.Metadata)
	}
	if out.Spec.Hostname != "bm1.example.com" || out.Spec.Image.URL != "http://images/example.qcow2" {
		t.Fatalf("round-trip spec: %+v", out.Spec)
	}
	if out.Spec.Online == nil || !*out.Spec.Online {
		t.Fatalf("round-trip online: %+v", out.Spec.Online)
	}
	if out.Spec.RootDeviceHints == nil || out.Spec.RootDeviceHints.Rotational == nil || *out.Spec.RootDeviceHints.Rotational {
		t.Fatalf("round-trip root device hints: %+v", out.Spec.RootDeviceHints)
	}
	if len(out.Spec.Raid.SoftwareRAIDVolumes[0].PhysicalDisks) != 2 {
		t.Fatalf("round-trip software raid disks: %+v", out.Spec.Raid.SoftwareRAIDVolumes)
	}
	if len(out.Status.Conditions) != 1 || out.Status.Conditions[0].Type != "Provisioned" {
		t.Fatalf("round-trip status: %+v", out.Status)
	}
}

func TestInfraBaremetalMachineListConvert(t *testing.T) {
	wire := &convert.InfraBaremetalMachineList{
		APIVersion: convert.InfraAPIVersion,
		Kind:       convert.InfraBaremetalMachineListKind,
		Items: []convert.InfraBaremetalMachine{
			{
				APIVersion: convert.InfraAPIVersion,
				Kind:       convert.InfraBaremetalMachineKind,
				Metadata:   convert.InfraMetadata{Name: "bm1", Project: "demo"},
			},
			{
				APIVersion: convert.InfraAPIVersion,
				Kind:       convert.InfraBaremetalMachineKind,
				Metadata:   convert.InfraMetadata{Name: "bm2", Project: "demo"},
			},
		},
	}
	out := convert.FromInfraBaremetalMachineList(wire)
	if out == nil || len(out.Items) != 2 {
		t.Fatalf("list items: %+v", out)
	}
	if out.Kind != apiv1.KindBaremetalMachine+"List" {
		t.Fatalf("list kind: %s", out.Kind)
	}
	if out.Items[0].Metadata.Name != "bm1" || out.Items[1].Metadata.Name != "bm2" {
		t.Fatalf("list names: %+v", out.Items)
	}

	// Nil wire returns an empty list with the right kind.
	empty := convert.FromInfraBaremetalMachineList(nil)
	if empty == nil || empty.Kind != apiv1.KindBaremetalMachine+"List" {
		t.Fatalf("empty: %+v", empty)
	}
}

func TestInfraBaremetalMachineInfoConvert(t *testing.T) {
	wire := &convert.InfraBaremetalMachineInfo{
		Data: convert.InfraBaremetalMachineData{
			Fields: map[string]interface{}{
				"hardware": map[string]interface{}{"cpus": 96},
				"network":  []interface{}{"eth0", "eth1"},
			},
		},
	}
	out := convert.FromInfraBaremetalMachineInfo(wire)
	if out == nil {
		t.Fatal("info nil")
	}
	if out.Data.Fields["hardware"] == nil || out.Data.Fields["network"] == nil {
		t.Fatalf("info fields: %+v", out.Data.Fields)
	}

	empty := convert.FromInfraBaremetalMachineInfo(nil)
	if empty == nil {
		t.Fatal("expected empty info, got nil")
	}
}

func TestInfraBaremetalConsoleSessionConvert(t *testing.T) {
	req := convert.ToInfraBaremetalConsoleSessionRequest(&apiv1.BaremetalConsoleSessionRequest{
		ComputeID: "compute-42",
	})
	if req == nil || req.ComputeID != "compute-42" {
		t.Fatalf("request: %+v", req)
	}

	wire := &convert.InfraBaremetalConsoleSession{
		AgentSessionID: "agent",
		ConsoleURL:     "wss://example",
		SessionID:      "session",
	}
	out := convert.FromInfraBaremetalConsoleSession(wire)
	if out == nil || out.SessionID != "session" || out.AgentSessionID != "agent" || out.ConsoleURL != "wss://example" {
		t.Fatalf("session: %+v", out)
	}
}

func TestBaremetalMachinePaths(t *testing.T) {
	collection, item, subroute := convert.BaremetalMachinePaths(convert.InfraProjectScope{Project: "demo"})
	if got, want := collection(), "/apis/infra.k8smgmt.io/v3/projects/demo/baremetalmachines"; got != want {
		t.Fatalf("collection: %s", got)
	}
	if got, want := item("bm1"), "/apis/infra.k8smgmt.io/v3/projects/demo/baremetalmachines/bm1"; got != want {
		t.Fatalf("item: %s", got)
	}
	if got, want := subroute("bm1", "powerOn"), "/apis/infra.k8smgmt.io/v3/projects/demo/baremetalmachines/bm1/powerOn"; got != want {
		t.Fatalf("subroute: %s", got)
	}
}

func TestBaremetalMachineImageRoundTripNil(t *testing.T) {
	if got := convert.ToInfraBaremetalImage(nil); got != nil {
		t.Fatalf("expected nil image, got %+v", got)
	}
}
