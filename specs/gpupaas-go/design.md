# Design: gpupaas-go SDK

## Overview

`gpupaas-go` is a layered Go library aligned with [`paasctl/examples/go-client-basic`](https://github.com/RafaySystems/paasctl/blob/main/examples/go-client-basic/main.go):

```text
Consumers (CLI, Terraform, user programs)
        ↓
   client/         ← primary public API (Apply, Get, List, Delete)
        ↓
   engine/         ← validation + backend dispatch
        ↓
   backend/        ← memory (tests) | remote (REST via clientset)
        ↓
   rest/ + clientset/
        ↓
   gpupaas.ai API
```

Supporting packages:

```text
apis/v1alpha1/    ← API types (Project, Workspace)
runtime/          ← Scheme, codec, GroupVersionKind, Object interface
validation/       ← scope defaults and kind rules
apply/            ← ApplyFile / ApplyReader helpers (optional thin wrapper)
watch/            ← Event types (minimal v1)
```

Reference: `paasctl/specs/kubectl_style_cli_architecture_prompt.md` and `paasctl/pkg/client/client.go`.

## Platform scoping model

```text
gpupaas platform
└── Project (metadata.name)
    └── Workspace (metadata.project + metadata.name)
        └── future resources (metadata.project + metadata.workspace + metadata.name)
```

| Kind | Required metadata | REST collection path |
|---|---|---|
| `Project` | `name` | `/projects` |
| `Workspace` | `project`, `name` | `/projects/{project}/workspaces` |
| future kinds | `project`, `workspace`, `name` | `/projects/{project}/workspaces/{workspace}/...` |

Method arguments mirror metadata scope:

- `Apply(ctx, obj, project, workspace)` — fills empty `metadata.project` / `metadata.workspace` from args
- `List(ctx, gvk, project, workspace)` — `project` required; `workspace` filters when non-empty

## Repository layout

```text
github.com/gpupaas-ai/gpupaas-go/
├── go.mod
├── README.md
├── config.go
├── errors.go
├── options.go
├── version.go
├── apis/v1alpha1/
│   ├── types.go
│   ├── register.go
│   ├── project.go
│   └── workspace.go
├── client/
│   └── client.go
├── engine/
│   └── engine.go
├── validation/
│   └── validation.go
├── backend/
│   ├── backend.go
│   ├── memory/
│   │   └── memory.go
│   └── remote/
│       └── remote.go
├── clientset/
│   └── typed/v1alpha1/
│       ├── client.go
│       ├── project.go
│       └── workspace.go
├── runtime/
│   ├── object.go
│   ├── scheme.go
│   ├── codec.go
│   └── gvk.go
├── apply/
│   ├── apply.go
│   └── delete.go
├── rest/
│   ├── client.go
│   └── fake.go
├── watch/
│   ├── watch.go
│   └── event.go
└── examples/
    ├── typed_client/main.go   ← mirrors paasctl go-client-basic
    ├── apply_yaml/main.go
    └── delete_yaml/main.go
```

## Resource model

### Base types

```go
type TypeMeta struct {
    APIVersion string `json:"apiVersion" yaml:"apiVersion"`
    Kind       string `json:"kind" yaml:"kind"`
}

type ObjectMeta struct {
    Name        string            `json:"name" yaml:"name"`
    Project     string            `json:"project,omitempty" yaml:"project,omitempty"`
    Workspace   string            `json:"workspace,omitempty" yaml:"workspace,omitempty"`
    Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}
```

### Initial resources

| Kind | metadata.project | metadata.workspace | Spec highlights | Status highlights |
|---|---|---|---|---|
| `Project` | — | — | `displayName`, `description` | `phase` |
| `Workspace` | required | — | `description` | `phase` |

Example (same shape as paasctl go-client-basic):

```go
ws := &v1alpha1.Workspace{
    TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindWorkspace},
    Metadata: v1alpha1.ObjectMeta{Name: "example", Project: "demo"},
}
```

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: Workspace
metadata:
  name: dev
  project: demo
spec:
  description: Development workspace
```

## Runtime package

### Object interface

```go
type Object interface {
    GetAPIVersion() string
    GetKind() string
    GetName() string
    GetProject() string
    GetWorkspace() string
    SetProject(string)
    SetWorkspace(string)
    DeepCopyObject() Object
}

type GroupVersionKind struct {
    Group   string
    Version string
    Kind    string
}
```

Resource identity:

```text
apiVersion + kind + project + workspace + name
```

(`project` and/or `workspace` omitted when not applicable to the kind.)

## High-level client

```go
type Options struct {
    Config    gpupaas.Config
    UseMemory bool
}

type Client struct {
    Engine *engine.Engine
    Scheme *runtime.Scheme
}

func New(opts Options) *Client

func (c *Client) Apply(ctx context.Context, obj runtime.Object, project, workspace string) (runtime.Object, error)
func (c *Client) Get(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error)
func (c *Client) List(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace string) ([]runtime.Object, error)
func (c *Client) Delete(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) error
```

`New` wiring:

- `UseMemory: true` → `backend/memory.New()`
- else → `backend/remote.New(config, scheme)` using clientset

## Engine

```go
type Engine struct {
    backend backend.Backend
    scheme  *runtime.Scheme
}

func (e *Engine) Apply(ctx context.Context, obj runtime.Object, opts validation.Options) (runtime.Object, error)
func (e *Engine) Get(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error)
func (e *Engine) List(ctx context.Context, gvk runtime.GroupVersionKind, opts backend.ListOptions) ([]runtime.Object, error)
func (e *Engine) Delete(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) error
```

Apply flow:

1. `validation.ValidateObject(obj, opts)` — apply defaults, check required scope
2. `backend.Apply(ctx, obj)` — create or update; do not write status from input

## Validation

```go
type Options struct {
    DefaultProject   string
    DefaultWorkspace string
}

func ValidateObject(obj runtime.Object, opts Options) error
```

Rules (aligned with paasctl):

- Require `apiVersion`, `kind`, `metadata.name`
- If `metadata.project` empty, set from `DefaultProject`; error if still empty
- For `Workspace`, stop after project is set
- For future project+workspace kinds, set `metadata.workspace` from `DefaultWorkspace` when empty

## Backend interface

```go
type Backend interface {
    Apply(ctx context.Context, obj runtime.Object) (runtime.Object, error)
    Get(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error)
    List(ctx context.Context, gvk runtime.GroupVersionKind, opts ListOptions) ([]runtime.Object, error)
    Delete(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) error
}

type ListOptions struct {
    Project   string
    Workspace string
}
```

**memory**: in-process map keyed by GVK + project + workspace + name (for examples/tests).

**remote**: translates to clientset REST calls.

## Typed clientset (remote backend internals)

```go
type Interface interface {
    Projects() ProjectInterface
    Workspaces(project string) WorkspaceInterface
}
```

REST paths:

```text
GET/POST  /apis/gpupaas.ai/v1alpha1/projects
GET/PUT/DELETE /apis/gpupaas.ai/v1alpha1/projects/{name}

GET/POST  /apis/gpupaas.ai/v1alpha1/projects/{project}/workspaces
GET/PUT/DELETE /apis/gpupaas.ai/v1alpha1/projects/{project}/workspaces/{name}
```

## Apply helpers (files)

```go
func ApplyReader(ctx context.Context, c *client.Client, r io.Reader, project, workspace string) error
func DeleteReader(ctx context.Context, c *client.Client, r io.Reader, project, workspace string, ignoreNotFound bool) error
```

Decode multi-doc YAML via scheme, then call `client.Apply` per document.

## Example: typed_client/main.go

Must mirror paasctl go-client-basic:

```go
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
    c := client.New(client.Options{UseMemory: true})
    ctx := context.Background()

    ws := &v1alpha1.Workspace{
        TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindWorkspace},
        Metadata: v1alpha1.ObjectMeta{Name: "example", Project: "demo"},
    }
    applied, err := c.Apply(ctx, ws, "demo", "")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("applied %s/%s\n", applied.GetKind(), applied.GetName())

    gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindWorkspace}
    items, err := c.List(ctx, gvk, "demo", "")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("listed %d workspace(s)\n", len(items))
}
```

## Error handling

- Wrap with `%w` and resource context (`apiVersion/kind/project/workspace/name`)
- Use `APIError` helpers for HTTP failures
- Never log tokens

## Testing strategy

| Area | Approach |
|---|---|
| memory backend | Apply + List integration without network |
| validation | Default project/workspace injection |
| codec | YAML/JSON/multi-doc decode |
| remote backend | httptest + clientset path assertions |
| examples | `go build ./examples/...` in CI |

## Relationship to other repos

```text
gpupaas-go (this SDK)
  ↑ consumed by
  ├── paasctl (kubectl-style CLI)
  └── terraform-provider-gpupaas
```

Pattern reference: `paasctl/pkg/client`, `paasctl/pkg/engine`, `paasctl/examples/go-client-basic`.
