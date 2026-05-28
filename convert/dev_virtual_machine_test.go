package convert_test

import (
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func TestDevVirtualMachineRoundTrip(t *testing.T) {
	in := &apiv1.VirtualMachine{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindVirtualMachine},
		Metadata: apiv1.ObjectMeta{
			Name:      "my-vm",
			Project:   "demo",
			Workspace: "dev",
			Labels:    map[string]string{"env": "test"},
		},
		Spec: apiv1.VirtualMachineSpec{
			VirtualMachine: apiv1.ResourceRef{Name: "ubuntu-22", SystemCatalog: true},
			CPUCount:       "4",
			Memory:         "8Gi",
			SecurityGroup:  "default-sg",
			SSHKey:         "my-key",
			VPC:            "tenant-vpc",
			Subnet:         "private-subnet",
			AssignPublicIP: true,
			Image:          "ubuntu-22.04",
			BootDiskSize:   50,
			DNSServers:     []string{"8.8.8.8", "1.1.1.1"},
			Sharing: &apiv1.VirtualMachineSharingSpec{
				ShareMode:  "Custom",
				Workspaces: []string{"shared-ws"},
			},
		},
		Status: apiv1.VirtualMachineStatus{
			Status: "success",
			Output: &apiv1.VirtualMachineOutput{
				PrivateIP: "10.0.0.5",
				PublicIP:  "203.0.113.10",
			},
		},
	}

	wire := convert.ToDevVirtualMachine(in, "demo", "dev")
	if wire.Spec.VirtualMachine.Name != "ubuntu-22" {
		t.Fatalf("wire virtual_machine.name: %q", wire.Spec.VirtualMachine.Name)
	}
	if wire.Spec.CPUCount != "4" {
		t.Fatalf("wire cpu_count: %q", wire.Spec.CPUCount)
	}
	if wire.Spec.SecurityGroup != "default-sg" {
		t.Fatalf("wire security_group: %q", wire.Spec.SecurityGroup)
	}
	if wire.Spec.AssignPublicIP != true {
		t.Fatalf("wire assign_public_ip: %v", wire.Spec.AssignPublicIP)
	}
	if len(wire.Spec.DNSServers) != 2 {
		t.Fatalf("wire dns_servers: %v", wire.Spec.DNSServers)
	}

	wire.Status = convert.DevVirtualMachineStatus{
		Status: "success",
		Output: &convert.DevVirtualMachineOutput{
			PrivateIP: "10.0.0.5",
			PublicIP:  "203.0.113.10",
		},
	}

	out := convert.FromDevVirtualMachine(wire, "dev")
	if out.APIVersion != apiv1.APIVersion {
		t.Fatalf("apiVersion: %q", out.APIVersion)
	}
	if out.Metadata.Workspace != "dev" {
		t.Fatalf("workspace: %q", out.Metadata.Workspace)
	}
	if out.Spec.CPUCount != "4" || out.Spec.SecurityGroup != "default-sg" {
		t.Fatalf("spec round-trip failed: %+v", out.Spec)
	}
	if out.Status.Output == nil || out.Status.Output.PrivateIP != "10.0.0.5" {
		t.Fatalf("status output: %+v", out.Status.Output)
	}
}

func TestVMPathsProjectScope(t *testing.T) {
	collection, item, status, action := convert.VMPaths(convert.VMScope{Project: "demo"})
	if collection() != "/apis/dev.envmgmt.io/v1/projects/demo/virtualmachines" {
		t.Fatalf("collection: %s", collection())
	}
	if item("vm1") != "/apis/dev.envmgmt.io/v1/projects/demo/virtualmachines/vm1" {
		t.Fatalf("item: %s", item("vm1"))
	}
	if status("vm1") != "/apis/dev.envmgmt.io/v1/projects/demo/virtualmachines/vm1/status" {
		t.Fatalf("status: %s", status("vm1"))
	}
	if action("vm1", "start") != "/apis/dev.envmgmt.io/v1/projects/demo/virtualmachines/vm1/action/start" {
		t.Fatalf("action: %s", action("vm1", "start"))
	}
}

func TestVMPathsWorkspaceScope(t *testing.T) {
	collection, item, _, action := convert.VMPaths(convert.VMScope{Project: "demo", Workspace: "dev"})
	if collection() != "/apis/dev.envmgmt.io/v1/projects/demo/workspaces/dev/virtualmachines" {
		t.Fatalf("collection: %s", collection())
	}
	if item("vm1") != "/apis/dev.envmgmt.io/v1/projects/demo/workspaces/dev/virtualmachines/vm1" {
		t.Fatalf("item: %s", item("vm1"))
	}
	if action("vm1", "stop") != "/apis/dev.envmgmt.io/v1/projects/demo/workspaces/dev/virtualmachines/vm1/action/stop" {
		t.Fatalf("action: %s", action("vm1", "stop"))
	}
}
