// Delete a bare-metal machine by name.
//
// Delete starts teardown; the machine stays visible (deleting) until it is gone. Pass -wait to
// poll until it has disappeared.
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/delete-baremetal -project demo -name example-bm
//	go run ./examples/delete-baremetal -project demo -workspace dev -name example-bm -wait
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

	fmt.Printf("=== delete bare-metal machine %q (%s) ===\n", cfg.Name, baremetalcommon.ScopeLabel(cfg.Workspace))
	if err := bms.Delete(ctx, cfg.Name, gpupaas.DeleteOptions{}); err != nil {
		log.Fatal(err)
	}
	if cfg.Wait {
		if _, err := typed.WaitForBareMetal(ctx, bms, cfg.Name, typed.BareMetalWaitOptions{Done: typed.BareMetalGone, GoneIsDone: true}); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("deleted %q\n", cfg.Name)
}
