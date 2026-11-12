// Upgrade a first-class MKS cluster by applying a new Kubernetes version, then wait until it
// is RUNNING again. Clusters are declarative: an upgrade is an update of spec.kubernetesVersion.
//
//	go run ./examples/mks-cluster-upgrade -project demo -name my-cluster -k8s-version 1.36
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	typedv1alpha1 "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
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

	cluster, err := clusters.Get(ctx, cfg.Name, gpupaas.GetOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("=== upgrade MKS cluster %q from %s to %s ===\n", cfg.Name, cluster.Spec.KubernetesVersion, cfg.K8sVersion)
	cluster.Spec.KubernetesVersion = cfg.K8sVersion
	if _, err := clusters.Update(ctx, cluster, gpupaas.UpdateOptions{}); err != nil {
		panic(err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()
	done, err := typedv1alpha1.WaitForMKSCluster(waitCtx, clusters, cfg.Name, typedv1alpha1.MKSWaitOptions{
		Done:   typedv1alpha1.MKSClusterMatches(cluster.Spec),
		Stable: 3,
	})
	if err != nil {
		panic(err)
	}
	mkscommon.PrintYAML("mkscluster", done)
}
