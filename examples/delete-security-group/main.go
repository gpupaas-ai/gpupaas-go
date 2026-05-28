// Delete a security group by name.
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
	fmt.Printf("=== delete security group %q (%s) ===\n", cfg.Name, devcommon.ScopeLabel(cfg.Workspace))

	if cfg.UseMemory {
		c, _ := devcommon.NewGenericClient(cfg)
		devcommon.DeleteMemory(ctx, c, v1alpha1.KindSecurityGroup, cfg.Project, cfg.Workspace, cfg.Name)
		fmt.Println("deleted")
		return
	}

	cs, err := devcommon.NewTypedClientset(cfg)
	if err != nil {
		panic(err)
	}
	if cfg.Workspace != "" {
		err = cs.V1alpha1().Workspaces(cfg.Project).SecurityGroups(cfg.Workspace).Delete(ctx, cfg.Name, gpupaas.DeleteOptions{})
	} else {
		err = cs.V1alpha1().SecurityGroups(cfg.Project).Delete(ctx, cfg.Name, gpupaas.DeleteOptions{})
	}
	if err != nil {
		panic(err)
	}
	fmt.Println("deleted")
}
