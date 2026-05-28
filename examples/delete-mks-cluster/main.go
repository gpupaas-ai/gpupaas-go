// Delete an MKS cluster.
//
//	go run ./examples/delete-mks-cluster -memory -project demo -name my-cluster
//	go run ./examples/delete-mks-cluster -project demo -name my-cluster
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

	fmt.Printf("=== delete MKS cluster %q ===\n", cfg.Name)

	if memClient != nil {
		mkscommon.DeleteMemory(ctx, memClient, cfg.Project, cfg.Name)
		fmt.Printf("deleted %q\n", cfg.Name)
		return
	}

	if err := clusters.Delete(ctx, cfg.Name, gpupaas.DeleteOptions{}); err != nil {
		panic(err)
	}
	fmt.Printf("deleted %q\n", cfg.Name)
}
