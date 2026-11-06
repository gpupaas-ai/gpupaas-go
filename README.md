# gpupaas-go

Go SDK for the [gpupaas.ai](https://gpupaas.ai) API. Shared client library for **terraform-provider-gpupaas**, and custom automation.

The SDK covers three resource families:

| Family | Kinds | Scope | Backend API |
|--------|-------|-------|-------------|
| **Virtual machines** | `VMInstance` (wire kind `VirtualMachine`), `VMProfile` (read-only SKU) | project or workspace | `dev.envmgmt.io/v1` |
| **Bare metal** | `BareMetal`, `BmProfile` (read-only SKU) | project or workspace | `dev.envmgmt.io/v1` |
| **MKS clusters** (managed Kubernetes) | `MKSCluster`, `MKSClusterProfile` and `MKSNode`, kubeconfig, day-2 routes (pools, upgrade, metadata, redeploy), status, audit events, node and pool operations, matchmaker | project or workspace (profiles and nodes: project) | `paas.k8smgmt.io/v3` |

Two read-only helpers support them: `SshKey` (look a key up by name) and `ComputeInstance` (the instance that sits behind every VM, bare-metal machine and cluster node and explains a failure; see [ComputeInstance](#computeinstance-read-only)).

The API uses a Kubernetes-inspired resource shape (`apiVersion`, `kind`, `metadata`, `spec`, `status`) with gpupaas-native scoping: `metadata.project` plus an optional `metadata.workspace`. Projects and workspaces themselves are not managed by this SDK; they must already exist, and the workspace is named by the caller (`V1alpha1().Workspaces(project)` is only the accessor for the resources inside it).

## Install

```bash
go get github.com/gpupaas-ai/gpupaas-go
```

## Controller compatibility

The SDK tags `vX.Y.Z` of one line (`vX.Y`) speak to one controller API line, which the constant `gpupaas.ControllerAPI`
reports (`"vX.Y"`). Use the SDK line that matches your controller. A patch tag fixes a defect inside its line; a
controller generation that changes the API gets a new line and a `release/vX.Y` branch.

| Line   | Controller                                                              | Status         |
|--------|-------------------------------------------------------------------------|----------------|
| v0.1.x | Pre-release: verified against development controllers only, no release | In development |

## Quick start

Typed clientset (every family; a live API key is required):

```go
cs, err := clientset.NewForConfig(gpupaas.ConfigFromEnv())
if err != nil {
    log.Fatal(err)
}
vms := cs.V1alpha1().Workspaces("demo").VMInstances("dev") // or cs.V1alpha1().VMInstances("demo") for project scope

_, err = vms.Create(ctx, &v1alpha1.VMInstance{
    TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindVMInstance},
    Metadata: v1alpha1.ObjectMeta{Name: "my-vm"},
    Spec:     v1alpha1.VMInstanceSpec{Type: "vm-sku", CPUCount: "2", Memory: "4", Image: "ubuntu-24", SSHKey: "my-key"},
}, gpupaas.CreateOptions{})

ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
defer cancel()
running, err := typed.WaitForVMInstance(ctx, vms, "my-vm", typed.VMInstanceWaitOptions{Done: typed.VMInstanceRunning})
```

Generic client (apply, get, list and delete by kind; handles `MKSCluster`), here on the in-memory backend:

```go
c := client.New(client.Options{UseMemory: true})
ctx := context.Background()

cluster := &v1alpha1.MKSCluster{
    TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindMKSCluster},
    Metadata: v1alpha1.ObjectMeta{Name: "example", Project: "demo"},
    Spec:     v1alpha1.MKSClusterSpec{Profile: &v1alpha1.MKSProfileRef{Name: "mks-oneclick-1"}, KubernetesVersion: "1.36"},
}
applied, err := c.Apply(ctx, cluster, "demo", "")
items, err := c.List(ctx, gvk, "demo", "")
```

Use `UseMemory: false` with `Config: gpupaas.ConfigFromEnv()` for the live API.

## Backend translation

The public SDK speaks **Kubernetes-style** `gpupaas.ai/v1alpha1` objects. The remote backend uses different REST shapes; the clientset translates automatically:

| Resource | SDK (k8s-style) | Backend API |
|----------|-----------------|-------------|
| VMInstance (project) | `VMInstance`, `VMInstanceList` | `GET/POST /apis/dev.envmgmt.io/v1/projects/{project}/virtualmachines`, `GET/DELETE .../{name}`, `POST .../{name}/action/{verb}` |
| VMInstance (workspace) | `VMInstance` | Same operations under `.../projects/{project}/workspaces/{workspace}/virtualmachines` |
| VMProfile (read-only) | `VMProfile`, `VMProfileList` | `GET /apis/dev.envmgmt.io/v1/projects/{project}/vmprofiles[/{name}]` |
| BareMetal | `BareMetal`, `BareMetalList` | `GET/POST /apis/dev.envmgmt.io/v1/projects/{project}/baremetals`, `GET/DELETE .../{name}`, `POST .../{name}/action/{verb}`; same under `.../workspaces/{workspace}/baremetals` |
| BmProfile (read-only) | `BmProfile`, `BmProfileList` | `GET /apis/dev.envmgmt.io/v1/projects/{project}/bmprofiles[/{name}]` |
| MKSCluster | `MKSCluster`, `MKSClusterList` | `GET/POST /apis/paas.k8smgmt.io/v3/projects/{project}/clusters`, `GET/DELETE .../{name}`; same under `.../workspaces/{workspace}/clusters` |
| MKSClusterProfile (read-only) | `MKSClusterProfile`, `MKSClusterProfileList` | `GET /apis/paas.k8smgmt.io/v3/projects/{project}/clusterprofiles[/{name}]` |
| MKSNode (read-only) | `MKSNode`, `MKSNodeList` | `GET /apis/paas.k8smgmt.io/v3/projects/{project}/clusternodes?cluster_name={cluster}` |
| SshKey (read-only) | `SshKey` | `GET /apis/dev.envmgmt.io/v1/projects/{project}/[workspaces/{workspace}/]sshkeys/{name}` |
| ComputeInstance (read-only) | `ComputeInstance`, `ComputeInstanceList` | `GET /apis/paas.envmgmt.io/v1/projects/{project}/[workspaces/{workspace}/]computeinstances[/{name}]?include_first_class=true`, plus the EaaS run events for `FailureLog` |

Conversion lives in the `convert/` package (`ToVMInstance`, `FromBareMetal`, `FromMKSCluster`, etc.). Verbose logging shows wire HTTP payloads; returned objects are always normalized to `gpupaas.ai/v1alpha1`.

See the [OpenAPI explorer](https://console.gpupaas.ai/openapi-explorer) for the backend API reference.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GPUPAAS_ENDPOINT` | `https://api.gpupaas.ai` | API base URL |
| `GPUPAAS_API_KEY` | — | Rafay API key (`X-API-KEY`, same as paasctl `api_key`) |
| `GPUPAAS_API_SECRET` | — | API secret for HMAC signing (defaults to API key if unset) |
| `GPUPAAS_ORGANIZATION_ID` | — | Organization id or exact name, sent as `X-Organization-ID`. Needed by a partner-level API key; leave unset for an organization-level key (the platform ignores it there) |
| `GPUPAAS_VERBOSE` | — | Log HTTP traffic when set (`1`, `true`) |
| `GPUPAAS_TOKEN` | — | Deprecated alias for `GPUPAAS_API_KEY` |

Authentication matches [paasctl](https://github.com/RafaySystems/paasctl): `X-API-KEY`, `X-RAFAY-API-KEYID`, and HMAC-SHA256 request signing (`content-md5`, `date`, `host`, `nonce`). There is no `Authorization: Bearer` header.

```go
cfg := gpupaas.ConfigFromEnv()
c := client.New(client.Options{Config: cfg, Verbose: true})
```

## Apply YAML

The apply engine handles `MKSCluster` documents (the kind the generic client manages):

```go
c := client.New(client.Options{UseMemory: true})
err := apply.ApplyFile(ctx, c, "cluster.yaml", "demo", "")
```

Multi-document YAML (`---`) is supported. `apply.DeleteFile` deletes the objects of a manifest. VMs and bare-metal machines are managed through the typed clientset.

## Error handling

```go
if gpupaas.IsNotFound(err) {
    // handle missing resource
}
```

`IsConflict`, `IsUnauthorized`, `IsForbidden` and `IsServerError` classify the other common failures; `*gpupaas.APIError` carries the status code, message and raw body.

## Examples

Virtual machines, bare metal and the MKS actions below need a live API (`GPUPAAS_API_KEY`); the MKS create, get, list and delete examples also run on the in-memory backend with `-memory`.

```bash
export GPUPAAS_API_KEY=... GPUPAAS_API_SECRET=...

# virtual machines (first-class)
go run ./examples/create-vm -project demo -workspace dev -name example-vm -sku vm-sku -cpu 2 -memory 4 -image ubuntu-24 -ssh-key my-key -wait
go run ./examples/get-vm -project demo -workspace dev -name example-vm
go run ./examples/list-vms -project demo -workspace dev
go run ./examples/vm-start -project demo -workspace dev -name example-vm
go run ./examples/vm-stop -project demo -workspace dev -name example-vm
go run ./examples/vm-reboot -project demo -workspace dev -name example-vm
go run ./examples/delete-vm -project demo -workspace dev -name example-vm -wait

# bare metal (first-class)
go run ./examples/create-baremetal -project demo -workspace dev -name example-bm -sku bm-sku -image ubuntu-24 -ssh-key my-key -wait
go run ./examples/get-baremetal -project demo -workspace dev -name example-bm
go run ./examples/list-baremetals -project demo -workspace dev
go run ./examples/baremetal-reboot -project demo -workspace dev -name example-bm
go run ./examples/baremetal-reinstall-os -project demo -workspace dev -name example-bm -image ubuntu-26
go run ./examples/delete-baremetal -project demo -workspace dev -name example-bm -wait

# MKS clusters
go run ./examples/create-mks-cluster -memory -project demo -name my-cluster
go run ./examples/get-mks-cluster -memory -project demo -name my-cluster
go run ./examples/list-mks-clusters -memory -project demo
go run ./examples/delete-mks-cluster -memory -project demo -name my-cluster
go run ./examples/mks-cluster-upgrade -project demo -name my-cluster -k8s-version 1.36
go run ./examples/list-mks-nodes -project demo -name my-cluster

# generic client and YAML apply (MKSCluster, in memory)
go run ./examples/typed_client
go run ./examples/apply_yaml ./cluster.yaml demo
go run ./examples/delete_yaml ./cluster.yaml demo
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
| `apiVersion` / `kind` | Identifies the type (`gpupaas.ai/v1alpha1`, `VMInstance`, `BareMetal` or `MKSCluster`) |
| `metadata` | Identity, scope, labels, and annotations |
| `spec` | **Desired state** — what you set on create/apply |
| `status` | **Observed state** — populated by the platform on read; do not set on apply |

Platform hierarchy:

```text
Organization (implicit — from your API token)
└── Project              metadata.name (must exist)
    ├── VMProfile, BmProfile, MKSClusterProfile, MKSNode   project-scoped, read-only
    ├── VMInstance       project or workspace scope (dev.envmgmt.io/v1)
    ├── BareMetal        project or workspace scope (dev.envmgmt.io/v1)
    ├── MKSCluster       project or workspace scope (paas.k8smgmt.io/v3)
    └── Workspace        metadata.project + metadata.workspace (must exist)
        ├── VMInstance
        ├── BareMetal
        └── MKSCluster
```

Method arguments mirror metadata scope: `Apply(ctx, obj, project, workspace)` and `List(ctx, gvk, project, workspace)` on the generic client; `V1alpha1().<Kind>s(project)` or `V1alpha1().Workspaces(project).<Kind>s(workspace)` on the typed clientset.

---

### VMInstance

**What it is for:** A **VMInstance** is the virtual machine a developer asks for (the platform's "Virtual Machine" experience). It is created from a **VMProfile** (the VM SKU), which fixes the allowed images, sizes, datacenters, boot disk range and public IP policy, and the platform provisions the VM on a host of the profile's datacenter. The wire kind is `VirtualMachine` (`dev.envmgmt.io/v1`, snake_case fields).

**Scope:** project or workspace. Use `Workspaces(project).VMInstances(workspace)` for workspace scope, or `VMInstances(project)` for a project-scoped VM.

**Backend:** Apply is an upsert by name. `Create` reads first and fails with 409 when the name exists; `Update` applies a changed spec. Both return when the backend accepts the request; provisioning takes minutes, so poll `Get` or use `WaitForVMInstance`. Each VM has a hidden compute instance of the same name that holds the workflow error and log of a failure; read it with `ComputeInstances(project)`, which asks for hidden instances.

| Method | Route | Notes |
|--------|-------|-------|
| Create / Update | `POST .../virtualmachines` | Upsert by name; `spec.type` cannot change after create |
| Get / List | `GET .../virtualmachines[/{name}]` | |
| Delete | `DELETE .../virtualmachines/{name}` | Starts a teardown; the record disappears when it succeeds |
| Action | `POST .../virtualmachines/{name}/action/{verb}` | `start`, `stop`, `reboot`, ...; the payload is optional |
| VMProfiles(project) | `GET .../vmprofiles[/{name}]` | Read-only |

#### Object shape

VMs are created through the typed clientset (`apply.ApplyFile` does not handle them). In YAML form the object is:

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: VirtualMachine
metadata:
  name: my-vm
  project: demo
  workspace: dev            # omit for project scope
spec:
  type: vm-sku              # the VMProfile (SKU)
  cpu_count: "2"            # cpu_count and memory must match one preset row of the SKU
  memory: "4"
  image: ubuntu-24
  ssh_key: my-key
```

#### `spec` fields

| Field | Notes |
|-------|-------|
| `type` | The VMProfile (SKU) name; immutable |
| `cpu_count`, `memory`, `gpu_count` | Strings; together they must match one preset row of the SKU (`gpu_count` is required when the SKU has GPUs). `gpu_model` and `gpu_vendor` are inherited from the SKU |
| `image` | One of the SKU's images |
| `ssh_key`, `guest_password` | Set at least one on create. The password is write-only and never returned |
| `datacenter`, `vpc`, `subnet`, `security_group` | Defaults come from the SKU |
| `assign_public_ip` | Honored only when the SKU's public IP mode is `Optional` |
| `shared_storage` | Name of a shared filesystem storage to mount |
| `user_data` | Cloud-init script as plain text |
| `boot_disk_size` | GB; within the SKU's boot disk range |
| `sharing` | `shareMode` and the workspaces or projects shared with |
| read-only | `vm_id` (device id in inventory) |

Fields the SDK does not model are kept in `Extra` and sent back on update, so applying a spec read from the API never drops server-owned fields.

`status` has `status`, `reason`, `reasonDetail` (the workflow error behind a failed reason), `action` (`deploy`, `start`, `stop`, `reboot`, `destroy`, ...) and `output{host_name, instance_id, os_name, private_ip, public_ip, server_host, user_name}`. `Phase()` returns `pending`, `in_progress`, `running`, `stopped`, `failed`, `deleting`, `delete_pending` (the delete was parked for the operator; end users cannot delete again) or `destroyfailed`; `PowerState()` derives `running`, `stopped` or `unknown`.

#### Waiting

```go
vms := cs.V1alpha1().Workspaces(project).VMInstances(workspace)
running, err := typed.WaitForVMInstance(ctx, vms, "my-vm", typed.VMInstanceWaitOptions{
    Done:   typed.VMInstanceRunning, // fails with *VMInstanceFailedError on failed / destroyfailed / delete_pending
    Stable: 2,
})
```

After an action use `typed.VMInstanceActionDone(verb)` with `Stable: 2`: right after the request the backend can still show the previous state. `typed.VMInstanceGone` with `GoneIsDone: true` waits for a delete.

#### Examples

```bash
go run ./examples/create-vm -project demo -workspace dev -name example-vm -wait
go run ./examples/vm-stop -project demo -workspace dev -name example-vm -wait
```

The in-process mock `mock/vmapi` reproduces the API (SKU validation, capacity failures, asynchronous provisioning and actions, failed deletes, the hidden compute instance) for tests.

---

### MKSCluster

**What it is for:** An **MKSCluster** is a first-class managed Kubernetes cluster (the platform's "Kubernetes SKU" experience). It is created from a **ClusterProfile**, which fixes the allowed Kubernetes versions, CNIs, compute SKUs, images and datacenters. The platform provisions the nodes (bare metal or VMs from the profile's SKUs), installs Kubernetes and keeps a mirror compute instance of the same name.

**Scope:** project scope (administrators) and workspace scope (developers). The wire `apiVersion` is `paas.k8smgmt.io/v3`; the SDK normalizes reads to `gpupaas.ai/v1alpha1`. `ClusterProfile` and `ClusterNode` are project-scoped and read-only.

Clusters are **declarative**. `Create` stores a new cluster (it fails with 409 if the name exists, because the backend upserts by name); `Update` applies a changed spec: change `spec.kubernetesVersion` to upgrade, edit `spec.workerPools` to add, remove or resize pools. Both return when the backend accepts the request; provisioning takes many minutes, so poll `Get` or use `WaitForMKSCluster`.

**Backend mapping** (`paas.k8smgmt.io/v3`):

| SDK operation | Backend call | Notes |
|---------------|--------------|-------|
| Create | `GET .../clusters/{name}` then `POST .../clusters` | Checks the name is free first |
| Update / Apply | `POST .../clusters` | Upsert; send the spec read from `Get` with your edits (the SDK keeps unmodeled fields and the `version` revision) |
| Get / List | `GET .../clusters[/{name}]` | |
| Delete | `DELETE .../clusters/{name}` | Asynchronous: `DESTROYPENDING`, `DESTROYING`, then gone; `DESTROYFAILED` can be retried with another DELETE |
| MKSClusterProfiles(project) | `GET .../clusterprofiles[/{name}]` | Read-only |
| MKSClusterProfiles(project).CreateOptions | `GET .../clusterprofiles/{name}/createoptions` | What the SKU allows for the control plane and the workers |
| MKSClusterProfiles(project).AvailableCapacity | `GET .../clusterprofiles/{name}/availablecapacity` | The answer is returned as received (`Raw`): the reference does not describe it |
| MKSNodes(project).ListForCluster | `GET .../clusternodes?cluster_name=` | Read-only; labels carry `nodepoolname` (`master` for control plane); v3.1.41 controllers also report private IP, roles, OS and status |
| MKSClusters(...).AddNodePool / RemoveNodePool | `POST .../clusters/{c}/nodepools`, `DELETE .../nodepools/{pool}` | The dedicated routes for one pool; an unset `assignPublicIp` takes the existing workers' value |
| MKSClusters(...).Upgrade | `POST .../clusters/{c}/upgrade` | Send only what changes (`K8sVersion`, `Blueprint`, `BlueprintVersion`, `CNI`); one minor version at a time, never down |
| MKSClusters(...).UpdateMetadata | `PATCH .../clusters/{c}/metadata` | Labels, annotations and tags without a redeploy; a nil map is left alone, a map with its `Replace` flag replaces, an empty map clears |
| MKSClusters(...).Redeploy | `POST .../clusters/{c}/redeploy` | Only from ERROR or DEGRADED (412 otherwise) |
| MKSClusters(...).Status | `GET .../clusters/{c}/status` | The declared pools joined with the live nodes, their addresses and conditions, and the health |
| MKSClusters(...).AuditEvents | `GET .../clusters/{c}/auditevents` | Filters: action, result, resource type, actor, from, to; paged |
| MKSNodes(project).CordonNode / UncordonNode / DrainNode / DeleteNode | `POST .../clusternodes/{host}/{verb}?cluster_name=`, `DELETE .../clusternodes/{host}` | One node, by hostname |
| MKSNodes(project).UpdateNodeMetadata / UpdateNodeTaints | `PUT .../clusternodes/{host}/metadata`, `.../taints` | Labels and annotations as a per-key change; taints replace the whole set (empty clears) |
| MKSMatchmaker(project) | `GET .../clustermatchmakers/{kubernetesversions,availableos,availableimages,availablespecs,matchskus,availablecomputeskus}` | Read-only pickers of the console. Datacenters repeat; SKU kinds and compute types are joined with `+`. Keys are read as camelCase or snake_case |

The day-2 calls address the project route whatever the scope of the client: cluster names are unique per organization. They return the
cluster as the platform answers, usually still in the phase the operation started: wait with `WaitForMKSCluster`.

The v3.1.41 controllers validate pools strictly: every new pool names a VPC and a subnet (`Network`), the control plane has 1 or 3 nodes and keeps
its size (convert to HA is not implemented), every worker pool carries the same `AssignPublicIP`, a pool's datacenter is the cluster's location,
a boot disk is for virtual-machine pools only, and `SSHKey` is the name of an SSH key resource. The mock in `mock/clusterapi` enforces these rules
with `Options{PerPoolNetwork: true}`.

Workspace scope: `cs.V1alpha1().Workspaces(project).MKSClusters(workspace)`. The generic client (`client.Apply/Get/List/Delete`, `apply.ApplyFile`) supports `MKSCluster` at both scopes.

#### Create / Apply

```yaml
apiVersion: gpupaas.ai/v1alpha1
kind: MKSCluster
metadata:
  name: demo-cluster
  project: system-catalog
  workspace: team-a            # omit for project scope
spec:
  profile:
    name: mks-oneclick-1
  kubernetesVersion: "1.36"    # short form accepted; the platform resolves the latest patch
  controlPlane:
    name: control-plane
    skuName: bmaas-test-1
    skuType: BMAAS
    osImage: ubuntu-26
    sshKey: mine
    scaling: { mode: manual, count: 1 }
  workerPools:
    - name: pool-a
      skuName: bmaas-test-1
      skuType: BMAAS
      osImage: ubuntu-26
      scaling: { mode: manual, count: 2 }
```

Leave out what the profile decides (SKU, image, datacenter, networking, blueprint): the platform fills them in and returns them on `Get`.

#### `spec` fields

| Field | Type | Description |
|-------|------|-------------|
| `profile.name` | string | The ClusterProfile (Kubernetes SKU) |
| `kubernetesVersion` | string | For example `1.36` or `v1.36.4`; changing it upgrades the cluster |
| `location` | string | Datacenter; defaults to the profile's |
| `blueprint` | `MKSBlueprint` | `name` + `version`; defaults to the profile's |
| `networking` | `MKSNetworking` | `vpc`, `subnet`, `podSubnet`, `serviceSubnet`, `cni{name,version}` |
| `controlPlane` | `MKSNodeGroup` | Control-plane nodes: `name`, `datacenter`, `osImage`, `skuName`, `skuType` (`BMAAS`/`VMAAS`), `sshKey`, `scaling{mode,count}`; the platform adds `computeType`, `shape`, `skuDisplayName` |
| `workerPools` | `[]MKSNodeGroup` | Worker pools, identified by `name`. An empty list removes every pool |
| `tags` | map | Free-form tags |
| `billing` | `MKSBilling` | `variables.ContractTerm` (months, `0` = on demand) |
| `sharing` | `DevSharingSpec` | Cross-project sharing |

Fields the SDK does not model are kept in `Extra` (on the spec and on each node group) and sent back on update, so applying a spec read from the API never drops server-owned fields.

#### `status` fields (read-only)

| Field | Description |
|-------|-------------|
| `status` | Enum name, for example `MKS_CLUSTER_STATUS_RUNNING`; `Phase()` returns `running`. Others: `provisioning`, `updating`, `upgrading`, `degraded`, `error`, `destroypending`, `destroying`, `destroyfailed` |
| `reason`, `action` | Human-readable reason and the last action (`deploy`, `redeploy`, `destroy`) |
| `output.clusterIdEdgesrv` | Identifier in the Kubernetes management plane |

#### Waiting

```go
clusters := cs.V1alpha1().Workspaces(project).MKSClusters(workspace)
created, _ := clusters.Create(ctx, cluster, gpupaas.CreateOptions{})
running, err := typed.WaitForMKSCluster(ctx, clusters, "demo-cluster", typed.MKSWaitOptions{
    Done:   typed.MKSClusterRunning, // fails with *MKSClusterFailedError on ERROR / DESTROYFAILED
    Stable: 2,
})
```

After an update use `typed.MKSClusterMatches(desiredSpec)` with `Stable: 3`: the backend can still report RUNNING with the old spec for a moment after accepting a write, and can flap RUNNING, UPDATING, RUNNING while it finishes. `typed.MKSClusterGone` with `GoneIsDone: true` waits for a delete. For resizes, `MKSNodes(project).ListForCluster` shows when new nodes joined (the status can return to RUNNING before they do).

#### Kubeconfig and node pool operations

```go
clusters := cs.V1alpha1().MKSClusters(project)
kubeconfig, err := clusters.Kubeconfig(ctx, "demo-cluster") // user kubeconfig of a registered, running cluster

nodes := cs.V1alpha1().MKSNodes(project)
res, err := nodes.CordonPool(ctx, "demo-cluster", "pool-a") // also UncordonPool and DrainPool(ctx, cluster, pool, typed.MKSDrainOptions{})
```

A pool operation answers with the number of nodes it tried and one result per node. Read the per-node messages: the platform has reported `succeeded` for a node whose message was a kubectl connection error.

#### Examples

```bash
go run ./examples/create-mks-cluster -memory -project demo -name my-cluster
go run ./examples/get-mks-cluster -memory -project demo -name my-cluster
go run ./examples/list-mks-clusters -memory -project demo
go run ./examples/delete-mks-cluster -memory -project demo -name my-cluster
GPUPAAS_API_KEY=... go run ./examples/mks-cluster-upgrade -project demo -workspace team-a -name my-cluster -k8s-version 1.36
GPUPAAS_API_KEY=... go run ./examples/list-mks-nodes -project demo -name my-cluster
```

The in-process mock `mock/clusterapi` reproduces the API (profile validation, derived fields, revisions, status flaps, failure injection) for tests.


---

### BareMetal

**What it is for:** A **BareMetal** is the bare-metal machine a developer asks for (the platform's "Bare Metal" experience). It is created from a **BmProfile** (the bare-metal SKU), which fixes the allowed images, datacenters, public IP policy and contract terms, and the platform provisions a machine from the SKU's inventory.

**Scope:** project or workspace. Use `Workspaces(project).BareMetals(workspace)` for workspace scope, or `BareMetals(project)` for a project-scoped machine.

**Backend:** `dev.envmgmt.io/v1`. Apply is an upsert by name. `Create` reads first and fails with 409 when the name exists; `Update` applies a changed spec. Both return when the backend accepts the request; provisioning takes many minutes, so poll `Get` or use `WaitForBareMetal`. Each machine has a hidden compute instance of the same name (`spec.computeInstanceId`) that holds the workflow error and log of a failure; read it with `ComputeInstances(project)`, which asks for hidden instances.

| Method | Route | Notes |
|--------|-------|-------|
| Create / Update | `POST .../baremetals` | Upsert by name; `spec.type` cannot change after create |
| Get / List | `GET .../baremetals[/{name}]` | |
| Delete | `DELETE .../baremetals/{name}` | Starts a teardown; the record disappears when it succeeds |
| Action | `POST .../baremetals/{name}/action/{verb}` | `start`, `stop`, `reboot`, `Reinstall OS` (takes `{image, preserveUserConfiguration}`) |
| BmProfiles(project) | `GET .../bmprofiles[/{name}]` | Read-only |

#### Spec

| Field | Notes |
|-------|-------|
| `type` | The BmProfile (SKU) name; immutable |
| `image` | One of the SKU's images; required on create |
| `sshKey`, `userPassword` | Set at least one on create. The password is write-only and never returned |
| `datacenter`, `vpc`, `subnet`, `securityGroup` | Defaults come from the SKU |
| `assignPublicIp` | Allowed when the SKU's policy is `Enabled` or `CustomerSelectable` |
| `hssStorage`, `objectStorage` | Names of storages to attach |
| `userData` | Cloud-init script as plain text; the service encodes it |
| `contractTerm` | Months; must be one the SKU offers |
| `sharing` | `shareMode` and the workspaces or projects shared with |
| read-only | `computeInstanceId`, `provisionerType` (`bcm` or `non_bcm`), `gpuName` |

`status` has `status`, `reason`, `action` (`deploy`, `start`, `stop`, `reboot`, `destroy`, ...) and `output{hostName, privateIp, publicIp, userName, ...}`. `Phase()` returns `pending`, `in_progress`, `running`, `stopped`, `failed`, `deleting`, `delete_pending` (the delete was parked for the operator; end users cannot delete again) or `destroyfailed`.

#### Waiting

```go
machines := cs.V1alpha1().Workspaces(project).BareMetals(workspace)
running, err := typed.WaitForBareMetal(ctx, machines, "demo-machine", typed.BareMetalWaitOptions{
    Done:   typed.BareMetalRunning, // fails with *BareMetalFailedError on failed / destroyfailed / delete_pending
    Stable: 2,
})
```

After an action use `typed.BareMetalActionDone(verb)` with `Stable: 2`: right after the request the backend can still show the previous state. `typed.BareMetalGone` with `GoneIsDone: true` waits for a delete; a failed record that has not started its teardown can still read `failed` with the `deploy` action for a moment, so tolerate a few such polls.

The in-process mock `mock/bmapi` reproduces the API (SKU validation, capacity failures, asynchronous provisioning and actions, failed deletes, the hidden compute instance) for tests.


---

### SshKey (read-only)

An **SshKey** is a key a VM, bare-metal machine or cluster node can name in `sshKey` / `ssh_key`. The SDK only looks keys up, for example to check that a key a cluster names exists. Keys are created elsewhere.

```go
key, err := cs.V1alpha1().Workspaces(project).SshKeys(workspace).Get(ctx, "my-key", gpupaas.GetOptions{}) // or SshKeys(project) for project scope
if gpupaas.IsNotFound(err) {
    // the key does not exist
}
```

`GET /apis/dev.envmgmt.io/v1/projects/{project}/[workspaces/{workspace}/]sshkeys/{name}`.

---

### ComputeInstance (read-only)

**What it is for:** Every first-class VM, bare-metal machine and MKS cluster or node has a **ComputeInstance** of the same name behind it (`paas.envmgmt.io/v1`). The platform hides these instances from normal listings, so the client asks for them with `include_first_class=true`. They hold what the first-class record does not: the workflow error and the provisioning log of a failure, and the teardown that outlives the record on delete. The SDK reads them only; it does not create, publish, change or delete them.

**Scope:** project or workspace. Use `Workspaces(project).ComputeInstances(workspace)` for workspace scope, or `ComputeInstances(project)` for a project-scoped instance.

| Method | Route | Notes |
|--------|-------|-------|
| Get | `GET .../computeinstances/{name}?include_first_class=true` | |
| List | `GET .../computeinstances?include_first_class=true` | The server caps a page at 50 items whatever `Limit` you ask for; follow `list.Metadata.Continue` (an offset) until a page is empty |
| FailureLog | `GET /apis/eaas.envmgmt.io/v1/projects/{project}/environments/{env}/events/{trigger}[/activity/{id}/stdout]` | The WARNING and ERROR lines of the failed activities of the last run |

```go
ci := cs.V1alpha1().Workspaces("demo").ComputeInstances("dev")
inst, err := ci.Get(ctx, "my-vm", gpupaas.GetOptions{})
if err == nil && inst.Status.Failed() {
    fmt.Println(inst.Status.Reason, inst.Status.WorkflowStatus) // per-task errors
    // The log usually names the real cause when status.reason is only "Failed to deploy".
    if failure, err := ci.FailureLog(ctx, "my-vm", 12); err == nil {
        fmt.Println(failure.Lines)
    }
}
```

Notes from the live API:

- **Missing objects answer HTTP 502** with a body like `{"internal":"ent: workspace_compute_instance not found","external":"Upstream misbehaving"}`. `gpupaas.IsNotFound` recognises this; the error message is taken from `internal`.
- A failed run reports `status.reason` and `status.workflowStatus` (per-task errors). The reason is often generic; `FailureLog` returns the detailed lines from the environment activity output. Log text can name hosts and addresses: show it to the operator but do not store it.
- `status.environment` and `status.provisionedResource` (with the id of the latest run, `TriggerID()`) are present only after a run succeeded or failed; a destroy failure has neither.
- `Phase()` normalizes the raw status (`not_deployed`, `success`, `failed`, `destroyfailed`, ...); `InProgress()` and `Failed()` classify it.
- **Which organization a request acts in depends on the kind of API key.** An organization-level key is bound to its organization and needs nothing more; the platform ignores any `X-Organization-ID` sent with it. A partner-level key is not bound to an organization, so set `Config.OrganizationID` (env `GPUPAAS_ORGANIZATION_ID`) to the organization's id or its exact, case-sensitive name; it becomes the `X-Organization-ID` header and the organization must belong to the key's partner. Without it the platform answers 502 `id or partnerID/orgID/name not set`; an unknown id or name answers 500 `get organization by name returned error`. The SDK keeps those messages and appends what to set.

#### `status` fields (read-only)

| Field | Description |
|-------|-------------|
| `status` | Raw status (`not_deployed`, `success`, `failed`, ...); `Phase()` normalizes it |
| `reason` | Human-readable reason of the last run |
| `action` | Last action: `publish`, `start`, `stop`, `reboot`, `destroy`, ... |
| `workflowStatus` | Per-task result, including errors |
| `environment`, `provisionedResource` | The EaaS environment and run |

---

### Shared metadata conventions

| Field | Used on | Notes |
|-------|---------|-------|
| `metadata.name` | All resources | Primary identifier; immutable after create |
| `metadata.project` | VMInstance, BareMetal, MKSCluster, SshKey, ComputeInstance | Parent project; required for scoped resources |
| `metadata.workspace` | VMInstance, BareMetal, MKSCluster, SshKey, ComputeInstance (workspace scope) | Parent workspace within a project |
| `metadata.displayName` | All | Optional human-friendly name. Sent on writes. |
| `metadata.description` | All | Optional human-readable description. Sent on writes. |
| `metadata.labels` | All | Selectors and automation hooks |
| `metadata.annotations` | All | Tooling-specific key/value data |
| `metadata.createdBy` | All (read-only) | `UserMeta` populated by the backend on reads; **ignored on writes** |
| `metadata.modifiedBy` | All (read-only) | `UserMeta` populated by the backend on reads; **ignored on writes** |

`UserMeta` carries `username`, `isSSOUser`, and an optional `options` block (`description`, `required`, `override.type`, `override.restrictedValues`). It is read-only; the converters strip it before sending.

When applying YAML with `apply.ApplyFile`, pass the default project (and workspace when needed) as arguments; empty fields on the object are filled from those defaults.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
