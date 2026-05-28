// Create (apply) an MKS cluster.
//
//	go run ./examples/create-mks-cluster -memory -project demo -name my-cluster
//	go run ./examples/create-mks-cluster -project demo -name my-cluster
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

	fmt.Printf("=== create MKS cluster %q (project-scoped) ===\n", cfg.Name)
	cluster := mkscommon.SampleCluster(cfg.Project, cfg.Name)

	if memClient != nil {
		created := mkscommon.ApplyMemory(ctx, memClient, cluster)
		mkscommon.PrintYAML("mkscluster", created)
		return
	}

	created, err := clusters.Create(ctx, cluster, gpupaas.CreateOptions{})
	if err != nil {
		panic(err)
	}
	mkscommon.PrintYAML("mkscluster", created)
}
