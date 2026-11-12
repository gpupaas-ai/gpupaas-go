package v1alpha1

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

// First-class managed Kubernetes (MKS), served by the platform under
// paas.k8smgmt.io/v3. A Cluster is created from a ClusterProfile (the Kubernetes SKU) and
// is fully declarative: the Kubernetes version, the worker pools and their sizes change by
// applying a new spec.

// ---------------------------------------------------------------------------
// Spec building blocks
// ---------------------------------------------------------------------------

// MKSProfileRef names the ClusterProfile a cluster is created from.
type MKSProfileRef struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

// MKSBlueprint references the blueprint applied to a cluster.
type MKSBlueprint struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// MKSCNI selects the container network interface and its version.
type MKSCNI struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// MKSNetworking holds cluster networking configuration.
type MKSNetworking struct {
	VPC           string  `json:"vpc,omitempty" yaml:"vpc,omitempty"`
	Subnet        string  `json:"subnet,omitempty" yaml:"subnet,omitempty"`
	PodSubnet     string  `json:"podSubnet,omitempty" yaml:"podSubnet,omitempty"`
	ServiceSubnet string  `json:"serviceSubnet,omitempty" yaml:"serviceSubnet,omitempty"`
	CNI           *MKSCNI `json:"cni,omitempty" yaml:"cni,omitempty"`
}

// MKSScaling sizes a node group. The backend omits zero values, so a count of 0 and an
// unset count look the same on the wire. Min, Desired and Max are reported by the backend
// for autoscaled groups.
type MKSScaling struct {
	Mode    string `json:"mode,omitempty" yaml:"mode,omitempty"` // manual
	Count   int32  `json:"count,omitempty" yaml:"count,omitempty"`
	Min     int32  `json:"min,omitempty" yaml:"min,omitempty"`
	Desired int32  `json:"desired,omitempty" yaml:"desired,omitempty"`
	Max     int32  `json:"max,omitempty" yaml:"max,omitempty"`
}

// MKSNodePoolNetwork places the nodes of a pool in a VPC and subnet. Both are required for a new
// pool on controllers that take the network per pool, and fixed once the pool exists. SecurityGroup
// is optional and passed to the VM and bare-metal APIs as given.
type MKSNodePoolNetwork struct {
	VPC           string `json:"vpc,omitempty" yaml:"vpc,omitempty"`
	Subnet        string `json:"subnet,omitempty" yaml:"subnet,omitempty"`
	SecurityGroup string `json:"securityGroup,omitempty" yaml:"securityGroup,omitempty"`
}

// MKSInt is an integer that the backend may write as a JSON number or, for 64-bit protobuf fields, as a
// quoted string ("60"). It is always written as a number.
type MKSInt int32

// UnmarshalJSON accepts 60 and "60".
func (n *MKSInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		*n = 0
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return fmt.Errorf("%q is not a 32-bit integer", s)
	}
	*n = MKSInt(v)
	return nil
}

// MKS scaling modes of a node group.
const (
	MKSScalingManual    = "manual"
	MKSScalingAutomatic = "automatic"
)

// MKS node taint effects.
const (
	MKSTaintNoSchedule       = "NoSchedule"
	MKSTaintPreferNoSchedule = "PreferNoSchedule"
	MKSTaintNoExecute        = "NoExecute"
)

// Compute SKU families and classes of a node group. The backend stamps both from the compute SKU on
// every save, so a value sent by a client is advisory.
const (
	MKSSKUTypeVM        = "VMAAS"
	MKSSKUTypeBareMetal = "BMAAS"

	MKSComputeCPU = "CPU"
	MKSComputeGPU = "GPU"
	// MKSComputeCustom is a bare-metal pool without a catalog GPU. The backend stores CUSTOM even
	// when CPU was sent.
	MKSComputeCustom = "CUSTOM"
)

// MKSBilling carries billing variables such as the contract term.
type MKSBilling struct {
	Variables map[string]string `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// MKSNodeGroup describes the control plane or one worker pool. Fields the backend derives
// (computeType, shape, skuDisplayName, skuDisplayText) are read back and sent unchanged.
// Anything else the backend returns is kept in Extra and sent back on update, so applying a
// spec read from the API never drops server-owned fields.
type MKSNodeGroup struct {
	Name           string `json:"name,omitempty" yaml:"name,omitempty"`
	DisplayName    string `json:"displayName,omitempty" yaml:"displayName,omitempty"`
	Datacenter     string `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	OSImage        string `json:"osImage,omitempty" yaml:"osImage,omitempty"`
	SKUName        string `json:"skuName,omitempty" yaml:"skuName,omitempty"`
	SKUType        string `json:"skuType,omitempty" yaml:"skuType,omitempty"` // BMAAS | VMAAS, stamped from the SKU
	SSHKey         string `json:"sshKey,omitempty" yaml:"sshKey,omitempty"`
	ComputeType    string `json:"computeType,omitempty" yaml:"computeType,omitempty"` // CPU | GPU | CUSTOM, stamped from the SKU
	SKUDisplayName string `json:"skuDisplayName,omitempty" yaml:"skuDisplayName,omitempty"`
	SKUDisplayText string `json:"skuDisplayText,omitempty" yaml:"skuDisplayText,omitempty"`
	// Network is the pool's VPC, subnet and security group. Fixed after the pool is created.
	Network *MKSNodePoolNetwork `json:"network,omitempty" yaml:"network,omitempty"`
	// BootDiskSizeGb is the boot disk of a virtual-machine pool. Fixed after the pool is created.
	BootDiskSizeGb MKSInt `json:"bootDiskSizeGb,omitempty" yaml:"bootDiskSizeGb,omitempty"`
	// AssignPublicIP is a pointer so that false can be sent; every worker pool must carry the same value.
	AssignPublicIP *bool `json:"assignPublicIp,omitempty" yaml:"assignPublicIp,omitempty"`
	// UserPassword is the node login password. The backend never returns it: a read gives "******".
	UserPassword string            `json:"userPassword,omitempty" yaml:"userPassword,omitempty"`
	Scaling      *MKSScaling       `json:"scaling,omitempty" yaml:"scaling,omitempty"`
	Shape        map[string]any    `json:"shape,omitempty" yaml:"shape,omitempty"`
	Labels       map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Taints       []any             `json:"taints,omitempty" yaml:"taints,omitempty"`
	Tags         map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Extra holds fields this SDK does not model.
	Extra map[string]json.RawMessage `json:"-" yaml:"-"`
}

type mksNodeGroupAlias MKSNodeGroup

// MarshalJSON writes the modeled fields plus Extra.
func (g MKSNodeGroup) MarshalJSON() ([]byte, error) {
	return marshalWithExtras(mksNodeGroupAlias(g), g.Extra)
}

// UnmarshalJSON reads the modeled fields and keeps the rest in Extra.
func (g *MKSNodeGroup) UnmarshalJSON(b []byte) error {
	var a mksNodeGroupAlias
	extra, err := unmarshalWithExtras(b, &a)
	if err != nil {
		return err
	}
	*g = MKSNodeGroup(a)
	g.Extra = extra
	return nil
}

// MKSClusterSpec holds desired cluster state.
type MKSClusterSpec struct {
	Profile           *MKSProfileRef    `json:"profile,omitempty" yaml:"profile,omitempty"`
	KubernetesVersion string            `json:"kubernetesVersion,omitempty" yaml:"kubernetesVersion,omitempty"`
	Location          string            `json:"location,omitempty" yaml:"location,omitempty"`
	Blueprint         *MKSBlueprint     `json:"blueprint,omitempty" yaml:"blueprint,omitempty"`
	Networking        *MKSNetworking    `json:"networking,omitempty" yaml:"networking,omitempty"`
	ControlPlane      *MKSNodeGroup     `json:"controlPlane,omitempty" yaml:"controlPlane,omitempty"`
	WorkerPools       []MKSNodeGroup    `json:"workerPools" yaml:"workerPools,omitempty"`
	Tags              map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Billing           *MKSBilling       `json:"billing,omitempty" yaml:"billing,omitempty"`
	Sharing           *DevSharingSpec   `json:"sharing,omitempty" yaml:"sharing,omitempty"`

	// Extra holds top-level spec fields this SDK does not model.
	Extra map[string]json.RawMessage `json:"-" yaml:"-"`
}

type mksClusterSpecAlias MKSClusterSpec

// MarshalJSON writes the modeled fields plus Extra. workerPools is always written (an empty
// list removes every pool on apply).
func (s MKSClusterSpec) MarshalJSON() ([]byte, error) {
	if s.WorkerPools == nil {
		s.WorkerPools = []MKSNodeGroup{}
	}
	return marshalWithExtras(mksClusterSpecAlias(s), s.Extra)
}

// UnmarshalJSON reads the modeled fields and keeps the rest in Extra.
func (s *MKSClusterSpec) UnmarshalJSON(b []byte) error {
	var a mksClusterSpecAlias
	extra, err := unmarshalWithExtras(b, &a)
	if err != nil {
		return err
	}
	*s = MKSClusterSpec(a)
	s.Extra = extra
	return nil
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

// MKSClusterOutput holds provisioning output reported by the backend.
type MKSClusterOutput struct {
	ClusterIDEdgesrv  string `json:"clusterIdEdgesrv,omitempty" yaml:"clusterIdEdgesrv,omitempty"`
	APIServerEndpoint string `json:"apiServerEndpoint,omitempty" yaml:"apiServerEndpoint,omitempty"`
}

// MKSClusterStatus holds observed cluster state. Status is the backend enum name, for
// example MKS_CLUSTER_STATUS_RUNNING; use Phase for the normalized lowercase form.
type MKSClusterStatus struct {
	Status string            `json:"status,omitempty" yaml:"status,omitempty"`
	Reason string            `json:"reason,omitempty" yaml:"reason,omitempty"`
	Action string            `json:"action,omitempty" yaml:"action,omitempty"`
	Output *MKSClusterOutput `json:"output,omitempty" yaml:"output,omitempty"`
	// AvailableKubernetesVersions are the versions the cluster can be upgraded to (one minor version
	// at a time, never a downgrade).
	AvailableKubernetesVersions []string `json:"availableKubernetesVersions,omitempty" yaml:"availableKubernetesVersions,omitempty"`
}

// Cluster phases (the status enum without its MKS_CLUSTER_STATUS_ prefix, lowercase).
const (
	MKSPhaseProvisioning  = "provisioning"
	MKSPhaseRunning       = "running"
	MKSPhaseUpdating      = "updating"
	MKSPhaseUpgrading     = "upgrading"
	MKSPhaseDegraded      = "degraded"
	MKSPhaseError         = "error"
	MKSPhaseDestroyPend   = "destroypending"
	MKSPhaseDestroying    = "destroying"
	MKSPhaseDestroyFailed = "destroyfailed"
)

// Phase returns the normalized status: "MKS_CLUSTER_STATUS_RUNNING" becomes "running".
func (s MKSClusterStatus) Phase() string {
	p := strings.ToLower(strings.TrimSpace(s.Status))
	return strings.TrimPrefix(p, "mks_cluster_status_")
}

// Running reports whether the cluster is up.
func (s MKSClusterStatus) Running() bool { return s.Phase() == MKSPhaseRunning }

// Failed reports a terminal failure phase.
func (s MKSClusterStatus) Failed() bool {
	switch s.Phase() {
	case MKSPhaseError, MKSPhaseDestroyFailed:
		return true
	}
	return false
}

// Deleting reports that teardown has started.
func (s MKSClusterStatus) Deleting() bool {
	switch s.Phase() {
	case MKSPhaseDestroyPend, MKSPhaseDestroying:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// MKSCluster
// ---------------------------------------------------------------------------

// MKSCluster is a first-class managed Kubernetes cluster.
type MKSCluster struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta       `json:"metadata" yaml:"metadata"`
	Spec     MKSClusterSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   MKSClusterStatus `json:"status,omitempty" yaml:"status,omitempty"`
	// Version is the revision the backend reports for the cluster. It is sent back on update
	// so the backend can detect a concurrent change.
	Version int64 `json:"version,omitempty" yaml:"version,omitempty"`
}

func (c *MKSCluster) GetAPIVersion() string { return c.APIVersion }
func (c *MKSCluster) GetKind() string       { return c.Kind }
func (c *MKSCluster) GetName() string       { return c.Metadata.Name }
func (c *MKSCluster) GetProject() string    { return c.Metadata.Project }
func (c *MKSCluster) GetWorkspace() string  { return c.Metadata.Workspace }
func (c *MKSCluster) SetProject(v string)   { c.Metadata.Project = v }
func (c *MKSCluster) SetWorkspace(v string) { c.Metadata.Workspace = v }
func (c *MKSCluster) DeepCopyObject() runtime.Object {
	cp := *c
	cp.Metadata = copyObjectMeta(c.Metadata)
	cp.Spec = jsonCopy(c.Spec)
	cp.Status = jsonCopy(c.Status)
	return &cp
}

// MKSClusterList is a list of MKSCluster resources.
type MKSClusterList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta     `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []MKSCluster `json:"items" yaml:"items"`
}

func (l *MKSClusterList) GetAPIVersion() string { return l.APIVersion }
func (l *MKSClusterList) GetKind() string       { return l.Kind }
func (l *MKSClusterList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *MKSClusterList) SetItems(items []runtime.Object) {
	l.Items = make([]MKSCluster, len(items))
	for i, item := range items {
		if c, ok := item.(*MKSCluster); ok {
			l.Items[i] = *c
		}
	}
}

// ---------------------------------------------------------------------------
// ClusterProfile (the Kubernetes SKU, read-only)
// ---------------------------------------------------------------------------

// MKSGroupProfile says what a profile allows for the control plane or the workers.
type MKSGroupProfile struct {
	ComputeTypes   []string `json:"computeTypes,omitempty" yaml:"computeTypes,omitempty"` // CPU | GPU | CUSTOM (CUSTOM needs BMAAS)
	DefaultImage   string   `json:"defaultImage,omitempty" yaml:"defaultImage,omitempty"`
	DefaultSKUName string   `json:"defaultSkuName,omitempty" yaml:"defaultSkuName,omitempty"`
	Images         []string `json:"images,omitempty" yaml:"images,omitempty"`
	SKUNames       []string `json:"skuNames,omitempty" yaml:"skuNames,omitempty"`
	SKUTypes       []string `json:"skuTypes,omitempty" yaml:"skuTypes,omitempty"` // BMAAS | VMAAS
}

// MKSClusterProfileSpec describes a Kubernetes SKU.
type MKSClusterProfileSpec struct {
	ControlPlane                *MKSGroupProfile `json:"controlPlane,omitempty" yaml:"controlPlane,omitempty"`
	Workers                     *MKSGroupProfile `json:"workers,omitempty" yaml:"workers,omitempty"`
	Datacenters                 []string         `json:"datacenters,omitempty" yaml:"datacenters,omitempty"`
	SupportedCNIs               []MKSCNI         `json:"supportedCnis,omitempty" yaml:"supportedCnis,omitempty"`
	DefaultCNI                  *MKSCNI          `json:"defaultCni,omitempty" yaml:"defaultCni,omitempty"`
	DefaultBlueprint            *MKSBlueprint    `json:"defaultBlueprint,omitempty" yaml:"defaultBlueprint,omitempty"`
	DefaultKubernetesVersion    string           `json:"defaultKubernetesVersion,omitempty" yaml:"defaultKubernetesVersion,omitempty"`
	SupportedKubernetesVersions []string         `json:"supportedKubernetesVersions,omitempty" yaml:"supportedKubernetesVersions,omitempty"`
	DefaultSSHKey               string           `json:"defaultSshKey,omitempty" yaml:"defaultSshKey,omitempty"`
	// ExposeSSHKeyChoice makes a new cluster need a node password or an SSH key.
	ExposeSSHKeyChoice bool `json:"exposeSshKeyChoice,omitempty" yaml:"exposeSshKeyChoice,omitempty"`
	// DefaultUserPassword is the node password a cluster gets when it sets none. The platform never
	// shows it: a profile that holds one reads back "******".
	DefaultUserPassword string `json:"defaultUserPassword,omitempty" yaml:"defaultUserPassword,omitempty"`
	// ContractTermsMonths are the billing terms a cluster may pick.
	ContractTermsMonths  []int32 `json:"contractTermsMonths,omitempty" yaml:"contractTermsMonths,omitempty"`
	DefaultPodSubnet     string  `json:"defaultPodSubnet,omitempty" yaml:"defaultPodSubnet,omitempty"`
	DefaultServiceSubnet string  `json:"defaultServiceSubnet,omitempty" yaml:"defaultServiceSubnet,omitempty"`
	WorkerBatchSize      int32   `json:"workerBatchSize,omitempty" yaml:"workerBatchSize,omitempty"`
	Version              string  `json:"version,omitempty" yaml:"version,omitempty"`
}

// MKSClusterProfileStatus holds observed profile state.
type MKSClusterProfileStatus struct {
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
}

// MKSClusterProfile is a Kubernetes SKU. Administrators manage profiles; the SDK reads them so
// callers can learn which versions, CNIs, SKUs, images and datacenters a cluster may use.
type MKSClusterProfile struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta              `json:"metadata" yaml:"metadata"`
	Spec     MKSClusterProfileSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   MKSClusterProfileStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (p *MKSClusterProfile) GetAPIVersion() string { return p.APIVersion }
func (p *MKSClusterProfile) GetKind() string       { return p.Kind }
func (p *MKSClusterProfile) GetName() string       { return p.Metadata.Name }
func (p *MKSClusterProfile) GetProject() string    { return p.Metadata.Project }
func (p *MKSClusterProfile) GetWorkspace() string  { return p.Metadata.Workspace }
func (p *MKSClusterProfile) SetProject(v string)   { p.Metadata.Project = v }
func (p *MKSClusterProfile) SetWorkspace(v string) { p.Metadata.Workspace = v }
func (p *MKSClusterProfile) DeepCopyObject() runtime.Object {
	cp := *p
	cp.Metadata = copyObjectMeta(p.Metadata)
	cp.Spec = jsonCopy(p.Spec)
	return &cp
}

// MKSClusterProfileList is a list of MKSClusterProfile resources.
type MKSClusterProfileList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta            `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []MKSClusterProfile `json:"items" yaml:"items"`
}

func (l *MKSClusterProfileList) GetAPIVersion() string { return l.APIVersion }
func (l *MKSClusterProfileList) GetKind() string       { return l.Kind }
func (l *MKSClusterProfileList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *MKSClusterProfileList) SetItems(items []runtime.Object) {
	l.Items = make([]MKSClusterProfile, len(items))
	for i, item := range items {
		if p, ok := item.(*MKSClusterProfile); ok {
			l.Items[i] = *p
		}
	}
}

// ---------------------------------------------------------------------------
// ClusterNode (read-only)
// ---------------------------------------------------------------------------

// MKSTaint is a Kubernetes node taint.
type MKSTaint struct {
	Key    string `json:"key,omitempty" yaml:"key,omitempty"`
	Value  string `json:"value,omitempty" yaml:"value,omitempty"`
	Effect string `json:"effect,omitempty" yaml:"effect,omitempty"` // NoSchedule | PreferNoSchedule | NoExecute
}

// MKSNodeSpec is the observed description of a cluster node. Older controllers report only the
// hostname, architecture and kubelet version; the other fields are empty there.
type MKSNodeSpec struct {
	Hostname    string `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Arch        string `json:"arch,omitempty" yaml:"arch,omitempty"`
	KubeVersion string `json:"kubeVersion,omitempty" yaml:"kubeVersion,omitempty"`
	// Roles are the node's roles ("Master", "Worker").
	Roles           []string `json:"roles,omitempty" yaml:"roles,omitempty"`
	PrivateIP       string   `json:"privateIp,omitempty" yaml:"privateIp,omitempty"`
	OperatingSystem string   `json:"operatingSystem,omitempty" yaml:"operatingSystem,omitempty"`
	// NodePool is the worker pool the node belongs to ("master" for control-plane nodes).
	NodePool string `json:"nodePool,omitempty" yaml:"nodePool,omitempty"`
	// SKU is the compute SKU the node was created from.
	SKU      string `json:"sku,omitempty" yaml:"sku,omitempty"`
	PublicIP bool   `json:"publicIp,omitempty" yaml:"publicIp,omitempty"`
	// NodeLabels, NodeAnnotations and NodeTaints are the live Kubernetes values of the node.
	NodeLabels      map[string]string `json:"nodeLabels,omitempty" yaml:"nodeLabels,omitempty"`
	NodeAnnotations map[string]string `json:"nodeAnnotations,omitempty" yaml:"nodeAnnotations,omitempty"`
	NodeTaints      []MKSTaint        `json:"nodeTaints,omitempty" yaml:"nodeTaints,omitempty"`
}

// MKSNodeStatus is the observed lifecycle of a cluster node.
type MKSNodeStatus struct {
	// Phase is the normalized phase, for example MKS_NODE_PHASE_RUNNING.
	Phase string `json:"phase,omitempty" yaml:"phase,omitempty"`
	// Reason explains the phase.
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
	// EdgeStatus is the platform's own status word for the node, for example READY.
	EdgeStatus string `json:"edgeStatus,omitempty" yaml:"edgeStatus,omitempty"`
}

// MKSNode is a node registered by a cluster. Labels carry cluster_name, nodepoolname
// ("master" for control-plane nodes) and sku_name.
type MKSNode struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta    `json:"metadata" yaml:"metadata"`
	Spec     MKSNodeSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   MKSNodeStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (n *MKSNode) GetAPIVersion() string { return n.APIVersion }
func (n *MKSNode) GetKind() string       { return n.Kind }
func (n *MKSNode) GetName() string       { return n.Metadata.Name }
func (n *MKSNode) GetProject() string    { return n.Metadata.Project }
func (n *MKSNode) GetWorkspace() string  { return n.Metadata.Workspace }
func (n *MKSNode) SetProject(v string)   { n.Metadata.Project = v }
func (n *MKSNode) SetWorkspace(v string) { n.Metadata.Workspace = v }
func (n *MKSNode) DeepCopyObject() runtime.Object {
	cp := *n
	cp.Metadata = copyObjectMeta(n.Metadata)
	cp.Spec.Roles = append([]string(nil), n.Spec.Roles...)
	cp.Spec.NodeLabels = copyStringMap(n.Spec.NodeLabels)
	cp.Spec.NodeAnnotations = copyStringMap(n.Spec.NodeAnnotations)
	cp.Spec.NodeTaints = append([]MKSTaint(nil), n.Spec.NodeTaints...)
	return &cp
}

// NodePool returns the node pool label ("master" for control-plane nodes).
func (n *MKSNode) NodePool() string { return n.Metadata.Labels["nodepoolname"] }

// MKSNodeList is a list of MKSNode resources.
type MKSNodeList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta  `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []MKSNode `json:"items" yaml:"items"`
}

func (l *MKSNodeList) GetAPIVersion() string { return l.APIVersion }
func (l *MKSNodeList) GetKind() string       { return l.Kind }
func (l *MKSNodeList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *MKSNodeList) SetItems(items []runtime.Object) {
	l.Items = make([]MKSNode, len(items))
	for i, item := range items {
		if n, ok := item.(*MKSNode); ok {
			l.Items[i] = *n
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// jsonCopy deep-copies a value through its JSON form. The types it is used on have custom
// marshalers that preserve unmodeled fields, so the copy is complete.
func jsonCopy[T any](in T) T {
	var out T
	b, err := json.Marshal(in)
	if err != nil {
		return in
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return in
	}
	return out
}

// jsonFieldNames returns the JSON object keys a struct type models.
func jsonFieldNames(t reflect.Type) map[string]bool {
	names := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			name = t.Field(i).Name
		}
		names[name] = true
	}
	return names
}

// marshalWithExtras encodes v and adds the extra keys that v does not already write.
func marshalWithExtras(v any, extra map[string]json.RawMessage) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil || len(extra) == 0 {
		return b, err
	}
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	for k, raw := range extra {
		if _, ok := m[k]; !ok {
			m[k] = raw
		}
	}
	return json.Marshal(m)
}

// unmarshalWithExtras decodes data into v (a pointer to a struct alias without methods) and
// returns the keys v does not model.
func unmarshalWithExtras(data []byte, v any) (map[string]json.RawMessage, error) {
	if err := json.Unmarshal(data, v); err != nil {
		return nil, err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, nil // not an object (null): nothing extra
	}
	known := jsonFieldNames(reflect.TypeOf(v).Elem())
	var extra map[string]json.RawMessage
	for k, raw := range all {
		if known[k] {
			continue
		}
		if extra == nil {
			extra = map[string]json.RawMessage{}
		}
		extra[k] = raw
	}
	return extra, nil
}
