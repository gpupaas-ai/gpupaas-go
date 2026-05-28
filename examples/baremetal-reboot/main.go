// Reboot a baremetal machine (GET .../reboot).
//
// Requires live API — actions are not supported on the in-memory backend.
//
//	go run ./examples/baremetal-reboot -project demo -name example-bm
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
	if cfg.UseMemory {
		log.Fatal("baremetal-reboot requires live API; omit -memory")
	}
	ctx := context.Background()

	bms, _, err := baremetalcommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== reboot baremetal machine %q ===\n", cfg.Name)
	result, err := bms.Reboot(ctx, cfg.Name, gpupaas.ActionOptions{})
	if err != nil {
		panic(err)
	}
	baremetalcommon.PrintYAML("baremetalmachine", result)
}
