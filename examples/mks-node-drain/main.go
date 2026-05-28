// Drain, cordon, and uncordon an MKS node (POST .../mksnodes/{name}/{action}).
//
// Requires live API — MKS sub-resources are not supported on the in-memory backend.
//
//	go run ./examples/mks-node-drain -project demo -name my-cluster -node master-1
package main

import (
	"context"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/examples/mkscommon"
)

func main() {
	cfg := mkscommon.ParseFlags()
	if cfg.UseMemory {
		log.Fatal("mks-node-drain requires live API; omit -memory")
	}
	if cfg.NodeName == "" {
		log.Fatal("-node is required")
	}
	ctx := context.Background()

	clusters, _, err := mkscommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}
	nodes := clusters.Nodes(cfg.Name)

	force := true
	fmt.Printf("=== drain node %q in cluster %q ===\n", cfg.NodeName, cfg.Name)
	drained, err := nodes.Drain(ctx, cfg.NodeName, &v1alpha1.MKSDrainRequest{Force: &force}, gpupaas.ActionOptions{})
	if err != nil {
		panic(err)
	}
	mkscommon.PrintAny("mksnode", drained)

	fmt.Printf("=== cordon node %q ===\n", cfg.NodeName)
	cordoned, err := nodes.Cordon(ctx, cfg.NodeName, gpupaas.ActionOptions{})
	if err != nil {
		panic(err)
	}
	mkscommon.PrintAny("mksnode", cordoned)

	fmt.Printf("=== uncordon node %q ===\n", cfg.NodeName)
	uncordoned, err := nodes.Uncordon(ctx, cfg.NodeName, gpupaas.ActionOptions{})
	if err != nil {
		panic(err)
	}
	mkscommon.PrintAny("mksnode", uncordoned)
}
