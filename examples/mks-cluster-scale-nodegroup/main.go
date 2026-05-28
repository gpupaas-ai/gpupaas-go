// Scale an MKS worker node group (POST .../scaleNodeGroup).
//
// Requires live API — sub-actions are not supported on the in-memory backend.
//
//	go run ./examples/mks-cluster-scale-nodegroup -project demo -name my-cluster -node-group wng-1 -desired-size 5
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
		log.Fatal("mks-cluster-scale-nodegroup requires live API; omit -memory")
	}
	if cfg.NodeGroup == "" {
		log.Fatal("-node-group is required")
	}
	ctx := context.Background()

	clusters, _, err := mkscommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	desired := int32(cfg.DesiredSize)
	fmt.Printf("=== scale node group %q in cluster %q to %d ===\n", cfg.NodeGroup, cfg.Name, cfg.DesiredSize)
	result, err := clusters.ScaleNodeGroup(ctx, cfg.Name, &v1alpha1.MKSScaleNodeGroupRequest{
		NodeGroupName: cfg.NodeGroup,
		DesiredCount:  &desired,
	}, gpupaas.ActionOptions{})
	if err != nil {
		panic(err)
	}
	mkscommon.PrintYAML("mkscluster", result)
}
