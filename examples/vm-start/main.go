// Start a virtual machine (POST .../action/start).
//
// Requires live API — actions are not supported on the in-memory backend.
//
//	go run ./examples/vm-start -project demo -name example-vm
//	go run ./examples/vm-start -project demo -workspace dev -name example-vm
package main

import (
	"context"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/examples/vmcommon"
)

func main() {
	cfg := vmcommon.ParseFlags()
	if cfg.UseMemory {
		log.Fatal("vm-start requires live API; omit -memory")
	}
	ctx := context.Background()

	vms, _, err := vmcommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== start virtual machine %q (%s) ===\n", cfg.Name, vmcommon.ScopeLabel(cfg.Workspace))

	result, err := vms.Start(ctx, cfg.Name, gpupaas.ActionOptions{})
	if err != nil {
		panic(err)
	}
	vmcommon.PrintYAML("virtualmachine", result)
}
