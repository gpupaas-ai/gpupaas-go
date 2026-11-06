// Create a first-class virtual machine from a VM profile (SKU).
//
// Create returns as soon as the backend accepts the request; the platform provisions the VM
// asynchronously. Pass -wait to poll until it is running (or has failed).
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/create-vm -project demo -name example-vm -sku vm-sku -cpu 2 -memory 4 -ssh-key my-key
//	go run ./examples/create-vm -project demo -workspace dev -name example-vm -wait
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

	fmt.Printf("=== create virtual machine %q (%s) ===\n", cfg.Name, vmcommon.ScopeLabel(cfg.Workspace))
	created, err := vms.Create(ctx, vmcommon.SampleVM(cfg), gpupaas.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	}
	vmcommon.Print("virtualmachine", created)

	if cfg.Wait {
		ready, err := typed.WaitForVMInstance(ctx, vms, cfg.Name, typed.VMInstanceWaitOptions{Done: typed.VMInstanceRunning})
		if err != nil {
			log.Fatal(err)
		}
		vmcommon.Print("virtualmachine (running)", ready)
	}
}
