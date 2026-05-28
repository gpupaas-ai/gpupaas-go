// List security groups at project or workspace scope.
package main

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/examples/devcommon"
)

func main() {
	cfg := devcommon.ParseFlags("default-sg")
	ctx := context.Background()
	fmt.Printf("=== list security groups (%s) ===\n", devcommon.ScopeLabel(cfg.Workspace))

	if cfg.UseMemory {
		c, _ := devcommon.NewGenericClient(cfg)
		items := devcommon.ListMemory(ctx, c, v1alpha1.KindSecurityGroup, cfg.Project, cfg.Workspace)
		fmt.Printf("count: %d\n", len(items))
		for _, item := range items {
			devcommon.PrintYAML("securitygroup", item)
		}
		return
	}

	cs, err := devcommon.NewTypedClientset(cfg)
	if err != nil {
		panic(err)
	}
	var list *v1alpha1.SecurityGroupList
	if cfg.Workspace != "" {
		list, err = cs.V1alpha1().Workspaces(cfg.Project).SecurityGroups(cfg.Workspace).List(ctx, gpupaas.ListOptions{})
	} else {
		list, err = cs.V1alpha1().SecurityGroups(cfg.Project).List(ctx, gpupaas.ListOptions{})
	}
	if err != nil {
		panic(err)
	}
	fmt.Printf("count: %d\n", len(list.Items))
	for i := range list.Items {
		devcommon.PrintYAML("securitygroup", &list.Items[i])
	}
}
