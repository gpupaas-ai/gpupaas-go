// Apply and list an MKS cluster through the generic client (in-memory backend).
//
// The generic client (client.Client) applies, gets, lists and deletes objects by kind; it handles
// MKSCluster. Use the typed clientset (clientset.NewForConfig) for first-class VMs and bare metal.
package main

import (
	"context"
	"fmt"
	"log"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/client"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

func main() {
	//c := client.New(client.Options{Config: gpupaas.ConfigFromEnv()})
	// OR
	c := client.New(client.Options{UseMemory: true})
	// OR
	// c := client.New(client.Options{
	// 	Config: gpupaas.NewConfig("https://api.gpupaas.ai", "your-api-token"),
	// })

	ctx := context.Background()

	cluster := &v1alpha1.MKSCluster{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindMKSCluster},
		Metadata: v1alpha1.ObjectMeta{Name: "example", Project: "demo"},
		Spec: v1alpha1.MKSClusterSpec{
			Profile:           &v1alpha1.MKSProfileRef{Name: "mks-oneclick-1"},
			KubernetesVersion: "1.36",
		},
	}
	applied, err := c.Apply(ctx, cluster, "demo", "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("applied %s/%s\n", applied.GetKind(), applied.GetName())

	gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindMKSCluster}
	items, err := c.List(ctx, gvk, "demo", "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("listed %d cluster(s)\n", len(items))
}
