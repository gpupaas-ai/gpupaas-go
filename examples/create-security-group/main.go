// Create a security group.
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
	obj := devcommon.SampleSecurityGroup(cfg.Project, cfg.Workspace, cfg.Name)
	fmt.Printf("=== create security group %q (%s) ===\n", cfg.Name, devcommon.ScopeLabel(cfg.Workspace))

	if cfg.UseMemory {
		c, _ := devcommon.NewGenericClient(cfg)
		devcommon.PrintYAML("securitygroup", devcommon.ApplyMemory(ctx, c, obj, cfg.Project, cfg.Workspace))
		return
	}

	cs, err := devcommon.NewTypedClientset(cfg)
	if err != nil {
		panic(err)
	}
	var created *v1alpha1.SecurityGroup
	if cfg.Workspace != "" {
		created, err = cs.V1alpha1().Workspaces(cfg.Project).SecurityGroups(cfg.Workspace).Create(ctx, obj, gpupaas.CreateOptions{})
	} else {
		created, err = cs.V1alpha1().SecurityGroups(cfg.Project).Create(ctx, obj, gpupaas.CreateOptions{})
	}
	if err != nil {
		panic(err)
	}
	devcommon.PrintYAML("securitygroup", created)
}
