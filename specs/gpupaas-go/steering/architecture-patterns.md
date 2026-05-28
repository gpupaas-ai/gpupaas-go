---
inclusion: auto
---

# gpupaas-go — Architecture Patterns

## Layering

Modeled on [`paasctl/examples/go-client-basic`](https://github.com/RafaySystems/paasctl/blob/main/examples/go-client-basic/main.go) and `paasctl/pkg/client`:

```text
┌─────────────────────────────────────────┐
│  Consumers: CLI, Terraform, user code   │
└────────────────────┬────────────────────┘
                     ↓
┌─────────────────────────────────────────┐
│  client/         Primary public API     │
│  Apply, Get, List, Delete               │
│  Options{UseMemory}                     │
└────────────────────┬────────────────────┘
                     ↓
┌─────────────────────────────────────────┐
│  engine/         Orchestration          │
│  validation → backend                   │
└────────────────────┬────────────────────┘
                     ↓
┌─────────────────────────────────────────┐
│  backend/        Pluggable storage      │
│  memory/ | remote/                      │
└────────────────────┬────────────────────┘
                     ↓
┌─────────────────────────────────────────┐
│  clientset/ + rest/   (remote only)     │
└────────────────────┬────────────────────┘
                     ↓
              gpupaas.ai API
```

Supporting packages:

```text
apis/v1alpha1/  → API structs and GVK registration
runtime/        → Scheme, codec, GroupVersionKind, Object
validation/     → DefaultProject, DefaultWorkspace, kind rules
apply/          → ApplyFile / ApplyReader helpers (thin wrapper over client)
watch/          → Event types (minimal in v1)
```

## Platform scoping

Resources use a Kubernetes-like envelope (`apiVersion`, `kind`, `metadata`, `spec`, `status`) but **all objects are native to the gpupaas platform**:

```text
Project          → metadata.name
Workspace        → metadata.project + metadata.name
Future resources → metadata.project + metadata.workspace + metadata.name
```

Method arguments mirror metadata:

- `Apply(ctx, obj, project, workspace)` — fills empty `metadata.project` / `metadata.workspace`
- `List(ctx, gvk, project, workspace)` — `project` required

## Resource identity

```text
apiVersion + kind + project + workspace + name
```

Omit `project`/`workspace` from identity when the kind does not use them (e.g. `Project` uses name only).

## High-level client pattern (preferred)

```go
c := client.New(client.Options{UseMemory: true})
ctx := context.Background()

ws := &v1alpha1.Workspace{
    TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindWorkspace},
    Metadata: v1alpha1.ObjectMeta{Name: "example", Project: "demo"},
}
applied, err := c.Apply(ctx, ws, "demo", "")

gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindWorkspace}
items, err := c.List(ctx, gvk, "demo", "")
```

`UseMemory: true` selects `backend/memory` for examples and unit tests without credentials.

For production:

```go
c := client.New(client.Options{Config: gpupaas.ConfigFromEnv()})
```

## Engine + validation pattern

```go
// Inside client.Apply:
validation.ValidateObject(obj, validation.Options{
    DefaultProject:   project,
    DefaultWorkspace: workspace,
})
return e.backend.Apply(ctx, obj)
```

Validation rules (aligned with paasctl):

1. Require `apiVersion`, `kind`, `metadata.name`
2. Set `metadata.project` from `DefaultProject` when empty; error if still empty
3. For `Workspace`, stop after project is set
4. For future project+workspace kinds, set `metadata.workspace` from `DefaultWorkspace` when empty

## Typed clientset pattern (remote backend internals)

Direct clientset use is for advanced callers and remote backend implementation:

```go
cs, err := clientset.NewForConfig(config)
proj, err := cs.V1alpha1().Projects().Create(ctx, obj, gpupaas.CreateOptions{})
ws, err := cs.V1alpha1().Workspaces("demo").Get(ctx, "dev", gpupaas.GetOptions{})
```

Each resource interface follows the same CRUD shape for consistency and testability.

## Apply-from-file pattern

```go
c := client.New(client.Options{UseMemory: true})
err := apply.ApplyReader(ctx, c, reader, "demo", "")
```

Semantics:

1. Decode object(s) from reader via `c.Scheme`
2. Validate and apply scope defaults per document
3. Create or update via backend; never mutate status from desired state

## REST client pattern

- All methods take `context.Context` first
- JSON request/response bodies
- Bearer token when configured
- Non-2xx → `APIError` with helpers

Do NOT log tokens. Do NOT print from library code.

## Error handling pattern

```go
if gpupaas.IsNotFound(err) {
    // handle missing resource
}
```

Wrap internal errors with platform scope:

```go
return fmt.Errorf("apply Workspace project=demo name=dev: %w", err)
```

## Multi-document YAML

Split on `---` (Kubernetes-style document separators). Apply each document in order. Include document index in decode error messages.

## Extensibility

To add a new resource kind:

1. Add types in `apis/v1alpha1/<kind>.go` with correct `metadata.project` / `metadata.workspace` rules
2. Register in `register.go`
3. Extend `backend/memory` and `backend/remote` dispatch
4. Add typed client in `clientset/typed/v1alpha1/`
5. Add tests and example YAML

## Anti-patterns

- Do NOT import `k8s.io/client-go`
- Do NOT require callers to use clientset directly for normal Apply/List flows
- Do NOT map platform scope to foreign tenancy models in docs or APIs
- Do NOT put CLI or Terraform logic in the SDK
- Do NOT use process-wide mutable HTTP clients (pass config per clientset)
- Do NOT expose raw filesystem or HTTP errors without wrapping
- Do NOT implement watch streaming in v1 unless types are stable

## Reference

- `paasctl/examples/go-client-basic/main.go` — canonical consumer example
- `paasctl/pkg/client`, `paasctl/pkg/engine`, `paasctl/pkg/validation`, `paasctl/pkg/backend/memory`
- `paasctl/specs/kubectl_style_cli_architecture_prompt.md` — broader kubectl-style framework
