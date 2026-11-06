// Create a first-class bare-metal machine from a BmProfile (SKU).
//
// Create returns as soon as the backend accepts the request; the platform provisions the machine
// asynchronously (minutes). Pass -wait to poll until it is running (or has failed).
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/create-baremetal -project demo -name example-bm -sku bm-sku -image ubuntu-24 -ssh-key my-key
//	go run ./examples/create-baremetal -project demo -workspace dev -name example-bm -wait
package main

import (
	"context"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/examples/baremetalcommon"
)

func main() {
	cfg := baremetalcommon.ParseFlags()
	ctx := context.Background()

	bms, err := baremetalcommon.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("=== create bare-metal machine %q (%s) ===\n", cfg.Name, baremetalcommon.ScopeLabel(cfg.Workspace))
	created, err := bms.Create(ctx, baremetalcommon.SampleBareMetal(cfg), gpupaas.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	}
	baremetalcommon.Print("baremetal", created)

	if cfg.Wait {
		ready, err := typed.WaitForBareMetal(ctx, bms, cfg.Name, typed.BareMetalWaitOptions{Done: typed.BareMetalRunning})
		if err != nil {
			log.Fatal(err)
		}
		baremetalcommon.Print("baremetal (running)", ready)
	}
}
