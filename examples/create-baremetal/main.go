// Create (apply) a baremetal machine.
//
//	go run ./examples/create-baremetal -memory -project demo -name example-bm
//	go run ./examples/create-baremetal -project demo -name example-bm
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

	fmt.Printf("=== create baremetal machine %q (project-scoped) ===\n", cfg.Name)
	bm := baremetalcommon.SampleBaremetalMachine(cfg.Project, cfg.Name, cfg.ImageURL)

	if memClient != nil {
		created := baremetalcommon.ApplyMemory(ctx, memClient, bm)
		baremetalcommon.PrintYAML("baremetalmachine", created)
		return
	}

	created, err := bms.Create(ctx, bm, gpupaas.CreateOptions{})
	if err != nil {
		panic(err)
	}
	baremetalcommon.PrintYAML("baremetalmachine", created)
}
