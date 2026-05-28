// Get an MKS cluster.
//
//	go run ./examples/get-mks-cluster -memory -project demo -name my-cluster
//	go run ./examples/get-mks-cluster -project demo -name my-cluster
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

	fmt.Printf("=== get MKS cluster %q ===\n", cfg.Name)

	if memClient != nil {
		got := mkscommon.GetMemory(ctx, memClient, cfg.Project, cfg.Name)
		mkscommon.PrintYAML("mkscluster", got)
		return
	}

	got, err := clusters.Get(ctx, cfg.Name, gpupaas.GetOptions{})
	if err != nil {
		panic(err)
	}
	mkscommon.PrintYAML("mkscluster", got)
}
