---
name: add-resource-from-swagger
description: >-
  Add or update a resource in the gpupaas-go SDK from a Swagger / OpenAPI
  document. Wires up apis/v1alpha1 types, convert wire types, clientset
  interface and client, remote backend dispatch, tests, examples, and README
  documentation, while preserving the Kubernetes-style envelope and backward
  compatibility. Use when the user provides a swagger/OpenAPI file (or URL) and
  asks to add, regenerate, or update an SDK resource in gpupaas-go.
---

# Add Resource to gpupaas-go from Swagger

## When to apply

Use this skill whenever the user supplies a Swagger / OpenAPI document and
asks to add, regenerate, or update an SDK resource in
`github.com/gpupaas-ai/gpupaas-go`. The skill:

1. Takes the swagger as input.
2. Implements backend interaction (REST paths + wire types + converters +
   clientset).
3. Exposes a Kubernetes-style `apiVersion / kind / metadata / spec / status`
   resource in `apis/v1alpha1` that round-trips to/from the backend wire
   format.
4. If the resource already exists in the SDK, diff the swagger against the
   current SDK and update in-place — **always ask the user before removing or
   renaming any existing field**.
5. Updates `README.md` so the new/changed resource is documented.

## Inputs to gather

Before generating code, confirm:

| Input | Notes |
|------|-------|
| Swagger / OpenAPI source | File path or URL. Note version (2.0 vs 3.x). |
| Resource kind name | PascalCase Go name (e.g. `VirtualMachine`, `LoadBalancer`). |
| Scope | One of: cluster (`metadata.name`), project (`metadata.project` + name), project-or-workspace dev resource, workspace sub-resource. |
| Resource group / API version | Almost always `gpupaas.ai/v1alpha1` on the SDK side; the **backend** wire `apiVersion` is the swagger value (e.g. `dev.envmgmt.io/v1`, `paas.envmgmt.io/v1`). |
| Update semantics | Idempotent `Create` (most dev resources), or explicit `Update`/`PUT` (e.g. `Project`). |
| Sub-resource actions | E.g. `/status`, `/action/{verb}` — these become extra interface methods (see `VirtualMachineInterface.Start/Stop/GetStatus`). |

If the swagger lacks any of these, ask before writing code.

## Architecture map (always)

```text
apis/v1alpha1/                   SDK k8s-style types + register kind
convert/                         Wire types (Dev*/PaaS*/Auth*) + ToX / FromX
convert/paths.go                 Path string constants (one per scope)
clientset/typed/v1alpha1/
  interfaces.go                  Add <Kind>Interface
  client.go                      Wire Interface entrypoints (project scope)
  dev_resource_client.go         Implement client (workspace + project scope)
backend/remote/remote.go         Dispatch in Apply / Get / List / Delete
examples/<verb>-<resource>/      Runnable example
README.md                        Backend translation row + Appendix entry
```

## Workflow

Copy this checklist into the task and tick items off as you go:

```text
- [ ] Step 1 — Parse swagger and produce a resource plan
- [ ] Step 2 — Detect existing SDK resource and diff
- [ ] Step 3 — Add/update apis/v1alpha1 types
- [ ] Step 4 — Register kind in apis/v1alpha1/register.go
- [ ] Step 5 — Add path constants in convert/paths.go
- [ ] Step 6 — Add wire types + ToX/FromX converters in convert/dev_<name>.go
- [ ] Step 7 — Add <Kind>Interface in clientset/typed/v1alpha1/interfaces.go
- [ ] Step 8 — Implement client and wire into Interface/WorkspaceInterface
- [ ] Step 9 — Extend backend/remote/remote.go dispatch
- [ ] Step 10 — Add tests (convert + clientset)
- [ ] Step 11 — Add an examples/ program
- [ ] Step 12 — Update README.md (table + Appendix)
- [ ] Step 13 — Verify: gofmt + go vet + go build + go test
```

---

### Step 1 — Parse swagger and produce a resource plan

Extract:

- Resource kind, list kind, and plural resource name (used in REST paths).
- Backend `apiVersion` and `kind` (from the swagger `definitions` / `components`).
- Operations and HTTP paths. Group by scope:
  - **Project-scoped:** `/apis/<group>/v1/projects/{project}/<plural>`
  - **Workspace-scoped:** `/apis/<group>/v1/projects/{project}/workspaces/{workspace}/<plural>`
  - **Sub-actions:** e.g. `/.../<plural>/{name}/status`, `/.../<plural>/{name}/action/{verb}`
- Spec fields (request bodies on create/update).
- Status fields (response-only fields like `status`, `reason`, `action`,
  `provisionedAt`, computed IDs).
- Sharing block, if present — usually `DevSharingSpec` (`shareMode`,
  `workspaces`, `projects[]`). `VirtualMachine` uses a richer
  `VirtualMachineSharingSpec` — verify against the swagger before reusing
  `DevSharingSpec`.
- Idempotency: if the swagger's create endpoint is documented as upsert (the
  pattern for `Storage`, `SecurityGroup`, `SshKey`, `VirtualMachine`), route
  `Update` through `Create` and document it on the interface.

Output of this step is a short markdown plan: kind, scope, paths, spec fields,
status fields, sub-actions, idempotency. Confirm with the user before writing
code if any of it is non-obvious.

### Step 2 — Detect existing SDK resource and diff

Search the repo:

```bash
rg -n "Kind<KindName>\b" apis/ clientset/ convert/ backend/ README.md
```

If the kind already exists:

1. List existing spec/status fields (`apis/v1alpha1/types_dev.go` or
   `types.go`) and current wire fields (`convert/dev_<name>.go`).
2. Compare against the swagger. Bucket changes as:
   - **Add** — new fields in swagger → safe, append.
   - **Modify** — type change, rename, or constraint tightening → confirm with
     user; provide both old and new field if rename can be aliased.
   - **Remove** — field gone from swagger → **STOP and ask the user**.
     Default to keeping the field for backward compatibility (mark
     `// Deprecated: …` in the Go doc comment and exclude it from new
     converters only after explicit approval).
3. Same diff applies to paths and sub-actions — never drop a method silently.

When in doubt, prefer additive changes and surface a short "proposed removals"
list to the user.

### Step 3 — Add/update `apis/v1alpha1` types

Add a Go file (typically `apis/v1alpha1/types_dev.go` for dev resources; new
file for non-dev groups). Pattern:

```go
// <Kind>Spec holds desired <kind> state.
type <Kind>Spec struct {
    // ... fields from swagger, camelCase JSON, with omitempty on optionals
    Sharing *DevSharingSpec `json:"sharing,omitempty" yaml:"sharing,omitempty"`
}

// <Kind>Status holds observed <kind> state.
type <Kind>Status struct {
    Status string `json:"status,omitempty" yaml:"status,omitempty"`
    Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
    Action string `json:"action,omitempty" yaml:"action,omitempty"`
}

type <Kind> struct {
    TypeMeta `json:",inline" yaml:",inline"`
    Metadata ObjectMeta    `json:"metadata" yaml:"metadata"`
    Spec     <Kind>Spec    `json:"spec,omitempty" yaml:"spec,omitempty"`
    Status   <Kind>Status  `json:"status,omitempty" yaml:"status,omitempty"`
}

func (s *<Kind>) GetAPIVersion() string  { return s.APIVersion }
func (s *<Kind>) GetKind() string        { return s.Kind }
func (s *<Kind>) GetName() string        { return s.Metadata.Name }
func (s *<Kind>) GetProject() string     { return s.Metadata.Project }
func (s *<Kind>) GetWorkspace() string   { return s.Metadata.Workspace }
func (s *<Kind>) SetProject(v string)    { s.Metadata.Project = v }
func (s *<Kind>) SetWorkspace(v string)  { s.Metadata.Workspace = v }
func (s *<Kind>) DeepCopyObject() runtime.Object {
    cp := *s
    cp.Metadata = copyObjectMeta(s.Metadata)
    cp.Spec     = copy<Kind>Spec(s.Spec)
    return &cp
}

type <Kind>List struct {
    TypeMeta `json:",inline" yaml:",inline"`
    Metadata ListMeta  `json:"metadata,omitempty" yaml:"metadata,omitempty"`
    Items    []<Kind>  `json:"items" yaml:"items"`
}
```

Use the kind constant `Kind<Kind>` (defined alongside other constants in
`apis/v1alpha1/types.go`). For cluster-scoped kinds, omit project/workspace
accessors.

Rules:

- JSON tags use camelCase to match the SDK envelope. **Wire** (backend) tags
  may be snake_case — keep those in `convert/`, not in `apis/`.
- Optional fields use `omitempty`.
- Never expose backend-only IDs on the SDK type unless required; if needed,
  carry them via annotations (`gpupaas.ai/<thing>-id`) like `Project` does.

### Step 4 — Register the kind

`apis/v1alpha1/register.go`:

```go
scheme.AddKnownTypeWithName(
    runtime.GroupVersionKind{Group: Group, Version: Version, Kind: Kind<Kind>},
    &<Kind>{},
    runtime.GroupVersionResource{Group: Group, Version: Version, Resource: "<plural>"},
)
```

`Resource` is the lowercase plural used internally; it does **not** have to
match the backend path segment, but conventionally does (`virtualmachines`,
`storages`, `securitygroups`, `sshkeys`).

### Step 5 — Add path constants

`convert/paths.go` — keep grouped per resource, both scopes:

```go
Dev<Group><Kind>Path        = "/apis/<group>/v1/projects/%s/<plural>/%s"
Dev<Group><Kind>sPath       = "/apis/<group>/v1/projects/%s/<plural>"
DevWorkspace<Kind>Path      = "/apis/<group>/v1/projects/%s/workspaces/%s/<plural>/%s"
DevWorkspace<Kind>sPath     = "/apis/<group>/v1/projects/%s/workspaces/%s/<plural>"
// Sub-actions (status, action/<verb>) if present in swagger
```

### Step 6 — Wire types and converters

Create `convert/dev_<kind>.go` modeled on `convert/dev_ssh_key.go`:

- `Dev<Kind>Spec`, `Dev<Kind>Status`, `Dev<Kind>`, `Dev<Kind>List` — fields use
  the swagger's JSON casing (often snake_case).
- `<Kind>Paths(scope DevScope) (collection, item func(...) string)` — returns
  workspace-scoped builders when `scope.Workspace != ""`, project-scoped
  otherwise.
- `ToDev<Kind>(*apiv1.<Kind>, project, workspace string) *Dev<Kind>` — uses
  `devMetadataToWire`. Strip status before write (the client also strips, but
  doing it here is defensive).
- `FromDev<Kind>(*Dev<Kind>, workspace string) *apiv1.<Kind>` — uses
  `devMetadataFromWire`.
- `FromDev<Kind>List(...) *apiv1.<Kind>List` — uses `devListContinue`.
- Reuse `toDevSharing` / `fromDevSharing` for `DevSharingSpec`; introduce a
  dedicated `<Kind>SharingSpec` only when the swagger says so (precedent:
  VirtualMachine).

### Step 7 — Add the typed interface

`clientset/typed/v1alpha1/interfaces.go`:

```go
type <Kind>Interface interface {
    Create(ctx context.Context, obj *apiv1.<Kind>, opts gpupaas.CreateOptions) (*apiv1.<Kind>, error)
    Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.<Kind>, error)
    List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.<Kind>List, error)
    Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
    // Add Update only if the swagger has a dedicated update endpoint that
    // is not just "POST is idempotent". For idempotent POST resources, omit
    // Update and document that callers should re-invoke Create.
    // Add GetStatus / Start / Stop only for resources with /status or
    // /action/{verb} sub-routes.
}
```

If the resource is workspace-scoped (lives under
`/projects/{project}/workspaces/{workspace}/...`) also extend
`WorkspaceInterface`:

```go
type WorkspaceInterface interface {
    // existing methods...
    <Kind>s(workspace string) <Kind>Interface
}
```

### Step 8 — Implement the client

For dev resources reuse `devScopedClient` from
`clientset/typed/v1alpha1/dev_resource_client.go`. Pattern (copy-and-adapt
from `sshKeyClient`):

```go
type <kind>Client struct { devScopedClient }

func new<Kind>Client(rc *rest.Client, project, workspace string) *<kind>Client {
    return &<kind>Client{devScopedClient{rest: rc, project: project, workspace: workspace}}
}

func (c *<kind>Client) Create(ctx context.Context, obj *apiv1.<Kind>, _ gpupaas.CreateOptions) (*apiv1.<Kind>, error) {
    obj.Metadata.Project   = firstNonEmpty(obj.Metadata.Project, c.project)
    obj.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
    wire := convert.ToDev<Kind>(stripStatus<Kind>(obj), c.project, c.workspace)
    collection, _ := convert.<Kind>Paths(c.scope())
    var out convert.Dev<Kind>
    if err := c.rest.Post(ctx, collection(), wire, &out); err != nil {
        return nil, err
    }
    if out.Metadata.Name != "" {
        return convert.FromDev<Kind>(&out, c.workspace), nil
    }
    return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

// Get / List / Delete follow the same shape as sshKeyClient.

func stripStatus<Kind>(obj *apiv1.<Kind>) *apiv1.<Kind> {
    cp := *obj
    cp.Status = apiv1.<Kind>Status{}
    return &cp
}
```

Then wire entrypoints in `clientset/typed/v1alpha1/client.go`:

```go
type Interface interface {
    // existing...
    <Kind>s(project string) <Kind>Interface
}

func (c *Client) <Kind>s(project string) <Kind>Interface {
    return new<Kind>Client(c.rest, project, "")
}
```

And on `workspaceClient`:

```go
func (w *workspaceClient) <Kind>s(workspace string) <Kind>Interface {
    return new<Kind>Client(w.rest, w.project, workspace)
}
```

Rules:

- Always take `ctx context.Context` first.
- Never log secrets or full request bodies from library code.
- Map non-2xx to `APIError` via the existing `rest.Client`. Use
  `gpupaas.IsNotFound` / `IsConflict` in higher layers.
- Strip `Status` before write paths.

### Step 9 — Extend remote backend dispatch

`backend/remote/remote.go` — add cases in `Apply`, `Get`, `List`, `Delete`
(and any sub-action method) mirroring an existing project-or-workspace dev
resource (e.g. `SshKey`):

```go
case *v1alpha1.<Kind>:
    if o.Metadata.Workspace != "" {
        return b.client.V1alpha1().Workspaces(o.Metadata.Project).
            <Kind>s(o.Metadata.Workspace).Create(ctx, o, gpupaas.CreateOptions{})
    }
    return b.client.V1alpha1().<Kind>s(o.Metadata.Project).
        Create(ctx, o, gpupaas.CreateOptions{})
```

### Step 10 — Tests

For each new resource add:

- `convert/dev_<kind>_test.go` — round-trip `apiv1.<Kind>` → wire → `apiv1.<Kind>`,
  plus path builder assertions for both scopes.
- `clientset/typed/v1alpha1/<kind>_client_test.go` — use `httptest.Server`
  fixture (mirror existing `dev_storage_test.go`) to assert request method,
  URL, and decoded body for Create / Get / List / Delete.

Run:

```bash
go test ./convert/... ./clientset/...
```

### Step 11 — Examples

Add `examples/<verb>-<kind>/main.go` for the primary operations (create / get
/ list / delete). Use `-memory` flag pattern so the example runs without
credentials. List the new commands in the README "Examples" block.

### Step 12 — Update `README.md`

Two updates are required:

**a) Backend translation table** — add a row in the table near
"Backend translation":

```text
| <Kind> (project)  | `<Kind>`, `<Kind>List` | `GET/POST /apis/<group>/v1/projects/{project}/<plural>`, `GET/DELETE .../{name}` |
| <Kind> (workspace)| `<Kind>`              | Same operations under `.../projects/{project}/workspaces/{name}/<plural>` |
```

**b) Appendix: Resource reference** — add a section after the existing
resources following this template:

```markdown
---

### <Kind>

**What it is for:** [One-paragraph description sourced from swagger.]

**Scope:** project-or-workspace dev resource — requires `metadata.project`;
set `metadata.workspace` for workspace-scoped objects.

**Backend:** lives under `<group>/v1`. Create is idempotent (POST is upsert).

#### Example

\`\`\`yaml
apiVersion: gpupaas.ai/v1alpha1
kind: <Kind>
metadata:
  name: example
  project: demo
  workspace: dev
spec:
  # ... key fields from swagger
\`\`\`

#### `metadata` fields
[name / project / workspace / labels / annotations table]

#### `spec` fields
[Generated from the swagger — one row per field, with type and description.]

#### `status` fields (read-only)
[Generated from swagger response schema.]
```

Mirror the layout and tone of the existing `Storage` / `SshKey` /
`VirtualMachine` sections — the README is the canonical reference, so be
specific about scope, idempotency, and any sub-actions.

### Step 13 — Verify

Run from `gpupaas-go/`:

```bash
go fmt ./...
go vet ./...
go test ./...
go build ./...
```

All four must pass before reporting completion.

---

## Backward compatibility rules (hard requirements)

1. **Never silently remove an SDK field, method, type, or kind.** If the
   swagger drops a field, ask the user first. Default action when ambiguous:
   keep the field, mark `// Deprecated:` in the doc comment, and stop reading
   it on writes only after explicit approval.
2. **Never rename a public Go identifier.** Add the new name and keep the old
   as an alias / wrapper unless the user accepts the breakage.
3. **Never change JSON tag names.** A renamed swagger field becomes a *new*
   spec field; the old one stays until the user approves removal.
4. **Never change resource scoping** (cluster ↔ project ↔ workspace) on an
   existing kind without confirming with the user — it breaks every caller's
   `Apply(ctx, obj, project, workspace)` invocation.
5. **Status is observed-only.** Always strip `Status` before write paths;
   never derive `spec` defaults from `status`.

## Anti-patterns

- Do **not** import `k8s.io/client-go` or anything from `k8s.io/apimachinery`.
- Do **not** put HTTP logic in `apis/v1alpha1/`. All wire concerns live in
  `convert/` and `rest/`.
- Do **not** introduce a new top-level package per resource. Extend the
  existing `convert/`, `clientset/typed/v1alpha1/`, and `backend/remote/`.
- Do **not** auto-generate from swagger without integrating the result into
  the patterns above. Treat swagger as the *input*, not the *output format*.
- Do **not** add CLI or Terraform-specific logic — those belong to
  `gpupaasctl` and `terraform-provider-gpupaas` respectively.

## Reference files (read these to mirror style)

- `apis/v1alpha1/types_dev.go` — type layout, sharing spec, deep-copy.
- `apis/v1alpha1/register.go` — kind registration.
- `convert/dev_ssh_key.go` — minimal project-or-workspace converter.
- `convert/dev_virtual_machine.go` — converter with sub-actions and
  resource-specific sharing spec.
- `clientset/typed/v1alpha1/interfaces.go` — interface shape.
- `clientset/typed/v1alpha1/dev_resource_client.go` — client implementation.
- `backend/remote/remote.go` — backend dispatch.
- `README.md` — Appendix tone and per-resource depth.
