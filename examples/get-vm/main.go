// Get a virtual machine by name.
//
//	go run ./examples/get-vm -memory -project demo -name example-vm
//	go run ./examples/get-vm -project demo -workspace dev -name example-vm
package main

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/examples/vmcommon"
)

func main() {
	cfg := vmcommon.ParseFlags()
	ctx := context.Background()

	vms, memClient, err := vmcommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== get virtual machine %q (%s) ===\n", cfg.Name, vmcommon.ScopeLabel(cfg.Workspace))

	if memClient != nil {
		vm := vmcommon.SampleVM(cfg.Project, cfg.Workspace, cfg.Name)
		vmcommon.ApplyMemory(ctx, memClient, vm)
		got := vmcommon.GetMemory(ctx, memClient, cfg.Project, cfg.Workspace, cfg.Name)
		vmcommon.PrintYAML("virtualmachine", got)
		return
	}

	got, err := vms.Get(ctx, cfg.Name, gpupaas.GetOptions{})
	if err != nil {
		panic(err)
	}
	vmcommon.PrintYAML("virtualmachine", got)
}
