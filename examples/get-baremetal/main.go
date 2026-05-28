// Get a baremetal machine by name.
//
//	go run ./examples/get-baremetal -memory -project demo -name example-bm
//	go run ./examples/get-baremetal -project demo -name example-bm
package main

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/examples/baremetalcommon"
)

func main() {
	cfg := baremetalcommon.ParseFlags()
	ctx := context.Background()

	bms, memClient, err := baremetalcommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== get baremetal machine %q (project-scoped) ===\n", cfg.Name)

	if memClient != nil {
		bm := baremetalcommon.SampleBaremetalMachine(cfg.Project, cfg.Name, cfg.ImageURL)
		baremetalcommon.ApplyMemory(ctx, memClient, bm)
		got := baremetalcommon.GetMemory(ctx, memClient, cfg.Project, cfg.Name)
		baremetalcommon.PrintYAML("baremetalmachine", got)
		return
	}

	got, err := bms.Get(ctx, cfg.Name, gpupaas.GetOptions{})
	if err != nil {
		panic(err)
	}
	baremetalcommon.PrintYAML("baremetalmachine", got)
}
