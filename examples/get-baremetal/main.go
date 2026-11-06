// Get a bare-metal machine by name.
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/get-baremetal -project demo -name example-bm
//	go run ./examples/get-baremetal -project demo -workspace dev -name example-bm
package main

import (
	"context"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/examples/baremetalcommon"
)

func main() {
	cfg := baremetalcommon.ParseFlags()
	ctx := context.Background()

	bms, err := baremetalcommon.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("=== get bare-metal machine %q (%s) ===\n", cfg.Name, baremetalcommon.ScopeLabel(cfg.Workspace))
	got, err := bms.Get(ctx, cfg.Name, gpupaas.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	baremetalcommon.Print("baremetal", got)
}
