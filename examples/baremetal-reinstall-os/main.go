// Reinstall the operating system of a bare-metal machine (POST .../action/Reinstall OS).
//
// The verb takes the new image in its payload; it must be one of the images of the machine's
// BmProfile. Action returns once the backend accepts the verb; pass -wait to poll until the
// machine has settled.
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/baremetal-reinstall-os -project demo -name example-bm -image ubuntu-26
package main

import (
	"context"
	"fmt"
	"log"

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

	fmt.Printf("=== reinstall OS on bare-metal machine %q (%s, image=%s) ===\n", cfg.Name, baremetalcommon.ScopeLabel(cfg.Workspace), cfg.Image)
	result, err := bms.Action(ctx, cfg.Name, "Reinstall OS", map[string]any{"image": cfg.Image})
	if err != nil {
		log.Fatal(err)
	}
	if cfg.Wait {
		result, err = typed.WaitForBareMetal(ctx, bms, cfg.Name, typed.BareMetalWaitOptions{Done: typed.BareMetalActionDone("Reinstall OS"), Stable: 2})
		if err != nil {
			log.Fatal(err)
		}
	}
	baremetalcommon.Print("baremetal", result)
}
