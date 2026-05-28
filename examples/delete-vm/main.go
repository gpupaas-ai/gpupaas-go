// Delete a virtual machine by name.
//
//	go run ./examples/delete-vm -memory -project demo -name example-vm
//	go run ./examples/delete-vm -project demo -workspace dev -name example-vm
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

	fmt.Printf("=== delete virtual machine %q (%s) ===\n", cfg.Name, vmcommon.ScopeLabel(cfg.Workspace))

	if memClient != nil {
		vm := vmcommon.SampleVM(cfg.Project, cfg.Workspace, cfg.Name)
		vmcommon.ApplyMemory(ctx, memClient, vm)
		vmcommon.DeleteMemory(ctx, memClient, cfg.Project, cfg.Workspace, cfg.Name)
		fmt.Printf("deleted %q\n", cfg.Name)
		return
	}

	if err := vms.Delete(ctx, cfg.Name, gpupaas.DeleteOptions{}); err != nil {
		panic(err)
	}
	fmt.Printf("deleted %q\n", cfg.Name)
}
