// Workspace collaborator example: list, add (read+write / read-only + SSO), remove.
//
// In-memory (no auth, no network):
//
//	go run ./examples/workspace-collaborator -memory -project demo -workspace dev
//
// Live API (API key required, same as paasctl):
//
//	export GPUPAAS_API_KEY=your-api-key
//	export GPUPAAS_API_SECRET=your-api-secret   # optional
//	export GPUPAAS_ENDPOINT=https://console.gpupaas.ai   # optional
//	go run ./examples/workspace-collaborator -project demo -workspace dev -v
//
// Use -cleanup=false to leave collaborators on the workspace after the run.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/client"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	typedv1alpha1 "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
	"gopkg.in/yaml.v3"
)

type collaboratorCase struct {
	name  string
	role  string
	isSSO bool
}

var demoCollaborators = []collaboratorCase{
	{name: "benny+batch1@rafay.co", role: v1alpha1.WorkspaceRoleCollaborator, isSSO: false},
	{name: "benny+labadmin1@rafay.co", role: v1alpha1.WorkspaceRoleCollaboratorReadOnly, isSSO: true},
}

func main() {
	useMemory := flag.Bool("memory", false, "use in-memory backend (no auth, no HTTP)")
	verbose := flag.Bool("v", false, "log HTTP requests and responses")
	verboseLong := flag.Bool("verbose", false, "log HTTP requests and responses")
	project := flag.String("project", "demo", "project name")
	workspace := flag.String("workspace", "dev", "workspace name")
	cleanup := flag.Bool("cleanup", true, "remove collaborators added by this example")
	flag.Parse()

	ctx := context.Background()
	v := *verbose || *verboseLong

	if *useMemory {
		runMemory(ctx, client.New(client.Options{UseMemory: true}), *project, *workspace, *cleanup)
		return
	}

	cfg := gpupaas.ConfigFromEnv()
	if cfg.APIKey == "" {
		log.Fatal("GPUPAAS_API_KEY is required for live API (or use -memory)")
	}
	cfg.Verbose = v

	cs, err := clientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("clientset: %v", err)
	}
	runTyped(ctx, cs.V1alpha1().Workspaces(*project).Collaborators(*workspace), *project, *workspace, *cleanup)
}

func runMemory(ctx context.Context, c *client.Client, project, workspace string, cleanup bool) {
	gvk := runtime.GroupVersionKind{
		Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindWorkspaceCollaborator,
	}

	fmt.Println("=== list collaborators (initial) ===")
	printCollaboratorList(listMemory(ctx, c, gvk, project, workspace))

	fmt.Println("=== add collaborators ===")
	for _, ex := range demoCollaborators {
		obj := newCollaborator(project, workspace, ex)
		applied, err := c.Apply(ctx, obj, project, workspace)
		if err != nil {
			log.Fatalf("apply collaborator %q: %v", ex.name, err)
		}
		printCollaborator("added", applied)
	}

	fmt.Println("=== list collaborators (after add) ===")
	printCollaboratorList(listMemory(ctx, c, gvk, project, workspace))

	if !cleanup {
		fmt.Println("skipping cleanup (-cleanup=false)")
		return
	}

	fmt.Println("=== remove collaborators ===")
	for _, ex := range demoCollaborators {
		if err := c.Delete(ctx, gvk, project, workspace, ex.name); err != nil {
			log.Fatalf("delete collaborator %q: %v", ex.name, err)
		}
		fmt.Printf("removed %q\n", ex.name)
	}

	fmt.Println("=== list collaborators (after remove) ===")
	printCollaboratorList(listMemory(ctx, c, gvk, project, workspace))
}

func runTyped(ctx context.Context, collab typedv1alpha1.WorkspaceCollaboratorInterface, project, workspace string, cleanup bool) {
	fmt.Println("=== list collaborators (initial) ===")
	printCollaboratorListTyped(listTyped(ctx, collab, nil))

	fmt.Println("=== add collaborators ===")
	added := make([]collaboratorCase, 0, len(demoCollaborators))
	for _, ex := range demoCollaborators {
		obj := newCollaborator(project, workspace, ex)
		created, err := collab.Create(ctx, obj, gpupaas.CreateOptions{})
		if err != nil {
			log.Fatalf("create collaborator %q: %v", ex.name, err)
		}
		printCollaborator("added", created)
		added = append(added, ex)
	}

	fmt.Println("=== list collaborators (after add) ===")
	printCollaboratorListTyped(listTyped(ctx, collab, nil))

	sso := true
	fmt.Println("=== list SSO collaborators only (ssoUsers=true) ===")
	printCollaboratorListTyped(listTyped(ctx, collab, &sso))

	if !cleanup {
		fmt.Println("skipping cleanup (-cleanup=false)")
		return
	}

	fmt.Println("=== remove collaborators ===")
	for _, ex := range added {
		opts := gpupaas.DeleteOptions{}
		if ex.isSSO {
			opts.SSOUser = &sso
		}
		if err := collab.Delete(ctx, ex.name, opts); err != nil {
			log.Fatalf("delete collaborator %q: %v", ex.name, err)
		}
		fmt.Printf("removed %q (sso=%v)\n", ex.name, ex.isSSO)
	}

	fmt.Println("=== list collaborators (after remove) ===")
	printCollaboratorListTyped(listTyped(ctx, collab, nil))
}

func newCollaborator(project, workspace string, ex collaboratorCase) *v1alpha1.WorkspaceCollaborator {
	return &v1alpha1.WorkspaceCollaborator{
		TypeMeta: v1alpha1.TypeMeta{
			APIVersion: v1alpha1.APIVersion,
			Kind:       v1alpha1.KindWorkspaceCollaborator,
		},
		Metadata: v1alpha1.ObjectMeta{
			Name:      ex.name,
			Project:   project,
			Workspace: workspace,
		},
		Spec: v1alpha1.WorkspaceCollaboratorSpec{
			Username:  ex.name,
			Role:      ex.role,
			IsSSOUser: ex.isSSO,
		},
	}
}

func listMemory(ctx context.Context, c *client.Client, gvk runtime.GroupVersionKind, project, workspace string) []runtime.Object {
	items, err := c.List(ctx, gvk, project, workspace)
	if err != nil {
		log.Fatalf("list collaborators: %v", err)
	}
	return items
}

func listTyped(ctx context.Context, collab typedv1alpha1.WorkspaceCollaboratorInterface, sso *bool) *v1alpha1.WorkspaceCollaboratorList {
	list, err := collab.List(ctx, gpupaas.ListOptions{SSOUsers: sso})
	if err != nil {
		log.Fatalf("list collaborators: %v", err)
	}
	return list
}

func printCollaboratorList(items []runtime.Object) {
	fmt.Printf("count: %d\n", len(items))
	for _, item := range items {
		printCollaborator("collaborator", item)
	}
}

func printCollaboratorListTyped(list *v1alpha1.WorkspaceCollaboratorList) {
	fmt.Printf("count: %d\n", len(list.Items))
	for i := range list.Items {
		printCollaborator("collaborator", &list.Items[i])
	}
}

func printCollaborator(label string, obj runtime.Object) {
	data, err := yaml.Marshal(obj)
	if err != nil {
		log.Fatalf("marshal %s: %v", label, err)
	}
	fmt.Printf("%s:\n%s---\n", label, string(data))
}
