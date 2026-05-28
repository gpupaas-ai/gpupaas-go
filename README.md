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
go run ./examples/apply_yaml ./manifest.yaml demo
go run ./examples/delete_yaml ./manifest.yaml demo
```

## Related projects

| Project | Role |
|---------|------|
| [gpupaasctl](https://github.com/gpupaas-ai/gpupaasctl) | kubectl-style CLI (consumer) |
| [terraform-provider-gpupaas](https://github.com/gpupaas-ai/terraform-provider-gpupaas) | Terraform provider (consumer) |
| [paasctl](https://github.com/RafaySystems/paasctl) | Reference client architecture |

## Stability

API group `gpupaas.ai/v1alpha1` may change until a beta or stable release.

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
    ├── VirtualMachine   project-scoped (metadata.project only)
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

**Backend:** `dev.envmgmt.io/v1` paths. Apply is **POST** to the collection URL. Start/stop use `POST .../action/start` and `POST .../action/stop`.

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

#### `spec` fields

| Field | Required | Description |
|-------|----------|-------------|
| `virtualMachine.name` | yes | VM profile / catalog reference |
| `virtualMachine.systemCatalog` | no | Use system catalog profile |
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
GPUPAAS_API_KEY=... go run ./examples/vm-start -project demo -workspace dev -name example-vm
GPUPAAS_API_KEY=... go run ./examples/vm-stop -project demo -workspace dev -name example-vm
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

### Shared metadata conventions

| Field | Used on | Notes |
|-------|---------|-------|
| `metadata.name` | All resources | Primary identifier; immutable after create |
| `metadata.project` | Workspace, VirtualMachine, Storage, SecurityGroup, SshKey, WorkspaceCollaborator | Parent project; required for scoped resources |
| `metadata.workspace` | VirtualMachine, Storage, SecurityGroup, SshKey (workspace scope), WorkspaceCollaborator | Parent workspace within a project |
| `metadata.labels` | All | Selectors and automation hooks |
| `metadata.annotations` | All | Tooling-specific key/value data |

When applying YAML with `apply.ApplyFile`, pass the default project (and workspace when needed) as arguments; empty fields on the object are filled from those defaults.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
