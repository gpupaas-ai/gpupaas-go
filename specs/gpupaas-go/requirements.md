# Requirements: gpupaas-go SDK

## Introduction

`github.com/gpupaas-ai/gpupaas-go` is the public Go SDK for the gpupaas.ai API. It is the shared client layer for the CLI (`paasctl`), the Terraform/OpenTofu provider (`terraform-provider-gpupaas`), and external automation programs.

The SDK uses a Kubernetes-inspired resource shape (`apiVersion`, `kind`, `metadata`, `spec`, `status`) and client layering modeled after [`paasctl/examples/go-client-basic`](https://github.com/RafaySystems/paasctl/blob/main/examples/go-client-basic/main.go), but all objects are native to the GPU PaaS platform.

## Platform scoping model

All GPU PaaS resources live on the gpupaas platform with explicit platform scope:

| Level | Meaning |
|---|---|
| **Project** | Top-level platform container. Identified by `metadata.name` only. |
| **Workspace** | Belongs to a project. Identified by `metadata.project` + `metadata.name`. |
| **Future resources** | Belong to a project and workspace. Identified by `metadata.project` + `metadata.workspace` + `metadata.name`. |

Projects contain workspaces. Workspaces contain (or will contain) workload resources. Scope is expressed only with `metadata.project` and `metadata.workspace`.

API calls also accept explicit `project` and `workspace` arguments (defaults/filters), matching paasctl's `Apply(ctx, obj, project, workspace)` and `List(ctx, gvk, project, workspace)` pattern.

## Glossary

- **Resource**: A declarative GPU PaaS object with `apiVersion`, `kind`, `metadata`, `spec`, and optional `status`
- **Client**: High-level SDK entry point (`client.New`) exposing Apply, Get, List, Delete
- **Engine**: Orchestrates validation and backend operations
- **Backend**: Pluggable storage/API layer (`memory` for tests, `remote` for gpupaas.ai REST)
- **Scheme**: Registry mapping `apiVersion + kind` to Go types; handles encode/decode
- **Clientset**: Low-level typed REST client used by the remote backend
- **Project scope**: The owning project name in `metadata.project` or the `project` method argument
- **Workspace scope**: The owning workspace name in `metadata.workspace` or the `workspace` method argument

## Requirements

### Requirement 1: Resource model

**User Story:** As an SDK consumer, I want declarative resource types so that CLI, Terraform, and Go programs share one platform-native model.

#### Acceptance Criteria

1. WHEN defining API types in `apis/v1alpha1`, THE SDK SHALL provide `TypeMeta`, `ObjectMeta`, and resource structs with inline `TypeMeta`, `Metadata`, `Spec`, and optional `Status`
2. `ObjectMeta` SHALL include `name`, optional `project`, optional `workspace`, `labels`, and `annotations`
3. THE SDK SHALL implement initial kinds: `Project`, `Workspace` at `apiVersion: gpupaas.ai/v1alpha1`
4. WHEN a resource is a `Workspace`, THE SDK SHALL require `metadata.project` (from the object or from Apply/List defaults)
5. WHEN a resource is a `Project`, THE SDK SHALL NOT require `metadata.project` or `metadata.workspace`
6. THE SDK SHALL expose list types (`ProjectList`, `WorkspaceList`, etc.) for list operations
7. EACH resource type SHALL implement `runtime.Object` including `GetProject`, `GetWorkspace`, `SetProject`, `SetWorkspace`, and `DeepCopyObject`

### Requirement 2: Runtime scheme and codec

**User Story:** As an SDK consumer, I want to decode YAML/JSON specs into typed objects so that apply workflows work from files and streams.

#### Acceptance Criteria

1. THE `runtime` package SHALL register constructors by `apiVersion` and `kind`
2. THE `runtime` package SHALL define `GroupVersionKind` with `Group`, `Version`, and `Kind` fields
3. WHEN decoding YAML or JSON, THE scheme SHALL return `runtime.Object` or a clear error for unknown types
4. THE codec SHALL support single-document JSON, single-document YAML, and multi-document YAML separated by `---`
5. WHEN required fields are missing (`apiVersion`, `kind`, `metadata.name`, and scope fields per kind), validation SHALL return errors with field context
6. THE scheme SHALL encode objects back to JSON and YAML

### Requirement 3: High-level client

**User Story:** As an SDK consumer, I want a simple client API like paasctl's go-client-basic example so that I can apply and list resources in a few lines of Go.

#### Acceptance Criteria

1. THE `client` package SHALL expose `New(client.Options) *Client`
2. `client.Options` SHALL support `UseMemory bool` for in-memory backend (examples and unit tests without a live API)
3. WHEN `UseMemory` is false, THE client SHALL use a remote backend configured from `gpupaas.Config` / env vars
4. THE client SHALL expose:
   - `Apply(ctx, obj runtime.Object, project, workspace string) (runtime.Object, error)`
   - `Get(ctx, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error)`
   - `List(ctx, gvk runtime.GroupVersionKind, project, workspace string) ([]runtime.Object, error)`
   - `Delete(ctx, gvk runtime.GroupVersionKind, project, workspace, name string) error`
5. THE client SHALL expose `Scheme *runtime.Scheme` for decode/encode helpers
6. `List` and `Get` SHALL require a non-empty `project` argument (same as paasctl)

Reference program (MUST compile and match this shape):

```go
c := client.New(client.Options{UseMemory: true})
ws := &v1alpha1.Workspace{
    TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindWorkspace},
    Metadata: v1alpha1.ObjectMeta{Name: "example", Project: "demo"},
}
applied, err := c.Apply(ctx, ws, "demo", "")
items, err := c.List(ctx, gvk, "demo", "")
```

### Requirement 4: Validation

**User Story:** As an SDK consumer, I want scope defaults applied consistently so that manifests and CLI flags behave the same way.

#### Acceptance Criteria

1. THE `validation` package SHALL validate `apiVersion`, `kind`, and `metadata.name`
2. WHEN `metadata.project` is empty on apply, THE validator SHALL set it from the `project` argument (`DefaultProject`)
3. WHEN `metadata.project` is still empty after defaults, THE validator SHALL return an error
4. FOR `Workspace`, THE validator SHALL NOT require `metadata.workspace`
5. FOR future project+workspace-scoped kinds, THE validator SHALL set `metadata.workspace` from the `workspace` argument when empty

### Requirement 5: REST transport and typed clientset

**User Story:** As an SDK consumer, I want a thin HTTP client and typed REST accessors used internally by the remote backend.

#### Acceptance Criteria

1. THE `rest` package SHALL wrap `net/http` with context-aware methods and Bearer auth
2. THE `clientset` SHALL expose `Projects()` and `Workspaces(project)` with Create, Get, Update, Delete, List
3. WHEN calling workspace APIs, paths SHALL be `/apis/gpupaas.ai/v1alpha1/projects/{project}/workspaces/...`
4. WHEN calling project APIs, paths SHALL be `/apis/gpupaas.ai/v1alpha1/projects/...`
5. THE remote backend SHALL delegate to the clientset; callers SHOULD prefer `client.Client` over direct clientset use

### Requirement 6: Declarative file apply and delete

**User Story:** As an SDK consumer, I want apply and delete from YAML/JSON files so that automation matches CLI `-f` semantics.

#### Acceptance Criteria

1. THE `apply` package (or client helpers) SHALL support `ApplyFile`, `ApplyReader`, `DeleteFile`, and `DeleteReader`
2. WHEN applying an object that does not exist, THE backend SHALL create it
3. WHEN applying an object that exists, THE backend SHALL update it
4. WHEN applying or deleting, THE SDK SHALL NOT modify `status` from the desired object
5. WHEN deleting with `IgnoreNotFound=true` and the object is missing, THE SDK SHALL return no error

### Requirement 7: Configuration and errors

**User Story:** As an SDK consumer, I want config from env vars and typed errors so that integration is straightforward.

#### Acceptance Criteria

1. THE root package SHALL expose `Config`, `NewConfig`, and `ConfigFromEnv` reading `GPUPAAS_ENDPOINT` and `GPUPAAS_TOKEN`
2. THE default endpoint SHALL be `https://api.gpupaas.ai`
3. THE SDK SHALL expose `APIError` with `IsNotFound`, `IsConflict`, `IsUnauthorized`, `IsForbidden`, and `IsServerError`
4. THE SDK SHALL expose package-level helpers using `errors.As`

### Requirement 8: Testability

**User Story:** As a contributor, I want unit tests and examples without a live API so that CI is reliable.

#### Acceptance Criteria

1. THE SDK SHALL provide `backend/memory` for in-memory apply/list/get/delete
2. THE SDK SHALL provide `rest/fake.go` or httptest patterns for remote backend tests
3. WHEN running `go test ./...`, ALL unit tests SHALL pass without real credentials
4. `examples/typed_client` SHALL use `client.Options{UseMemory: true}` by default

### Requirement 9: Examples and documentation

**User Story:** As a new user, I want compiling examples and a README so that I can adopt the SDK quickly.

#### Acceptance Criteria

1. THE repo SHALL include `examples/typed_client` modeled on `paasctl/examples/go-client-basic/main.go`
2. THE repo SHALL include `examples/apply_yaml` and `examples/delete_yaml` that compile
3. THE README SHALL document the high-level client, env vars, platform scoping, YAML apply/delete, and relationships to CLI and Terraform provider

## Out of scope (v1)

- Full watch/streaming implementation (types and fake watcher only)
- Dynamic/unstructured client (future)
- Kubernetes client-go dependency
- CLI implementation in this repo
- Workload resource kinds beyond Project and Workspace (future API versions)

## Acceptance criteria

```bash
go mod tidy
go fmt ./...
go test ./...
go build ./examples/...
```

The `examples/typed_client` program SHALL mirror paasctl `go-client-basic`: apply a Workspace under project `demo`, then list workspaces in that project.

## Dependency order

**gpupaas-go MUST be implemented before paasctl and terraform-provider-gpupaas consume it.**

Reference implementation pattern: `paasctl/pkg/client`, `paasctl/pkg/engine`, `paasctl/pkg/validation`, `paasctl/pkg/backend/memory`.
