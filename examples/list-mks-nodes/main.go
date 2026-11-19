// List the nodes a first-class MKS cluster has registered.
//
// Requires live API — nodes are not supported on the in-memory backend.
//
//	go run ./examples/list-mks-nodes -project demo -name my-cluster
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gpupaas-ai/gpupaas-go/examples/mkscommon"
)

func main() {
	cfg := mkscommon.ParseFlags()
	if cfg.UseMemory {
		log.Fatal("list-mks-nodes requires live API; omit -memory")
	}
	ctx := context.Background()

	typed, _, err := mkscommon.NewTyped(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== list nodes of MKS cluster %q ===\n", cfg.Name)
	list, err := typed.MKSNodes(cfg.Project).ListForCluster(ctx, cfg.Name)
	if err != nil {
		panic(err)
	}
	for i := range list.Items {
		n := &list.Items[i]
		fmt.Printf("%s pool=%s kube=%s\n", n.Metadata.Name, n.NodePool(), n.Spec.KubeVersion)
	}
}
