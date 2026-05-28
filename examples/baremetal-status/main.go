// Fetch the runtime status information for a baremetal machine
// (GET .../{name}/status). The response is the free-form BaremetalMachineInfo
// envelope and is distinct from the Conditions in BaremetalMachine.Status.
//
// Requires live API — status info is not modelled on the in-memory backend.
//
//	go run ./examples/baremetal-status -project demo -name example-bm
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
		log.Fatal("baremetal-status requires live API; omit -memory")
	}
	ctx := context.Background()

	bms, _, err := baremetalcommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== get baremetal machine %q status info ===\n", cfg.Name)
	info, err := bms.GetStatusInfo(ctx, cfg.Name, gpupaas.GetOptions{})
	if err != nil {
		panic(err)
	}
	baremetalcommon.PrintAny("baremetalmachine-info", info)
}
