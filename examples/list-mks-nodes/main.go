// List nodes in an MKS cluster.
//
// Requires live API — MKS sub-resources are not supported on the in-memory backend.
//
//	go run ./examples/list-mks-nodes -project demo -name my-cluster
package main

import (
	"context"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/examples/mkscommon"
)

func main() {
	cfg := mkscommon.ParseFlags()
	if cfg.UseMemory {
		log.Fatal("list-mks-nodes requires live API; omit -memory")
	}
	ctx := context.Background()

	clusters, _, err := mkscommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== list nodes in MKS cluster %q ===\n", cfg.Name)
	list, err := clusters.Nodes(cfg.Name).List(ctx, gpupaas.ListOptions{})
	if err != nil {
		panic(err)
	}
	for i := range list.Items {
		mkscommon.PrintAny("mksnode", &list.Items[i])
	}
}
