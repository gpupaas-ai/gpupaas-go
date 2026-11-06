// Reboot a bare-metal machine (POST .../action/reboot).
//
// Action returns once the backend accepts the verb; pass -wait to poll until the machine has settled.
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/baremetal-reboot -project demo -name example-bm
//	go run ./examples/baremetal-reboot -project demo -workspace dev -name example-bm -wait
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

	fmt.Printf("=== reboot bare-metal machine %q (%s) ===\n", cfg.Name, baremetalcommon.ScopeLabel(cfg.Workspace))
	result, err := bms.Action(ctx, cfg.Name, "reboot", nil)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.Wait {
		result, err = typed.WaitForBareMetal(ctx, bms, cfg.Name, typed.BareMetalWaitOptions{Done: typed.BareMetalActionDone("reboot"), Stable: 2})
		if err != nil {
			log.Fatal(err)
		}
	}
	baremetalcommon.Print("baremetal", result)
}
