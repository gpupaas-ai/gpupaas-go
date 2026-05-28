// Create (apply) a virtual machine.
//
// Project-scoped create:
//
//	go run ./examples/create-vm -memory -project demo -name example-vm
//
// Workspace-scoped create:
//
//	go run ./examples/create-vm -memory -project demo -workspace dev -name example-vm
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

	fmt.Printf("=== create virtual machine %q (%s) ===\n", cfg.Name, vmcommon.ScopeLabel(cfg.Workspace))
	vm := vmcommon.SampleVM(cfg.Project, cfg.Workspace, cfg.Name)

	if memClient != nil {
		created := vmcommon.ApplyMemory(ctx, memClient, vm)
		vmcommon.PrintYAML("virtualmachine", created)
		return
	}

	created, err := vms.Create(ctx, vm, gpupaas.CreateOptions{})
	if err != nil {
		panic(err)
	}
	vmcommon.PrintYAML("virtualmachine", created)
}
