// List MKS clusters in a project.
//
//	go run ./examples/list-mks-clusters -memory -project demo
//	go run ./examples/list-mks-clusters -project demo
package main

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/examples/mkscommon"
)

func main() {
	cfg := mkscommon.ParseFlags()
	ctx := context.Background()

	clusters, memClient, err := mkscommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== list MKS clusters in project %q ===\n", cfg.Project)

	if memClient != nil {
		for _, obj := range mkscommon.ListMemory(ctx, memClient, cfg.Project) {
			mkscommon.PrintYAML("mkscluster", obj)
		}
		return
	}

	list, err := clusters.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		panic(err)
	}
	for i := range list.Items {
		mkscommon.PrintYAML("mkscluster", &list.Items[i])
	}
}
