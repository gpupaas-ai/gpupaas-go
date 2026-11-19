// Delete a virtual machine by name.
//
// Delete starts teardown; the VM stays visible (deleting) until it is gone. Pass -wait to poll
// until it has disappeared.
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/delete-vm -project demo -name example-vm
//	go run ./examples/delete-vm -project demo -workspace dev -name example-vm -wait
package main

import (
	"context"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/examples/vmcommon"
)

func main() {
	cfg := vmcommon.ParseFlags()
	ctx := context.Background()

	vms, err := vmcommon.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("=== delete virtual machine %q (%s) ===\n", cfg.Name, vmcommon.ScopeLabel(cfg.Workspace))
	if err := vms.Delete(ctx, cfg.Name, gpupaas.DeleteOptions{}); err != nil {
		log.Fatal(err)
	}
	if cfg.Wait {
		if _, err := typed.WaitForVMInstance(ctx, vms, cfg.Name, typed.VMInstanceWaitOptions{Done: typed.VMInstanceGone, GoneIsDone: true}); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("deleted %q\n", cfg.Name)
}
