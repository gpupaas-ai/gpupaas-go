// Get a virtual machine by name.
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/get-vm -project demo -name example-vm
//	go run ./examples/get-vm -project demo -workspace dev -name example-vm
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
	ctx := context.Background()

	vms, err := vmcommon.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("=== get virtual machine %q (%s) ===\n", cfg.Name, vmcommon.ScopeLabel(cfg.Workspace))
	got, err := vms.Get(ctx, cfg.Name, gpupaas.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	vmcommon.Print("virtualmachine", got)
}
