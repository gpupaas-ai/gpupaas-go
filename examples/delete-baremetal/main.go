// Delete a baremetal machine by name.
//
//	go run ./examples/delete-baremetal -memory -project demo -name example-bm
//	go run ./examples/delete-baremetal -project demo -name example-bm
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

	fmt.Printf("=== delete baremetal machine %q (project-scoped) ===\n", cfg.Name)

	if memClient != nil {
		bm := baremetalcommon.SampleBaremetalMachine(cfg.Project, cfg.Name, cfg.ImageURL)
		baremetalcommon.ApplyMemory(ctx, memClient, bm)
		baremetalcommon.DeleteMemory(ctx, memClient, cfg.Project, cfg.Name)
		fmt.Println("deleted (memory backend)")
		return
	}

	if err := bms.Delete(ctx, cfg.Name, gpupaas.DeleteOptions{}); err != nil {
		panic(err)
	}
	fmt.Println("deleted")
}
