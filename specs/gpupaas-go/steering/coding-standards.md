---
inclusion: auto
---

# gpupaas-go — Coding Standards

## Go conventions

- Go 1.22+, idiomatic Go
- `gofmt` enforced; run `go fmt ./...` before commit
- Context as first parameter for blocking operations
- Errors as last return value; wrap with `%w`

## Package organization

| Package | Responsibility |
|---|---|
| Root (`gpupaas`) | Config, options, error helpers |
| `client` | **Primary public API** — `New`, Apply, Get, List, Delete |
| `engine` | Validation + backend orchestration |
| `validation` | Scope defaults (`DefaultProject`, `DefaultWorkspace`) |
| `backend` | Backend interface; `memory/` and `remote/` implementations |
| `apis/v1alpha1` | API types only — no HTTP |
| `runtime` | Scheme, codec, `GroupVersionKind`, Object interface |
| `rest` | HTTP transport |
| `clientset` | Typed REST clients (used by remote backend) |
| `apply` | ApplyFile / ApplyReader helpers over `client.Client` |
| `watch` | Event types |
| `internal/` | Test utilities, non-public helpers |

## Naming

- Exported types: PascalCase (`Project`, `APIError`)
- Interfaces: behavior suffix (`Backend`, `Object`)
- Files: snake_case for multi-word (`workspace.go`)
- JSON/YAML tags: camelCase matching API (`displayName`)

## API types

Every resource struct:

```go
type Project struct {
    TypeMeta   `json:",inline" yaml:",inline"`
    Metadata   ObjectMeta `json:"metadata" yaml:"metadata"`
    Spec       ProjectSpec `json:"spec" yaml:"spec"`
    Status     ProjectStatus `json:"status,omitempty" yaml:"status,omitempty"`
}
```

Implement full `Object` interface on each type:

```go
GetAPIVersion() string
GetKind() string
GetName() string
GetProject() string
GetWorkspace() string
SetProject(string)
SetWorkspace(string)
DeepCopyObject() Object
```

## Client API

Match paasctl signatures:

```go
func (c *Client) Apply(ctx context.Context, obj runtime.Object, project, workspace string) (runtime.Object, error)
func (c *Client) List(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace string) ([]runtime.Object, error)
```

- `List` and `Get` require non-empty `project`
- Examples default to `client.Options{UseMemory: true}`

## Platform scope in metadata

- `Project`: `metadata.name` only
- `Workspace`: `metadata.project` + `metadata.name`
- Future kinds: `metadata.project` + `metadata.workspace` + `metadata.name`

## Documentation

- Package comment in each public package
- Godoc on exported symbols
- Explain "why" in comments, not "what"

## Imports

1. Standard library
2. External (`gopkg.in/yaml.v3`)
3. Internal packages

## Testing

- Table-driven tests with subtests
- Use `backend/memory` for client and engine tests without network
- Use `httptest` for remote backend and clientset tests
- No real API credentials in unit tests
- Test files: `*_test.go` alongside implementation

Minimum coverage areas:

- YAML/JSON/multi-doc decode
- Unknown kind and validation errors
- Validation default project injection
- Memory backend apply create/update/list/delete
- Client Apply + List (go-client-basic flow)
- REST success and error paths
- Apply-from-file create, update, delete ignore-not-found

## Security

- Never log or print `Token` or `Authorization` headers
- Do not include secrets in error messages
- Normalize endpoint (trim trailing slash) in config

## Dependencies

Keep minimal:

- `gopkg.in/yaml.v3` for YAML
- Standard library for HTTP

Avoid: client-go, heavy REST frameworks, global state.

## Public API stability

- Group: `gpupaas.ai/v1alpha1` — may change until beta
- Prefer additive changes within v1alpha1
- Document breaking changes in README

## Code review checklist

- [ ] Normal flows go through `client/`, not raw clientset
- [ ] No HTTP logic outside `rest/`, `clientset/`, `backend/remote`
- [ ] All public funcs accept `context.Context` where blocking
- [ ] Errors wrapped with resource context
- [ ] Status not sent on create/update from apply
- [ ] `url.PathEscape` used for URL path segments
- [ ] Tests pass: `go test ./...`
- [ ] Examples compile: `go build ./examples/...`
- [ ] `examples/typed_client` matches paasctl go-client-basic shape
