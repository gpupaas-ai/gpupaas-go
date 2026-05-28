// List baremetal machines for a project.
//
//	go run ./examples/list-baremetals -memory -project demo
//	go run ./examples/list-baremetals -project demo
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

	fmt.Printf("=== list baremetal machines in project %q ===\n", cfg.Project)

	if memClient != nil {
		bm := baremetalcommon.SampleBaremetalMachine(cfg.Project, cfg.Name, cfg.ImageURL)
		baremetalcommon.ApplyMemory(ctx, memClient, bm)
		items := baremetalcommon.ListMemory(ctx, memClient, cfg.Project)
		fmt.Printf("found %d item(s)\n", len(items))
		for _, item := range items {
			baremetalcommon.PrintYAML("baremetalmachine", item)
		}
		return
	}

	list, err := bms.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("found %d item(s)\n", len(list.Items))
	for i := range list.Items {
		baremetalcommon.PrintYAML("baremetalmachine", &list.Items[i])
	}
}
