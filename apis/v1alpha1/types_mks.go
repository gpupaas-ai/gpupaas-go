package v1alpha1

import "github.com/gpupaas-ai/gpupaas-go/runtime"

// ---------------------------------------------------------------------------
// Shared nested MKS types
// ---------------------------------------------------------------------------

// MKSBlueprint references the blueprint applied to a cluster.
type MKSBlueprint struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// MKSNetworking holds cluster networking configuration.
type MKSNetworking struct {
	VPC            string   `json:"vpc,omitempty" yaml:"vpc,omitempty"`
	Subnet         string   `json:"subnet,omitempty" yaml:"subnet,omitempty"`
	PodCIDR        string   `json:"podCidr,omitempty" yaml:"podCidr,omitempty"`
	ServiceCIDR    string   `json:"serviceCidr,omitempty" yaml:"serviceCidr,omitempty"`
	IPFamily       string   `json:"ipFamily,omitempty" yaml:"ipFamily,omitempty"`
	PodCIDRV6      string   `json:"podCidrV6,omitempty" yaml:"podCidrV6,omitempty"`
	ServiceCIDRV6  string   `json:"serviceCidrV6,omitempty" yaml:"serviceCidrV6,omitempty"`
	SecurityGroups []string `json:"securityGroups,omitempty" yaml:"securityGroups,omitempty"`
}

// MKSProxy holds outbound proxy configuration.
type MKSProxy struct {
	HTTPProxy    string `json:"httpProxy,omitempty" yaml:"httpProxy,omitempty"`
	HTTPSProxy   string `json:"httpsProxy,omitempty" yaml:"httpsProxy,omitempty"`
	NoProxy      string `json:"noProxy,omitempty" yaml:"noProxy,omitempty"`
	ProxyRootCA  string `json:"proxyRootCa,omitempty" yaml:"proxyRootCa,omitempty"`
	TLSTerminate *bool  `json:"tlsTerminate,omitempty" yaml:"tlsTerminate,omitempty"`
}

// MKSStorageBackend configures a single storage backend.
type MKSStorageBackend struct {
	Type          string            `json:"type,omitempty" yaml:"type,omitempty"`
	AccessMode    string            `json:"accessMode,omitempty" yaml:"accessMode,omitempty"`
	ReclaimPolicy string            `json:"reclaimPolicy,omitempty" yaml:"reclaimPolicy,omitempty"`
	Config        map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

// MKSStorage holds cluster storage configuration.
type MKSStorage struct {
	Block               *MKSStorageBackend `json:"block,omitempty" yaml:"block,omitempty"`
	SharedFS            *MKSStorageBackend `json:"sharedFs,omitempty" yaml:"sharedFs,omitempty"`
	Object              *MKSStorageBackend `json:"object,omitempty" yaml:"object,omitempty"`
	HighSpeed           *MKSStorageBackend `json:"highSpeed,omitempty" yaml:"highSpeed,omitempty"`
	DefaultStorageClass string             `json:"defaultStorageClass,omitempty" yaml:"defaultStorageClass,omitempty"`
}

// MKSKubeletConfig holds kubelet overrides.
type MKSKubeletConfig struct {
	KeyValue map[string]string `json:"keyValue,omitempty" yaml:"keyValue,omitempty"`
	YAML     string            `json:"yaml,omitempty" yaml:"yaml,omitempty"`
}

// MKSNodeGroup describes a managed group of worker (or control-plane) nodes.
type MKSNodeGroup struct {
	ID              string            `json:"id,omitempty" yaml:"id,omitempty"`
	SKU             string            `json:"sku,omitempty" yaml:"sku,omitempty"`
	ScalingMode     string            `json:"scalingMode,omitempty" yaml:"scalingMode,omitempty"`
	NodeCount       int32             `json:"nodeCount,omitempty" yaml:"nodeCount,omitempty"`
	MinNodes        int32             `json:"minNodes,omitempty" yaml:"minNodes,omitempty"`
	MaxNodes        int32             `json:"maxNodes,omitempty" yaml:"maxNodes,omitempty"`
	DesiredNodes    int32             `json:"desiredNodes,omitempty" yaml:"desiredNodes,omitempty"`
	PublicIP        *bool             `json:"publicIp,omitempty" yaml:"publicIp,omitempty"`
	SSHKey          string            `json:"sshKey,omitempty" yaml:"sshKey,omitempty"`
	UserData        string            `json:"userData,omitempty" yaml:"userData,omitempty"`
	NodeLabels      map[string]string `json:"nodeLabels,omitempty" yaml:"nodeLabels,omitempty"`
	NodeAnnotations map[string]string `json:"nodeAnnotations,omitempty" yaml:"nodeAnnotations,omitempty"`
	KubeletConfig   *MKSKubeletConfig `json:"kubeletConfig,omitempty" yaml:"kubeletConfig,omitempty"`
}

// MKSNodeSpec describes a single device-based node.
type MKSNodeSpec struct {
	Hostname        string            `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	InventorySource string            `json:"inventorySource,omitempty" yaml:"inventorySource,omitempty"`
	Roles           []string          `json:"roles,omitempty" yaml:"roles,omitempty"`
	SSHKey          string            `json:"sshKey,omitempty" yaml:"sshKey,omitempty"`
	SSHUserName     string            `json:"sshUserName,omitempty" yaml:"sshUserName,omitempty"`
	SSHPort         int32             `json:"sshPort,omitempty" yaml:"sshPort,omitempty"`
	PrivateIP       string            `json:"privateIp,omitempty" yaml:"privateIp,omitempty"`
	Arch            string            `json:"arch,omitempty" yaml:"arch,omitempty"`
	OperatingSystem string            `json:"operatingSystem,omitempty" yaml:"operatingSystem,omitempty"`
	Interface       string            `json:"interface,omitempty" yaml:"interface,omitempty"`
	NodeLabels      map[string]string `json:"nodeLabels,omitempty" yaml:"nodeLabels,omitempty"`
	NodeAnnotations map[string]string `json:"nodeAnnotations,omitempty" yaml:"nodeAnnotations,omitempty"`
	NodeTaints      map[string]string `json:"nodeTaints,omitempty" yaml:"nodeTaints,omitempty"`
	UserData        string            `json:"userData,omitempty" yaml:"userData,omitempty"`
	KubeletConfig   *MKSKubeletConfig `json:"kubeletConfig,omitempty" yaml:"kubeletConfig,omitempty"`
	NodePool        string            `json:"nodePool,omitempty" yaml:"nodePool,omitempty"`
	SKU             string            `json:"sku,omitempty" yaml:"sku,omitempty"`
	PublicIP        *bool             `json:"publicIp,omitempty" yaml:"publicIp,omitempty"`
}

// ---------------------------------------------------------------------------
// MKS sub-action request payloads
// ---------------------------------------------------------------------------

// MKSUpgradeRequest is the body for upgrading a cluster Kubernetes version.
type MKSUpgradeRequest struct {
	K8sVersion      string `json:"k8sVersion,omitempty" yaml:"k8sVersion,omitempty"`
	PlatformVersion string `json:"platformVersion,omitempty" yaml:"platformVersion,omitempty"`
}

// MKSScaleNodeGroupRequest is the body for scaling a worker node group.
type MKSScaleNodeGroupRequest struct {
	NodeGroupName string `json:"nodeGroupName" yaml:"nodeGroupName"`
	DesiredCount  *int32 `json:"desiredCount,omitempty" yaml:"desiredCount,omitempty"`
	MinCount      *int32 `json:"minCount,omitempty" yaml:"minCount,omitempty"`
	MaxCount      *int32 `json:"maxCount,omitempty" yaml:"maxCount,omitempty"`
}

// MKSDrainRequest is the body for draining a node.
type MKSDrainRequest struct {
	Force              *bool  `json:"force,omitempty" yaml:"force,omitempty"`
	IgnoreDaemonsets   *bool  `json:"ignoreDaemonsets,omitempty" yaml:"ignoreDaemonsets,omitempty"`
	GracePeriodSeconds *int32 `json:"gracePeriodSeconds,omitempty" yaml:"gracePeriodSeconds,omitempty"`
}

// ---------------------------------------------------------------------------
// MKSCluster
// ---------------------------------------------------------------------------

// MKSClusterSpec holds desired MKS cluster state.
type MKSClusterSpec struct {
	KubernetesVersion     string            `json:"kubernetesVersion,omitempty" yaml:"kubernetesVersion,omitempty"`
	PlatformVersion       string            `json:"platformVersion,omitempty" yaml:"platformVersion,omitempty"`
	CNI                   string            `json:"cni,omitempty" yaml:"cni,omitempty"`
	CNIVersion            string            `json:"cniVersion,omitempty" yaml:"cniVersion,omitempty"`
	OS                    string            `json:"os,omitempty" yaml:"os,omitempty"`
	HAEnabled             *bool             `json:"haEnabled,omitempty" yaml:"haEnabled,omitempty"`
	DedicatedControlPlane *bool             `json:"dedicatedControlPlane,omitempty" yaml:"dedicatedControlPlane,omitempty"`
	Location              string            `json:"location,omitempty" yaml:"location,omitempty"`
	Blueprint             *MKSBlueprint     `json:"blueprint,omitempty" yaml:"blueprint,omitempty"`
	Networking            *MKSNetworking    `json:"networking,omitempty" yaml:"networking,omitempty"`
	Proxy                 *MKSProxy         `json:"proxy,omitempty" yaml:"proxy,omitempty"`
	Storage               *MKSStorage       `json:"storage,omitempty" yaml:"storage,omitempty"`
	Tags                  map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	ControlPlaneNodeGroup *MKSNodeGroup     `json:"controlPlaneNodeGroup,omitempty" yaml:"controlPlaneNodeGroup,omitempty"`
	WorkerNodeGroups      []MKSNodeGroup    `json:"workerNodeGroups,omitempty" yaml:"workerNodeGroups,omitempty"`
	Nodes                 []MKSNodeSpec     `json:"nodes,omitempty" yaml:"nodes,omitempty"`
}

// MKSClusterOutput holds provisioning output reported by the backend.
type MKSClusterOutput struct {
	APIServerEndpoint string `json:"apiServerEndpoint,omitempty" yaml:"apiServerEndpoint,omitempty"`
	ClusterIDEdgesrv  string `json:"clusterIdEdgesrv,omitempty" yaml:"clusterIdEdgesrv,omitempty"`
}

// MKSClusterStatus holds observed MKS cluster state.
type MKSClusterStatus struct {
	Condition       string            `json:"condition,omitempty" yaml:"condition,omitempty"`
	ConditionReason string            `json:"conditionReason,omitempty" yaml:"conditionReason,omitempty"`
	Output          *MKSClusterOutput `json:"output,omitempty" yaml:"output,omitempty"`
	Action          string            `json:"action,omitempty" yaml:"action,omitempty"`
}

// MKSCluster is a managed Kubernetes cluster.
type MKSCluster struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta       `json:"metadata" yaml:"metadata"`
	Spec     MKSClusterSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   MKSClusterStatus `json:"status,omitempty" yaml:"status,omitempty"`
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
	cp.Spec = copyMKSClusterSpec(c.Spec)
	cp.Status = copyMKSClusterStatus(c.Status)
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
// MKSNode
// ---------------------------------------------------------------------------

// MKSNodeStatus holds observed node state.
type MKSNodeStatus struct {
	Phase  string `json:"phase,omitempty" yaml:"phase,omitempty"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// MKSNode is a single node within an MKS cluster.
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
	cp.Spec = copyMKSNodeSpec(n.Spec)
	return &cp
}

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
// MKSWorkerNodeGroup
// ---------------------------------------------------------------------------

// MKSWorkerNodeGroupSpec holds desired worker node group state.
type MKSWorkerNodeGroupSpec struct {
	ClusterName string        `json:"clusterName,omitempty" yaml:"clusterName,omitempty"`
	NodeGroup   *MKSNodeGroup `json:"nodeGroup,omitempty" yaml:"nodeGroup,omitempty"`
}

// MKSWorkerNodeGroup is a worker node group attached to a cluster.
type MKSWorkerNodeGroup struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta             `json:"metadata" yaml:"metadata"`
	Spec     MKSWorkerNodeGroupSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

func (g *MKSWorkerNodeGroup) GetAPIVersion() string { return g.APIVersion }
func (g *MKSWorkerNodeGroup) GetKind() string       { return g.Kind }
func (g *MKSWorkerNodeGroup) GetName() string       { return g.Metadata.Name }
func (g *MKSWorkerNodeGroup) GetProject() string    { return g.Metadata.Project }
func (g *MKSWorkerNodeGroup) GetWorkspace() string  { return g.Metadata.Workspace }
func (g *MKSWorkerNodeGroup) SetProject(v string)   { g.Metadata.Project = v }
func (g *MKSWorkerNodeGroup) SetWorkspace(v string) { g.Metadata.Workspace = v }
func (g *MKSWorkerNodeGroup) DeepCopyObject() runtime.Object {
	cp := *g
	cp.Metadata = copyObjectMeta(g.Metadata)
	if g.Spec.NodeGroup != nil {
		cp.Spec.NodeGroup = copyMKSNodeGroup(g.Spec.NodeGroup)
	}
	return &cp
}

// MKSWorkerNodeGroupList is a list of MKSWorkerNodeGroup resources.
type MKSWorkerNodeGroupList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta             `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []MKSWorkerNodeGroup `json:"items" yaml:"items"`
}

func (l *MKSWorkerNodeGroupList) GetAPIVersion() string { return l.APIVersion }
func (l *MKSWorkerNodeGroupList) GetKind() string       { return l.Kind }
func (l *MKSWorkerNodeGroupList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *MKSWorkerNodeGroupList) SetItems(items []runtime.Object) {
	l.Items = make([]MKSWorkerNodeGroup, len(items))
	for i, item := range items {
		if g, ok := item.(*MKSWorkerNodeGroup); ok {
			l.Items[i] = *g
		}
	}
}

// ---------------------------------------------------------------------------
// MKSAuditEvent (read-only, flat envelope)
// ---------------------------------------------------------------------------

// MKSAuditEvent is a read-only audit record for an MKS cluster.
type MKSAuditEvent struct {
	ID           string     `json:"id,omitempty" yaml:"id,omitempty"`
	ResourceType string     `json:"resourceType,omitempty" yaml:"resourceType,omitempty"`
	ResourceName string     `json:"resourceName,omitempty" yaml:"resourceName,omitempty"`
	Action       string     `json:"action,omitempty" yaml:"action,omitempty"`
	Actor        string     `json:"actor,omitempty" yaml:"actor,omitempty"`
	Message      string     `json:"message,omitempty" yaml:"message,omitempty"`
	Timestamp    int64      `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	Metadata     ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// MKSAuditEventList is a list of MKSAuditEvent records.
type MKSAuditEventList struct {
	Items []MKSAuditEvent `json:"items" yaml:"items"`
}

// ---------------------------------------------------------------------------
// Deep-copy helpers
// ---------------------------------------------------------------------------

func copyBoolPtr(b *bool) *bool {
	if b == nil {
		return nil
	}
	v := *b
	return &v
}

func copyMKSKubeletConfig(k *MKSKubeletConfig) *MKSKubeletConfig {
	if k == nil {
		return nil
	}
	cp := *k
	cp.KeyValue = copyStringMap(k.KeyValue)
	return &cp
}

func copyMKSNodeGroup(g *MKSNodeGroup) *MKSNodeGroup {
	if g == nil {
		return nil
	}
	cp := *g
	cp.PublicIP = copyBoolPtr(g.PublicIP)
	cp.NodeLabels = copyStringMap(g.NodeLabels)
	cp.NodeAnnotations = copyStringMap(g.NodeAnnotations)
	cp.KubeletConfig = copyMKSKubeletConfig(g.KubeletConfig)
	return &cp
}

func copyMKSNodeSpec(s MKSNodeSpec) MKSNodeSpec {
	cp := s
	if len(s.Roles) > 0 {
		cp.Roles = append([]string(nil), s.Roles...)
	}
	cp.NodeLabels = copyStringMap(s.NodeLabels)
	cp.NodeAnnotations = copyStringMap(s.NodeAnnotations)
	cp.NodeTaints = copyStringMap(s.NodeTaints)
	cp.KubeletConfig = copyMKSKubeletConfig(s.KubeletConfig)
	cp.PublicIP = copyBoolPtr(s.PublicIP)
	return cp
}

func copyMKSStorageBackend(b *MKSStorageBackend) *MKSStorageBackend {
	if b == nil {
		return nil
	}
	cp := *b
	cp.Config = copyStringMap(b.Config)
	return &cp
}

func copyMKSClusterSpec(s MKSClusterSpec) MKSClusterSpec {
	cp := s
	cp.HAEnabled = copyBoolPtr(s.HAEnabled)
	cp.DedicatedControlPlane = copyBoolPtr(s.DedicatedControlPlane)
	cp.Tags = copyStringMap(s.Tags)
	if s.Blueprint != nil {
		bp := *s.Blueprint
		cp.Blueprint = &bp
	}
	if s.Networking != nil {
		nw := *s.Networking
		if len(s.Networking.SecurityGroups) > 0 {
			nw.SecurityGroups = append([]string(nil), s.Networking.SecurityGroups...)
		}
		cp.Networking = &nw
	}
	if s.Proxy != nil {
		px := *s.Proxy
		px.TLSTerminate = copyBoolPtr(s.Proxy.TLSTerminate)
		cp.Proxy = &px
	}
	if s.Storage != nil {
		st := *s.Storage
		st.Block = copyMKSStorageBackend(s.Storage.Block)
		st.SharedFS = copyMKSStorageBackend(s.Storage.SharedFS)
		st.Object = copyMKSStorageBackend(s.Storage.Object)
		st.HighSpeed = copyMKSStorageBackend(s.Storage.HighSpeed)
		cp.Storage = &st
	}
	cp.ControlPlaneNodeGroup = copyMKSNodeGroup(s.ControlPlaneNodeGroup)
	if len(s.WorkerNodeGroups) > 0 {
		cp.WorkerNodeGroups = make([]MKSNodeGroup, len(s.WorkerNodeGroups))
		for i := range s.WorkerNodeGroups {
			cp.WorkerNodeGroups[i] = *copyMKSNodeGroup(&s.WorkerNodeGroups[i])
		}
	}
	if len(s.Nodes) > 0 {
		cp.Nodes = make([]MKSNodeSpec, len(s.Nodes))
		for i := range s.Nodes {
			cp.Nodes[i] = copyMKSNodeSpec(s.Nodes[i])
		}
	}
	return cp
}

func copyMKSClusterStatus(s MKSClusterStatus) MKSClusterStatus {
	cp := s
	if s.Output != nil {
		out := *s.Output
		cp.Output = &out
	}
	return cp
}
