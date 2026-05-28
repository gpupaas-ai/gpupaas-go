# gpupaas-go

Go SDK for the [gpupaas.ai](https://gpupaas.ai) API. Shared client library for **terraform-provider-gpupaas**, and custom automation.

The API uses a Kubernetes-inspired resource shape (`apiVersion`, `kind`, `metadata`, `spec`, `status`) with gpupaas-native scoping:

- **Project** — top-level container (`metadata.name`)
- **Workspace** — belongs to a project (`metadata.project` + `metadata.name`)
- **VirtualMachine** — dev VM instance at **project** or **workspace** scope (`metadata.project`, optional `metadata.workspace`)

## Install

```bash
go get github.com/gpupaas-ai/gpupaas-go
```

## Quick start

Mirrors [`paasctl/examples/go-client-basic`](https://github.com/RafaySystems/paasctl/blob/main/examples/go-client-basic/main.go):

```go
c := client.New(client.Options{UseMemory: true})
ctx := context.Background()

ws := &v1alpha1.Workspace{
    TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindWorkspace},
    Metadata: v1alpha1.ObjectMeta{Name: "example", Project: "demo"},
}
applied, err := c.Apply(ctx, ws, "demo", "")
items, err := c.List(ctx, gvk, "demo", "")
```

Use `UseMemory: false` with `Config: gpupaas.ConfigFromEnv()` for the live API.

## Backend translation

The public SDK speaks **Kubernetes-style** `gpupaas.ai/v1alpha1` objects. The remote backend uses different REST shapes; the clientset translates automatically:

| Resource | SDK (k8s-style) | Backend API |
|----------|-----------------|-------------|
| Project list/get | `Project`, `ProjectList` | `GET /auth/v1/projects/` then `GET /auth/v1/projects/{id}/` by resolved id |
| Workspace list/apply/get | `Workspace`, `WorkspaceList` | `GET/POST /apis/paas.envmgmt.io/v1/projects/{project}/workspaces` |
| Workspace collaborator | `WorkspaceCollaborator` | `GET/POST .../workspaces/{name}/collaborators`, `POST .../assigncollaborators`, `POST .../unassigncollaborators` |
| VirtualMachine (project) | `VirtualMachine` | `GET/POST /apis/dev.envmgmt.io/v1/projects/{project}/virtualmachines`, `GET/DELETE .../{name}`, `GET .../status`, `POST .../action/{action}` |
| VirtualMachine (workspace) | `VirtualMachine` | Same operations under `.../projects/{project}/workspaces/{name}/virtualmachines` |
| BaremetalMachine | `BaremetalMachine`, `BaremetalMachineList`, `BaremetalMachineInfo`, `BaremetalConsoleSession` | `GET/POST /apis/infra.k8smgmt.io/v3/projects/{project}/baremetalmachines`, `GET/DELETE .../{name}`, `GET .../powerOn`, `GET .../powerOff`, `GET .../reboot`, `GET .../provision`, `POST .../reinstallOS`, `POST .../consoleSessions`, `GET .../status` |
| MKSCluster | `MKSCluster`, `MKSClusterList` | `GET/POST /apis/paas.envmgmt.io/v1/projects/{project}/mksclusters`, `GET/DELETE .../{name}`, `POST .../upgrade`, `POST .../scaleNodeGroup`, `POST .../addNodeGroup`, `POST .../removeNodeGroup` |
| MKSNode (cluster sub-resource) | `MKSNode`, `MKSNodeList` | `GET .../mksclusters/{cluster}/mksnodes`, `GET/DELETE .../{name}`, `POST .../drain`, `POST .../cordon`, `POST .../uncordon` |
| MKSWorkerNodeGroup (cluster sub-resource) | `MKSWorkerNodeGroup`, `MKSWorkerNodeGroupList` | `GET/POST .../mksclusters/{cluster}/workernodegroups`, `GET/DELETE .../{name}` |
| MKSAuditEvent (cluster sub-resource, read-only) | `MKSAuditEvent`, `MKSAuditEventList` | `GET .../mksclusters/{cluster}/auditevents`, `GET .../{id}` |

Conversion lives in the `convert/` package (`ToAuthProject`, `FromPaaSWorkspace`, etc.). Verbose logging shows wire HTTP payloads; returned objects are always normalized to `gpupaas.ai/v1alpha1`.

See the [OpenAPI explorer](https://console.gpupaas.ai/openapi-explorer) for workspace apply, list, detail, and extension APIs.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GPUPAAS_ENDPOINT` | `https://api.gpupaas.ai` | API base URL |
| `GPUPAAS_API_KEY` | — | Rafay API key (`X-API-KEY`, same as paasctl `api_key`) |
| `GPUPAAS_API_SECRET` | — | API secret for HMAC signing (defaults to API key if unset) |
| `GPUPAAS_VERBOSE` | — | Log HTTP traffic when set (`1`, `true`) |
| `GPUPAAS_TOKEN` | — | Deprecated alias for `GPUPAAS_API_KEY` |

Authentication matches [paasctl](https://github.com/RafaySystems/paasctl): `X-API-KEY`, `X-RAFAY-API-KEYID`, and HMAC-SHA256 request signing (`content-md5`, `date`, `host`, `nonce`). There is no `Authorization: Bearer` header.

```go
cfg := gpupaas.ConfigFromEnv()
c := client.New(client.Options{Config: cfg, Verbose: true})
```

## Apply YAML

```go
c := client.New(client.Options{UseMemory: true})
err := apply.ApplyFile(ctx, c, "workspace.yaml", "demo", "")
```

Multi-document YAML (`---`) is supported.

## Error handling

```go
if gpupaas.IsNotFound(err) {
    // handle missing resource
}
```

## Examples

```bash
go run ./examples/typed_client
go run ./examples/workspace -memory
go run ./examples/workspace-collaborator -memory -project demo -workspace dev
GPUPAAS_API_KEY=... GPUPAAS_API_SECRET=... go run ./examples/workspace-collaborator -project demo -workspace dev -v
go run ./examples/list-vms -memory -project demo
go run ./examples/create-vm -memory -project demo -workspace dev -name example-vm
go run ./examples/get-vm -memory -project demo -name example-vm
go run ./examples/delete-vm -memory -project demo -name example-vm
GPUPAAS_API_KEY=... go run ./examples/vm-start -project demo -workspace dev -name example-vm
GPUPAAS_API_KEY=... go run ./examples/vm-stop -project demo -workspace dev -name example-vm
go run ./examples/create-baremetal -memory -project demo -name example-bm
go run ./examples/get-baremetal -memory -project demo -name example-bm
go run ./examples/list-baremetals -memory -project demo
go run ./examples/delete-baremetal -memory -project demo -name example-bm
GPUPAAS_API_KEY=... go run ./examples/baremetal-reboot -project demo -name example-bm
GPUPAAS_API_KEY=... go run ./examples/baremetal-reinstall-os -project demo -name example-bm -image-url http://images/example.qcow2
GPUPAAS_API_KEY=... go run ./examples/baremetal-console-session -project demo -name example-bm -compute-id compute-123
GPUPAAS_API_KEY=... go run ./examples/baremetal-status -project demo -name example-bm
go run ./examples/create-mks-cluster -memory -project demo -name my-cluster
go run ./examples/get-mks-cluster -memory -project demo -name my-cluster
go run ./examples/list-mks-clusters -memory -project demo
go run ./examples/delete-mks-cluster -memory -project demo -name my-cluster
GPUPAAS_API_KEY=... go run ./examples/mks-cluster-upgrade -project demo -name my-cluster -k8s-version 1.32
GPUPAAS_API_KEY=... go run ./examples/mks-cluster-scale-nodegroup -project demo -name my-cluster -node-group wng-1 -desired-size 5
GPUPAAS_API_KEY=... go run ./examples/list-mks-nodes -project demo -name my-cluster
GPUPAAS_API_KEY=... go run ./examples/mks-node-drain -project demo -name my-cluster -node master-1
GPUPAAS_API_KEY=... go run ./examples/list-mks-worker-node-groups -project demo -name my-cluster
GPUPAAS_API_KEY=... go run ./examples/list-mks-audit-events -project demo -name my-cluster
go run ./examples/apply_yaml ./manifest.yaml demo
go run ./examples/delete_yaml ./manifest.yaml demo
```

## Related projects

| Project | Role |
|---------|------|
| [terraform-provider-gpupaas](https://github.com/gpupaas-ai/terraform-provider-gpupaas) | Terraform provider (consumer) |
| [paasctl](https://github.com/RafaySystems/paasctl) | Reference client architecture |

## Stability

API group `gpupaas.ai/v1alpha1` may change until a beta or stable release.

## Adding a resource from swagger

When extending the SDK with a new resource from a Swagger / OpenAPI document,
follow [`specs/add-resource-from-swagger/SKILL.md`](specs/add-resource-from-swagger/SKILL.md).
It documents the apis/v1alpha1 → convert → clientset → backend/remote pipeline,
backward-compatibility rules (never silently remove fields, methods, or
kinds — always confirm with the user first), the README updates required for
every new resource, and the `go fmt / vet / test / build` verification gate.

## Appendix: Resource reference

All resources use the Kubernetes-style envelope:

| Section | Purpose |
|---------|---------|
| `apiVersion` / `kind` | Identifies the type (`gpupaas.ai/v1alpha1`, `Project` or `Workspace`) |
| `metadata` | Identity, scope, labels, and annotations |
| `spec` | **Desired state** — what you set on create/apply |
| `status` | **Observed state** — populated by the platform on read; do not set on apply |

Platform hierarchy:

```text
Organization (implicit — from your API token)
└── Project          metadata.name
    ├── VirtualMachine    project-scoped (metadata.project only)
    ├── BaremetalMachine  project-scoped (metadata.project only)
    ├── MKSCluster        project-scoped (metadata.project only)
    │   ├── MKSNode             cluster sub-resource
    │   ├── MKSWorkerNodeGroup  cluster sub-resource
    │   └── MKSAuditEvent       cluster sub-resource (read-only)
    └── Workspace    metadata.project + metadata.name
        ├── WorkspaceCollaborator
        └── VirtualMachine   workspace-scoped (metadata.project + metadata.workspace)
```

Method arguments mirror metadata scope: `Apply(ctx, obj, project, workspace)` and `List(ctx, gvk, project, workspace)`.

---

### Project

**What it is for:** A **Project** is the top-level container on the gpupaas platform. It groups workspaces and isolates resources for a team, product line, or environment boundary within your organization. Every workspace belongs to exactly one project.

**Scope:** Cluster-scoped in SDK terms — identified by `metadata.name` only. Do not set `metadata.project` or `metadata.workspace` on a Project.

**Backend:** Listed via `GET /auth/v1/projects/`. **Get by name** lists projects, matches `results[].name`, then fetches `GET /auth/v1/projects/{id}/` using the internal project id (not the name). The auth id is stored on the k8s object as annotation `gpupaas.ai/project-id`.

#### Example

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: Project
metadata:
  name: demo
spec:
  displayName: Demo Project
  description: Shared GPU development environment
  default: false
```

#### `metadata` fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Unique project identifier (used in URLs and as `{project}` in workspace paths) |
| `labels` | no | Key/value tags for selection or tooling |
| `annotations` | no | Non-identifying metadata for integrations |

#### `spec` fields

| Field | Type | Description |
|-------|------|-------------|
| `displayName` | string | Human-readable title shown in the console |
| `description` | string | Short summary of the project's purpose |
| `default` | bool | When `true`, marks this as the organization's default project (maps to auth API `default`) |

#### `status` fields (read-only)

| Field | Description |
|-------|-------------|
| `phase` | High-level observed state (e.g. `Default` when `spec.default` is true) |

---

### Workspace

**What it is for:** A **Workspace** is a logical partition inside a project where GPU workloads run. Use workspaces to separate dev/staging/prod, teams, or experiments while sharing the same project-level settings. Compute instances, services, and other PaaS objects are attached to a workspace (see [Workspace Ext API](https://console.gpupaas.ai/openapi-explorer#/Workspace%20Ext%20ApI) in the OpenAPI explorer).

**Scope:** Project-scoped — requires `metadata.name` and `metadata.project` (the owning project). The `project` argument to `Apply` / `List` fills `metadata.project` when it is empty.

**Backend:** Apply, list, get, and delete use `paas.envmgmt.io/v1` paths under `/apis/paas.envmgmt.io/v1/projects/{project}/workspaces`. Apply is **POST** (upsert); the SDK sends the paas wire format and returns a normalized `gpupaas.ai/v1alpha1` object.

#### Example

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: Workspace
metadata:
  name: dev
  project: demo
  labels:
    env: development
spec:
  displayName: Development
  description: Interactive GPU development workspace
  iconURL: https://example.com/icon.png
  readme: |
    ## Getting started
    Launch a compute instance from the console or API.
```

#### `metadata` fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Unique workspace name within the project |
| `project` | yes | Owning project name (must match the project in the API path) |
| `labels` | no | Key/value tags (passed through to the paas API) |
| `annotations` | no | Non-identifying metadata (passed through to the paas API) |

#### `spec` fields

| Field | Type | Description |
|-------|------|-------------|
| `displayName` | string | Human-readable name shown in the console (maps to paas `metadata.displayName`) |
| `description` | string | Purpose or usage notes for this workspace (maps to paas `metadata.description`) |
| `iconURL` | string | URL of an icon for catalog or UI display |
| `readme` | string | Markdown or plain-text guidance for workspace users |

#### `status` fields (read-only)

| Field | Description |
|-------|-------------|
| `phase` | Observed condition from the backend (e.g. `StatusOK`, `StatusSubmitted`, `StatusFailed`) — sourced from paas `status.commonStatus.conditionStatus` |

#### Related backend operations

Workspace sub-resources are documented in the OpenAPI explorer:

- [Workspace List](https://console.gpupaas.ai/openapi-explorer#/Workspace%20List)
- [Workspace Apply](https://console.gpupaas.ai/openapi-explorer#/Workspace%20Apply)
- [Workspace Detail](https://console.gpupaas.ai/openapi-explorer#/Workspace%20Detail)
- [Workspace Ext API](https://console.gpupaas.ai/openapi-explorer#/Workspace%20Ext%20ApI) — collaborators, compute instances, services

---

### WorkspaceCollaborator

**What it is for:** A **WorkspaceCollaborator** grants a user access to a workspace. Use it to invite external users (not yet in the Rafay console) or assign existing Rafay/SSO users with a workspace access level.

**Scope:** Requires `metadata.name`, `metadata.project`, and `metadata.workspace`. Pass `project` and `workspace` to `Apply` / `List` / `Delete` when those metadata fields are empty.

**Backend mapping** ([Workspace Ext API](https://console.gpupaas.ai/openapi-explorer#/Workspace%20Ext%20ApI)):

| SDK operation | Condition | Backend call |
|---------------|-----------|--------------|
| Apply (add) | `spec.email` set | `POST .../collaborators` with paas `WorkspaceCollaborator` body |
| Apply (add) | `spec.username` or name only (no email) | `POST .../assigncollaborators?ssoUsers=true` when `spec.isSSOUser` is true |
| List | — | `GET .../collaborators` (optional `?ssoUsers=true` via list options) |
| Get | — | List + match by name, username, or email |
| Delete | — | `POST .../unassigncollaborators` (pass `DeleteOptions.SSOUser` for SSO users) |

#### Roles (read+write vs read-only)

Set **`spec.role`** to one of the supported backend role constants:

| `spec.role` | Capability |
|-------------|------------|
| `PAAS_WORKSPACE_COLLABORATOR` | Read and write workspace resources |
| `PAAS_WORKSPACE_COLLABORATOR_READ_ONLY` | Read-only workspace access |

Go constants: `v1alpha1.WorkspaceRoleCollaborator` and `v1alpha1.WorkspaceRoleCollaboratorReadOnly`.

#### SSO users (`isSSOUser`)

When assigning or removing an **existing SSO user** (not a local console user), set `spec.isSSOUser: true`. The SDK translates this to the backend query parameter `ssoUsers=true` on assign/unassign requests (this is how the hub API distinguishes SSO users — it is not a field on the paas JSON body).

For list filtering to SSO users only, use typed client `ListOptions.SSOUsers`. For unassign without the full object, use `DeleteOptions.SSOUser`.

#### Assign an existing Rafay user (read+write)

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: WorkspaceCollaborator
metadata:
  name: alice@gpupaas.ai
  project: demo
  workspace: dev
spec:
  username: alice@gpupaas.ai
  role: PAAS_WORKSPACE_COLLABORATOR
```

#### Assign an existing SSO user (read-only)

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: WorkspaceCollaborator
metadata:
  name: bob.sso@gpupaas.ai
  project: demo
  workspace: dev
spec:
  username: bob.sso@gpupaas.ai
  role: PAAS_WORKSPACE_COLLABORATOR_READ_ONLY
  isSSOUser: true
```

#### Invite a new external user

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: WorkspaceCollaborator
metadata:
  name: guest@gpupaas.ai
  project: demo
  workspace: dev
spec:
  email: guest@gpupaas.ai
  firstName: Guest
  lastName: User
  role: PAAS_WORKSPACE_COLLABORATOR
  userType: Console
```

#### Using Go role constants

```yaml
spec:
  username: alice@gpupaas.ai
  role: PAAS_WORKSPACE_COLLABORATOR_READ_ONLY
```

#### `metadata` fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Collaborator identifier — email-style Rafay username (assign) or invite email |
| `project` | yes | Owning project |
| `workspace` | yes | Target workspace within the project |
| `labels` | no | Passed through on invite (`POST .../collaborators`) |
| `annotations` | no | Tooling metadata |

#### `spec` fields

| Field | Required | Description |
|-------|----------|-------------|
| `username` | assign flow | Existing Rafay username (e.g. `alice@gpupaas.ai`); defaults to `metadata.name` when empty |
| `email` | invite flow | Email for a new external collaborator; triggers invite endpoint |
| `firstName` | no | First name for invited users |
| `lastName` | no | Last name for invited users |
| `role` | yes | `PAAS_WORKSPACE_COLLABORATOR` or `PAAS_WORKSPACE_COLLABORATOR_READ_ONLY` |
| `userType` | no | User type for invited users (e.g. `Console`, `API`) |
| `isSSOUser` | no | When `true`, adds `ssoUsers=true` on assign/unassign for existing users |

#### `status` fields (read-only)

| Field | Description |
|-------|-------------|
| `phase` | Backend condition (e.g. `StatusOK`) from paas `status.conditionStatus` |
| `role` | Collaborator role returned from the backend |

#### Typed client

```go
collab := cs.V1alpha1().Workspaces("demo").Collaborators("dev")

// Assign SSO user with read-only access
_, _ = collab.Create(ctx, &v1alpha1.WorkspaceCollaborator{
    Metadata: v1alpha1.ObjectMeta{Name: "bob.sso@gpupaas.ai", Project: "demo", Workspace: "dev"},
    Spec: v1alpha1.WorkspaceCollaboratorSpec{
        Username:  "bob.sso@gpupaas.ai",
        Role:      v1alpha1.WorkspaceRoleCollaboratorReadOnly,
        IsSSOUser: true,
    },
}, gpupaas.CreateOptions{})

// List SSO collaborators only
sso := true
list, _ := collab.List(ctx, gpupaas.ListOptions{SSOUsers: &sso})

// Remove SSO collaborator
_, _ = collab.Delete(ctx, "bob.sso@gpupaas.ai", gpupaas.DeleteOptions{SSOUser: &sso})
```

---

### VirtualMachine

**What it is for:** A **VirtualMachine** provisions and manages a dev virtual machine instance. VMs can be created at **project scope** (shared across workspaces) or **workspace scope** (scoped to a single workspace).

**Scope:**

| Scope | `metadata.workspace` | Typed client |
|-------|---------------------|--------------|
| Project | omit | `cs.V1alpha1().VirtualMachines(project)` |
| Workspace | required | `cs.V1alpha1().Workspaces(project).VirtualMachines(workspace)` |

**Backend:** `dev.envmgmt.io/v1` paths. Apply is **POST** to the collection URL. Lifecycle actions use `POST .../action/{start|stop|reboot|...}` with an optional `{variables, envs}` body. Observed provisioning state is read via `GET .../status`.

#### Project-scoped create

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: VirtualMachine
metadata:
  name: my-vm
  project: demo
spec:
  virtualMachine:
    name: ubuntu-22-profile
    systemCatalog: true
  cpuCount: "2"
  memory: 4Gi
  securityGroup: default-sg
  sshKey: my-ssh-key
  vpc: tenant-vpc
  subnet: private-subnet
  image: ubuntu-22.04
  bootDiskSize: 50
```

#### Workspace-scoped create

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: VirtualMachine
metadata:
  name: my-vm
  project: demo
  workspace: dev
spec:
  virtualMachine:
    name: ubuntu-22-profile
  cpuCount: "2"
  memory: 4Gi
  image: ubuntu-22.04
```

#### Operations

| Operation | Project client | Workspace client |
|-----------|----------------|------------------|
| List | `VirtualMachines("demo").List(ctx, opts)` | `Workspaces("demo").VirtualMachines("dev").List(...)` |
| Get | `.Get(ctx, name, opts)` | same |
| Create | `.Create(ctx, vm, opts)` | same |
| Delete | `.Delete(ctx, name, opts)` | same |
| Status | `.GetStatus(ctx, name, opts)` | same |
| Start | `.Start(ctx, name, ActionOptions{})` | same |
| Stop | `.Stop(ctx, name, ActionOptions{})` | same |
| Reboot | `.Reboot(ctx, name, ActionOptions{})` | same |
| Generic action | `.Action(ctx, name, verb, ActionOptions{Envs, Variables})` | same |

#### `spec` fields

| Field | Required | Description |
|-------|----------|-------------|
| `virtualMachine.name` | yes | VM profile / catalog reference |
| `virtualMachine.systemCatalog` | no | Use system catalog profile |
| `vmId` | no | Inventory device ID of the virtual machine. Usually observed from the backend; may be set by clients to pin a VM to a specific device (wire field: `vm_id`). |
| `cpuCount` | no | CPU count (string) |
| `memory` | no | Memory size (e.g. `4Gi`) |
| `securityGroup` | no | Security group name or ID — see [SecurityGroup](#securitygroup) |
| `sshKey` | no | SSH key name or ID — see [SshKey](#sshkey) |
| `vpc` / `subnet` | no | Network placement |
| `assignPublicIp` | no | Assign a public IP |
| `image` | no | OS image |
| `bootDiskSize` | no | Boot disk size in GB |
| `sharing` | no | Share mode (`None`, `All`, `Custom`) with workspaces/projects |
| `sharedStorage` / `blockStorageType` | no | Storage references — see [Storage](#storage) |

Wire mapping uses snake_case on `dev.envmgmt.io/v1` (e.g. `virtual_machine`, `security_group`); the SDK uses camelCase.

#### `status` fields (read-only)

| Field | Description |
|-------|-------------|
| `status` | Provisioning state (e.g. `success`, `pending`) |
| `reason` | Human-readable reason |
| `action` | Last action |
| `output` | `hostName`, `privateIp`, `publicIp`, etc. |
| `provisionedAt` / `lastConnectedAt` | Timestamps from backend |

#### Typed client

```go
// Project-scoped
projVMs := cs.V1alpha1().VirtualMachines("demo")
list, _ := projVMs.List(ctx, gpupaas.ListOptions{})
vm, _ := projVMs.Create(ctx, &v1alpha1.VirtualMachine{...}, gpupaas.CreateOptions{})
_, _ = projVMs.Start(ctx, "my-vm", gpupaas.ActionOptions{})

// Workspace-scoped
wsVMs := cs.V1alpha1().Workspaces("demo").VirtualMachines("dev")
vm, _ = wsVMs.Create(ctx, &v1alpha1.VirtualMachine{
    Metadata: v1alpha1.ObjectMeta{Name: "my-vm", Project: "demo", Workspace: "dev"},
    Spec:     v1alpha1.VirtualMachineSpec{VirtualMachine: v1alpha1.ResourceRef{Name: "ubuntu-22-profile"}},
}, gpupaas.CreateOptions{})
```

#### Examples

```bash
go run ./examples/list-vms -memory -project demo
go run ./examples/create-vm -memory -project demo -workspace dev -name example-vm
go run ./examples/get-vm -memory -project demo -name example-vm
go run ./examples/delete-vm -memory -project demo -name example-vm
GPUPAAS_API_KEY=... go run ./examples/vm-start  -project demo -workspace dev -name example-vm
GPUPAAS_API_KEY=... go run ./examples/vm-stop   -project demo -workspace dev -name example-vm
GPUPAAS_API_KEY=... go run ./examples/vm-reboot -project demo -workspace dev -name example-vm
```

---

### Storage

**What it is for:** Block or file storage volumes used by virtual machines (`spec.sharedStorage`, `spec.blockStorageType` on [VirtualMachine](#virtualmachine)).

**Scope:**

| Scope | `metadata.workspace` | Typed client |
|-------|---------------------|--------------|
| Project | omit | `cs.V1alpha1().Storages(project)` |
| Workspace | required | `cs.V1alpha1().Workspaces(project).Storages(workspace)` |

**Backend:** `POST /apis/dev.envmgmt.io/v1/projects/{project}/storages` (or workspace path). Apply is **POST** to the collection URL.

#### Project-scoped create

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: Storage
metadata:
  name: my-storage
  project: demo
spec:
  storage:
    name: my-storage
  type: standard
  size: 10Gi
```

#### Workspace-scoped create

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: Storage
metadata:
  name: my-storage
  project: demo
  workspace: dev
spec:
  storage:
    name: my-storage
  type: standard
  size: 10Gi
```

#### `spec` fields

| Field | Required | Description |
|-------|----------|-------------|
| `storage.name` | yes | Storage resource name |
| `storage.systemCatalog` | no | Use system catalog entry |
| `type` | no | Storage type (e.g. `standard`, `block`) |
| `size` | no | Capacity (e.g. `10Gi`) |
| `datacenter` | no | Target datacenter |
| `accessPolicy` | no | Access policy |
| `contractTerm` | no | Contract term |
| `enableEncryptionAtRest` / `enableEncryptionInTransit` | no | Encryption flags |
| `storageType` | no | Classification (`block`, `file`, `object`) |
| `sharing` | no | Share mode with workspaces/projects |

#### Typed client

```go
storages := cs.V1alpha1().Storages("demo")
list, _ := storages.List(ctx, gpupaas.ListOptions{})
vol, _ := storages.Create(ctx, &v1alpha1.Storage{...}, gpupaas.CreateOptions{})

wsStorages := cs.V1alpha1().Workspaces("demo").Storages("dev")
vol, _ = wsStorages.Create(ctx, &v1alpha1.Storage{...}, gpupaas.CreateOptions{})
```

#### Examples

```bash
go run ./examples/list-storages -memory -project demo
go run ./examples/create-storage -memory -project demo -name example-storage
go run ./examples/get-storage -memory -project demo -name example-storage
go run ./examples/delete-storage -memory -project demo -name example-storage
```

---

### SecurityGroup

**What it is for:** Firewall rules for VM network access. Referenced by `spec.securityGroup` on [VirtualMachine](#virtualmachine).

**Scope:** Same dual-scope pattern as Storage (`Storages` / `SecurityGroups` on project or workspace client).

**Backend:** `POST .../securitygroups` on `dev.envmgmt.io/v1`.

#### Project-scoped create

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: SecurityGroup
metadata:
  name: default-sg
  project: demo
spec:
  securityGroup:
    name: default-sg
  type: default
  ipRules:
    - sourceCidr: 0.0.0.0/0
      application: ssh
      action: allow
```

#### `spec` fields

| Field | Required | Description |
|-------|----------|-------------|
| `securityGroup.name` | yes | Security group name |
| `type` | no | Security group type |
| `ipRules` | no | IP-based rules (`sourceCidr`, `application`, `action`) |
| `portForwardRules` | no | Port forwarding rules |
| `rules` | no | Combined access rules |
| `sharing` | no | Share mode with workspaces/projects |

Wire fields: `security_group`, `ip_rules`, `port_forward_rules`, `source_cidr`.

#### Typed client

```go
sgs := cs.V1alpha1().SecurityGroups("demo")
sg, _ := sgs.Create(ctx, &v1alpha1.SecurityGroup{...}, gpupaas.CreateOptions{})

wsSGs := cs.V1alpha1().Workspaces("demo").SecurityGroups("dev")
sg, _ = wsSGs.Get(ctx, "default-sg", gpupaas.GetOptions{})
```

#### Examples

```bash
go run ./examples/list-security-groups -memory -project demo
go run ./examples/create-security-group -memory -project demo -name default-sg
go run ./examples/get-security-group -memory -project demo -name default-sg
go run ./examples/delete-security-group -memory -project demo -name default-sg
```

---

### SshKey

**What it is for:** SSH public keys for VM login. Referenced by `spec.sshKey` on [VirtualMachine](#virtualmachine).

**Scope:** Same dual-scope pattern (`SshKeys` on project or workspace client).

**Backend:** `POST .../sshkeys` on `dev.envmgmt.io/v1`.

#### Project-scoped create

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: SshKey
metadata:
  name: my-ssh-key
  project: demo
spec:
  sshKey:
    name: my-ssh-key
  publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...
```

#### `spec` fields

| Field | Required | Description |
|-------|----------|-------------|
| `sshKey.name` | yes | SSH key name |
| `publicKey` | no | Public key material |
| `type` | no | Key type |
| `sharing` | no | Share mode with workspaces/projects |

Wire fields: `ssh_key`, `public_key`.

#### Typed client

```go
keys := cs.V1alpha1().SshKeys("demo")
key, _ := keys.Create(ctx, &v1alpha1.SshKey{...}, gpupaas.CreateOptions{})

wsKeys := cs.V1alpha1().Workspaces("demo").SshKeys("dev")
key, _ = wsKeys.List(ctx, gpupaas.ListOptions{})
```

#### Examples

```bash
go run ./examples/list-ssh-keys -memory -project demo
go run ./examples/create-ssh-key -memory -project demo -name example-ssh-key
go run ./examples/get-ssh-key -memory -project demo -name example-ssh-key
go run ./examples/delete-ssh-key -memory -project demo -name example-ssh-key
```

---

### BaremetalMachine

**What it is for:** A **BaremetalMachine** represents a physical host managed by the gpupaas baremetal control plane (metal3 / ironic). Use it to enroll, image, power-cycle, and reinstall a baremetal node, and to open a Serial-Over-LAN (SOL) console session for remote diagnostics.

**Scope:** Project-scoped only — requires `metadata.name` and `metadata.project`. **There is no workspace scope** for BaremetalMachine; passing `metadata.workspace` is ignored. The wire `apiVersion` is `infra.k8smgmt.io/v3`; the SDK normalizes reads to `gpupaas.ai/v1alpha1`.

**Backend mapping** (`infra.k8smgmt.io/v3`):

| SDK operation | Backend call | Notes |
|---------------|--------------|-------|
| Create / Apply | `POST /apis/infra.k8smgmt.io/v3/projects/{project}/baremetalmachines` | Upsert; returns the applied object |
| List | `GET .../baremetalmachines` | Supports `limit`, `offset`, `selector` via `ListOptions` |
| Get | `GET .../baremetalmachines/{name}` | |
| Update | `POST .../baremetalmachines` | Same path as Create (idempotent apply) |
| Delete | `DELETE .../baremetalmachines/{name}` | Supports `auto-destroy`, `force` via `DeleteOptions` |
| PowerOn | `GET .../baremetalmachines/{name}/powerOn` | Returns the updated BaremetalMachine |
| PowerOff | `GET .../baremetalmachines/{name}/powerOff` | Returns the updated BaremetalMachine |
| Reboot | `GET .../baremetalmachines/{name}/reboot` | Returns the updated BaremetalMachine |
| Provision | `GET .../baremetalmachines/{name}/provision` | Triggers deployment; returns the updated BaremetalMachine |
| ReinstallOS | `POST .../baremetalmachines/{name}/reinstallOS` | Body is a `BaremetalImage` (URL, format, checksum, checksumType); returns the updated BaremetalMachine |
| CreateConsoleSession | `POST .../baremetalmachines/{name}/consoleSessions` | Body has `compute_id` (snake_case); returns `BaremetalConsoleSession` (`session_id`, `agent_session_id`, `console_url`) |
| GetStatusInfo | `GET .../baremetalmachines/{name}/status` | Returns `BaremetalMachineInfo` (free-form `data.fields`); distinct from inline `status.conditions` |

#### Create / Apply

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: BaremetalMachine
metadata:
  name: bm-01
  project: demo
spec:
  baremetalProvisionerName: default-provisioner
  hostname: bm-01
  datacenter: dc1
  deviceId: device-001
  macAddress: aa:bb:cc:dd:ee:ff
  architecture: x86_64
  bootMode: UEFI
  automatedCleaningMode: metadata
  online: true
  sshKey: my-ssh-key
  image:
    url: http://images/example.qcow2
    format: qcow2
    checksumType: auto
    checksum: auto
  rootDeviceHints:
    deviceName: /dev/sda
    minSizeGigabytes: 100
  raid:
    hardwareRAIDVolumes:
      - name: root
        level: "1"
        numberOfPhysicalDisks: 2
        sizeGibibytes: 200
  userData: |
    #cloud-config
    package_update: true
```

#### `metadata` fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Unique baremetal machine name within the project |
| `project` | yes | Owning project name |
| `displayName` | no | Human-readable name (writable) |
| `description` | no | Human-readable description (writable) |
| `labels` / `annotations` | no | Tags and tooling metadata |
| `createdBy` / `modifiedBy` | no (read-only) | `UserMeta` populated by the backend; ignored on writes |

#### `spec` fields

| Field | Type | Description |
|-------|------|-------------|
| `baremetalProvisionerName` | string | Provisioner that manages this host |
| `hostname` | string | Hostname assigned to the machine |
| `datacenter` | string | Inventory datacenter the host lives in |
| `deviceId` | string | Inventory device id (wire field: `deviceId`) |
| `macAddress` | string | MAC address of the provisioning NIC |
| `architecture` | string | CPU architecture (`x86_64`, `aarch64`); usually populated by inspection |
| `bootMode` | string | `UEFI` (default), `Legacy`, or `UEFISecureBoot` |
| `automatedCleaningMode` | string | Set to `disabled` to skip cleaning |
| `online` | bool | Desired power state when the host is in a stable state |
| `sshKey` | string | SSH key name injected for first-boot login |
| `userData` / `systemUserData` | string | cloud-init payloads |
| `image` | `BaremetalImage` | Deployment image (`url`, `format`, `checksum`, `checksumType`) |
| `rootDeviceHints` | `BaremetalRootDeviceHints` | Hints to narrow the OS deployment disk |
| `raid` | `BaremetalRaid` | `hardwareRAIDVolumes` / `softwareRAIDVolumes` configuration |

#### `status` fields (read-only)

| Field | Description |
|-------|-------------|
| `conditions[]` | `type`, `status` (`NotSet`, `Pending`, `Success`, `Failed`), `reason`, `lastUpdated` (RFC3339) — observed lifecycle conditions |

The richer per-host telemetry (hardware inventory, network observations) is exposed separately via `GetStatusInfo` as a `BaremetalMachineInfo{ Data: { Fields: map[string]any } }` envelope.

#### Imperative sub-actions

```go
bms := cs.V1alpha1().BaremetalMachines("demo")

// Power management (GET-based; return the updated BaremetalMachine)
_, _ = bms.PowerOn(ctx, "bm-01", gpupaas.ActionOptions{})
_, _ = bms.PowerOff(ctx, "bm-01", gpupaas.ActionOptions{})
_, _ = bms.Reboot(ctx, "bm-01", gpupaas.ActionOptions{})

// Provisioning lifecycle
_, _ = bms.Provision(ctx, "bm-01", gpupaas.ActionOptions{})
_, _ = bms.ReinstallOS(ctx, "bm-01", &v1alpha1.BaremetalImage{
    URL:          "http://images/example.qcow2",
    Format:       "qcow2",
    ChecksumType: "auto",
    Checksum:     "auto",
}, gpupaas.ActionOptions{})

// Serial-Over-LAN console (POST; returns a short-lived WebSocket URL)
session, _ := bms.CreateConsoleSession(ctx, "bm-01", &v1alpha1.BaremetalConsoleSessionRequest{
    ComputeID: "compute-123",
}, gpupaas.ActionOptions{})
// session.ConsoleURL, session.SessionID, session.AgentSessionID

// Free-form runtime info
info, _ := bms.GetStatusInfo(ctx, "bm-01", gpupaas.GetOptions{})
// info.Data.Fields ...
```

#### Typed client

```go
bms := cs.V1alpha1().BaremetalMachines("demo")
list, _ := bms.List(ctx, gpupaas.ListOptions{})
bm, _ := bms.Create(ctx, &v1alpha1.BaremetalMachine{
    TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindBaremetalMachine},
    Metadata: v1alpha1.ObjectMeta{Name: "bm-01", Project: "demo"},
    Spec:     v1alpha1.BaremetalMachineSpec{Hostname: "bm-01"},
}, gpupaas.CreateOptions{})
_, _ = bms.Get(ctx, "bm-01", gpupaas.GetOptions{})
_ = bms.Delete(ctx, "bm-01", gpupaas.DeleteOptions{})
```

#### Examples

```bash
go run ./examples/create-baremetal -memory -project demo -name example-bm
go run ./examples/get-baremetal -memory -project demo -name example-bm
go run ./examples/list-baremetals -memory -project demo
go run ./examples/delete-baremetal -memory -project demo -name example-bm
GPUPAAS_API_KEY=... go run ./examples/baremetal-reboot           -project demo -name example-bm
GPUPAAS_API_KEY=... go run ./examples/baremetal-reinstall-os     -project demo -name example-bm -image-url http://images/example.qcow2
GPUPAAS_API_KEY=... go run ./examples/baremetal-console-session  -project demo -name example-bm -compute-id compute-123
GPUPAAS_API_KEY=... go run ./examples/baremetal-status           -project demo -name example-bm
```

---

### MKSCluster

**What it is for:** An **MKSCluster** (Managed Kubernetes Service) provisions and manages a Kubernetes cluster on gpupaas infrastructure. Creating a cluster persists it in the MKS database, syncs node entries, and triggers a WorkspaceComputeInstance to provision the cluster via the EaaS `mks-provision` environment template. The SDK also exposes per-cluster sub-resources: nodes, worker node groups, and audit events.

**Scope:** Project-scoped only — requires `metadata.name` and `metadata.project`. **There is no workspace scope.** The wire `apiVersion` is `paas.envmgmt.io/v1`; the SDK normalizes reads to `gpupaas.ai/v1alpha1`. The `MKSProfile` resource from the MKS API is intentionally **not** part of this SDK.

**Backend mapping** (`paas.envmgmt.io/v1`):

| SDK operation | Backend call | Notes |
|---------------|--------------|-------|
| Create / Apply | `POST /apis/paas.envmgmt.io/v1/projects/{project}/mksclusters` | Upsert; persisted even if the provisioning trigger fails (with an error status) |
| List | `GET .../mksclusters` | Supports `limit`, `offset` via `ListOptions` |
| Get | `GET .../mksclusters/{name}` | |
| Delete | `DELETE .../mksclusters/{name}` | Soft-delete; triggers a destroy WCI |
| Upgrade | `POST .../mksclusters/{name}/upgrade` | Body `{k8sVersion, platformVersion}`; returns the updated cluster |
| ScaleNodeGroup | `POST .../mksclusters/{name}/scaleNodeGroup` | Body `{nodeGroupName, desiredCount, minCount, maxCount}` |
| AddNodeGroup | `POST .../mksclusters/{name}/addNodeGroup` | Body `{nodeGroup}` (an `MKSNodeGroup`) |
| RemoveNodeGroup | `POST .../mksclusters/{name}/removeNodeGroup` | Body `{nodeGroupName}` |

Sub-resource clients are reached through the cluster client:
`cs.V1alpha1().MKSClusters(project).Nodes(cluster)`, `.WorkerNodeGroups(cluster)`, `.AuditEvents(cluster)`.

| Sub-resource | SDK operations | Backend call |
|--------------|----------------|--------------|
| MKSNode | List, Get, Delete, Drain, Cordon, Uncordon | `GET .../mksnodes`, `GET/DELETE .../{name}`, `POST .../{name}/{drain\|cordon\|uncordon}` |
| MKSWorkerNodeGroup | List, Get, Create (apply), Delete | `GET/POST .../workernodegroups`, `GET/DELETE .../{name}` |
| MKSAuditEvent | List, Get (by id) | `GET .../auditevents`, `GET .../auditevents/{id}` (read-only) |

> The generic remote backend (`client.Apply/Get/List/Delete`) supports `MKSCluster` only. Sub-resources require a cluster name and are accessed through the typed clientset.

#### Create / Apply

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: MKSCluster
metadata:
  name: my-cluster
  project: demo
spec:
  kubernetesVersion: "1.31"
  cni: calico
  os: ubuntu22.04
  haEnabled: false
  dedicatedControlPlane: false
  blueprint:
    name: minimal
    version: v1
  networking:
    podCidr: 192.168.0.0/16
    serviceCidr: 10.96.0.0/12
    ipFamily: IPv4
  nodes:
    - hostname: master-1
      privateIp: 10.0.0.10
      sshUserName: ubuntu
      sshKey: "-----BEGIN OPENSSH PRIVATE KEY-----\n..."
      roles: [master, worker]
      arch: amd64
      operatingSystem: ubuntu22.04
```

#### `metadata` fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Unique cluster name within the project |
| `project` | yes | Owning project name |
| `labels` / `annotations` | no | Tags and tooling metadata |

#### `spec` fields

| Field | Type | Description |
|-------|------|-------------|
| `kubernetesVersion` | string | Kubernetes version in `major.minor` form (e.g. `1.31`) |
| `platformVersion` | string | Platform version |
| `cni` / `cniVersion` | string | CNI provider and version (e.g. `calico`) |
| `os` | string | Node operating system (e.g. `ubuntu22.04`) |
| `haEnabled` | bool | Highly-available control plane |
| `dedicatedControlPlane` | bool | Dedicate control-plane nodes (no workloads) |
| `location` | string | Cluster location/datacenter |
| `blueprint` | `MKSBlueprint` | `name` + `version` of the cluster blueprint |
| `networking` | `MKSNetworking` | `vpc`, `subnet`, `podCidr`, `serviceCidr`, `ipFamily`, `securityGroups`, … |
| `proxy` | `MKSProxy` | `httpProxy`, `httpsProxy`, `noProxy`, `proxyRootCa`, `tlsTerminate` |
| `storage` | `MKSStorage` | `block` / `sharedFs` / `object` / `highSpeed` backends + `defaultStorageClass` |
| `tags` | map | Free-form key/value tags |
| `controlPlaneNodeGroup` | `MKSNodeGroup` | Control-plane node group spec |
| `workerNodeGroups` | `[]MKSNodeGroup` | Worker node group specs |
| `nodes` | `[]MKSNodeSpec` | Node specifications for device-based clusters |

`MKSNodeGroup` carries `id`, `sku`, `scalingMode`, `nodeCount`, `minNodes`, `maxNodes`, `desiredNodes`, `publicIp`, `sshKey`, `userData`, `nodeLabels`, `nodeAnnotations`, and `kubeletConfig`. `MKSNodeSpec` carries `hostname`, `roles`, `sshUserName`, `sshKey`, `sshPort`, `privateIp`, `arch`, `operatingSystem`, `interface`, `nodeLabels`, `nodeAnnotations`, `nodeTaints`, `userData`, `kubeletConfig`, `nodePool`, `sku`, and `publicIp`.

#### `status` fields (read-only)

| Field | Description |
|-------|-------------|
| `condition` | Cluster condition (e.g. `MKS_CLUSTER_STATUS_PROVISIONING`, `MKS_CLUSTER_STATUS_RUNNING`, `MKS_CLUSTER_STATUS_ERROR`) |
| `conditionReason` | Human-readable reason |
| `output` | `apiServerEndpoint`, `clusterIdEdgesrv` |
| `action` | Last action |

#### Imperative sub-actions

```go
clusters := cs.V1alpha1().MKSClusters("demo")

// Upgrade Kubernetes version
_, _ = clusters.Upgrade(ctx, "my-cluster", &v1alpha1.MKSUpgradeRequest{
    K8sVersion: "1.32",
}, gpupaas.ActionOptions{})

// Scale a worker node group
desired := int32(5)
_, _ = clusters.ScaleNodeGroup(ctx, "my-cluster", &v1alpha1.MKSScaleNodeGroupRequest{
    NodeGroupName: "wng-1",
    DesiredCount:  &desired,
}, gpupaas.ActionOptions{})

// Add / remove worker node groups
_, _ = clusters.AddNodeGroup(ctx, "my-cluster", &v1alpha1.MKSNodeGroup{ID: "wng-2", SKU: "small"}, gpupaas.ActionOptions{})
_, _ = clusters.RemoveNodeGroup(ctx, "my-cluster", "wng-1", gpupaas.ActionOptions{})
```

#### Node, worker node group, and audit event sub-clients

```go
clusters := cs.V1alpha1().MKSClusters("demo")

// Nodes
nodes := clusters.Nodes("my-cluster")
nodeList, _ := nodes.List(ctx, gpupaas.ListOptions{})
force := true
_, _ = nodes.Drain(ctx, "master-1", &v1alpha1.MKSDrainRequest{Force: &force}, gpupaas.ActionOptions{})
_, _ = nodes.Cordon(ctx, "master-1", gpupaas.ActionOptions{})
_, _ = nodes.Uncordon(ctx, "master-1", gpupaas.ActionOptions{})

// Worker node groups (apply = create/update)
wngs := clusters.WorkerNodeGroups("my-cluster")
_, _ = wngs.Create(ctx, &v1alpha1.MKSWorkerNodeGroup{
    Metadata: v1alpha1.ObjectMeta{Name: "wng-1", Project: "demo"},
    Spec:     v1alpha1.MKSWorkerNodeGroupSpec{NodeGroup: &v1alpha1.MKSNodeGroup{ID: "wng-1", SKU: "medium"}},
}, gpupaas.CreateOptions{})

// Audit events (read-only)
events := clusters.AuditEvents("my-cluster")
evtList, _ := events.List(ctx, gpupaas.ListOptions{})
```

#### Typed client

```go
clusters := cs.V1alpha1().MKSClusters("demo")
list, _ := clusters.List(ctx, gpupaas.ListOptions{})
cluster, _ := clusters.Create(ctx, &v1alpha1.MKSCluster{
    TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindMKSCluster},
    Metadata: v1alpha1.ObjectMeta{Name: "my-cluster", Project: "demo"},
    Spec:     v1alpha1.MKSClusterSpec{KubernetesVersion: "1.31", CNI: "calico"},
}, gpupaas.CreateOptions{})
_, _ = clusters.Get(ctx, "my-cluster", gpupaas.GetOptions{})
_ = clusters.Delete(ctx, "my-cluster", gpupaas.DeleteOptions{})
```

#### Examples

```bash
go run ./examples/create-mks-cluster -memory -project demo -name my-cluster
go run ./examples/get-mks-cluster -memory -project demo -name my-cluster
go run ./examples/list-mks-clusters -memory -project demo
go run ./examples/delete-mks-cluster -memory -project demo -name my-cluster
GPUPAAS_API_KEY=... go run ./examples/mks-cluster-upgrade          -project demo -name my-cluster -k8s-version 1.32
GPUPAAS_API_KEY=... go run ./examples/mks-cluster-scale-nodegroup  -project demo -name my-cluster -node-group wng-1 -desired-size 5
GPUPAAS_API_KEY=... go run ./examples/list-mks-nodes              -project demo -name my-cluster
GPUPAAS_API_KEY=... go run ./examples/mks-node-drain             -project demo -name my-cluster -node master-1
GPUPAAS_API_KEY=... go run ./examples/list-mks-worker-node-groups -project demo -name my-cluster
GPUPAAS_API_KEY=... go run ./examples/list-mks-audit-events       -project demo -name my-cluster
```

---

### Shared metadata conventions

| Field | Used on | Notes |
|-------|---------|-------|
| `metadata.name` | All resources | Primary identifier; immutable after create |
| `metadata.project` | Workspace, VirtualMachine, BaremetalMachine, MKSCluster, Storage, SecurityGroup, SshKey, WorkspaceCollaborator | Parent project; required for scoped resources |
| `metadata.workspace` | VirtualMachine, Storage, SecurityGroup, SshKey (workspace scope), WorkspaceCollaborator | Parent workspace within a project |
| `metadata.displayName` | All | Optional human-friendly name. Sent on writes. |
| `metadata.description` | All | Optional human-readable description. Sent on writes. |
| `metadata.labels` | All | Selectors and automation hooks |
| `metadata.annotations` | All | Tooling-specific key/value data |
| `metadata.createdBy` | All (read-only) | `UserMeta` populated by the backend on reads; **ignored on writes** |
| `metadata.modifiedBy` | All (read-only) | `UserMeta` populated by the backend on reads; **ignored on writes** |

`UserMeta` carries `username`, `isSSOUser`, and an optional `options` block (`description`, `required`, `override.type`, `override.restrictedValues`). It is read-only — `ToDev*` converters strip it before sending.

When applying YAML with `apply.ApplyFile`, pass the default project (and workspace when needed) as arguments; empty fields on the object are filled from those defaults.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
