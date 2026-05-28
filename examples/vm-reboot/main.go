// Reboot a virtual machine (POST .../action/reboot).
//
// Requires live API — actions are not supported on the in-memory backend.
//
//	go run ./examples/vm-reboot -project demo -name example-vm
//	go run ./examples/vm-reboot -project demo -workspace dev -name example-vm
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
		log.Fatal("vm-reboot requires live API; omit -memory")
	}
	ctx := context.Background()

	vms, _, err := vmcommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== reboot virtual machine %q (%s) ===\n", cfg.Name, vmcommon.ScopeLabel(cfg.Workspace))

	// Reboot is the convenience for Action(ctx, name, "reboot", opts).
	result, err := vms.Reboot(ctx, cfg.Name, gpupaas.ActionOptions{})
	if err != nil {
		panic(err)
	}
	vmcommon.PrintYAML("virtualmachine", result)
}
