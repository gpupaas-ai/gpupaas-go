// Start a virtual machine (POST .../action/start).
//
// Action returns once the backend accepts the verb; pass -wait to poll until the VM has settled.
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/vm-start -project demo -name example-vm
//	go run ./examples/vm-start -project demo -workspace dev -name example-vm -wait
package main

import (
	"context"
	"fmt"
	"log"

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

	fmt.Printf("=== start virtual machine %q (%s) ===\n", cfg.Name, vmcommon.ScopeLabel(cfg.Workspace))
	result, err := vms.Action(ctx, cfg.Name, "start", nil)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.Wait {
		result, err = typed.WaitForVMInstance(ctx, vms, cfg.Name, typed.VMInstanceWaitOptions{Done: typed.VMInstanceActionDone("start"), Stable: 2})
		if err != nil {
			log.Fatal(err)
		}
	}
	vmcommon.Print("virtualmachine", result)
}
