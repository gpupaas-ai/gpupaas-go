// Upgrade an MKS cluster's Kubernetes version (POST .../upgrade).
//
// Requires live API — sub-actions are not supported on the in-memory backend.
//
//	go run ./examples/mks-cluster-upgrade -project demo -name my-cluster -k8s-version 1.32
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
		log.Fatal("mks-cluster-upgrade requires live API; omit -memory")
	}
	ctx := context.Background()

	clusters, _, err := mkscommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== upgrade MKS cluster %q to %s ===\n", cfg.Name, cfg.K8sVersion)
	result, err := clusters.Upgrade(ctx, cfg.Name, &v1alpha1.MKSUpgradeRequest{
		K8sVersion: cfg.K8sVersion,
	}, gpupaas.ActionOptions{})
	if err != nil {
		panic(err)
	}
	mkscommon.PrintYAML("mkscluster", result)
}
