package convert

import (
	"fmt"
	"net/url"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	PaaSMKSClusterKind         = "MKSCluster"
	PaaSMKSClusterListKind     = "MKSClusterList"
	PaaSMKSNodeKind            = "MKSNode"
	PaaSMKSNodeListKind        = "MKSNodeList"
	PaaSMKSWorkerNodeGroupKind = "MKSWorkerNodeGroup"
)

// ---------------------------------------------------------------------------
// Scopes & path builders
// ---------------------------------------------------------------------------

// PaaSMKSScope identifies a project on paas.envmgmt.io for MKS clusters.
type PaaSMKSScope struct {
	Project string
}

// PaaSMKSClusterScope identifies a cluster within a project for MKS
// sub-resources (nodes, worker node groups, audit events).
type PaaSMKSClusterScope struct {
	Project string
	Cluster string
}

// MKSClusterPaths returns path builders for MKS cluster endpoints.
func MKSClusterPaths(scope PaaSMKSScope) (
	collection func() string,
	item func(name string) string,
	subroute func(name, action string) string,
) {
	collection = func() string {
		return fmt.Sprintf(PaaSMKSClustersPath, url.PathEscape(scope.Project))
	}
	item = func(name string) string {
		return fmt.Sprintf(PaaSMKSClusterPath, url.PathEscape(scope.Project), url.PathEscape(name))
	}
	subroute = func(name, action string) string {
		base := fmt.Sprintf(PaaSMKSClusterPath, url.PathEscape(scope.Project), url.PathEscape(name))
		return base + "/" + action
	}
	return
}

// MKSNodePaths returns path builders for MKS node endpoints within a cluster.
func MKSNodePaths(scope PaaSMKSClusterScope) (
	collection func() string,
	item func(name string) string,
	subroute func(name, action string) string,
) {
	collection = func() string {
		return fmt.Sprintf(PaaSMKSNodesPath, url.PathEscape(scope.Project), url.PathEscape(scope.Cluster))
	}
	item = func(name string) string {
		return fmt.Sprintf(PaaSMKSNodePath, url.PathEscape(scope.Project), url.PathEscape(scope.Cluster), url.PathEscape(name))
	}
	subroute = func(name, action string) string {
		base := fmt.Sprintf(PaaSMKSNodePath, url.PathEscape(scope.Project), url.PathEscape(scope.Cluster), url.PathEscape(name))
		return base + "/" + action
	}
	return
}

// MKSWorkerNodeGroupPaths returns path builders for worker node group endpoints.
func MKSWorkerNodeGroupPaths(scope PaaSMKSClusterScope) (
	collection func() string,
	item func(name string) string,
) {
	collection = func() string {
		return fmt.Sprintf(PaaSMKSWorkerNodeGroupsPath, url.PathEscape(scope.Project), url.PathEscape(scope.Cluster))
	}
	item = func(name string) string {
		return fmt.Sprintf(PaaSMKSWorkerNodeGroupPath, url.PathEscape(scope.Project), url.PathEscape(scope.Cluster), url.PathEscape(name))
	}
	return
}

// MKSAuditEventPaths returns path builders for audit event endpoints.
func MKSAuditEventPaths(scope PaaSMKSClusterScope) (
	collection func() string,
	item func(id string) string,
) {
	collection = func() string {
		return fmt.Sprintf(PaaSMKSAuditEventsPath, url.PathEscape(scope.Project), url.PathEscape(scope.Cluster))
	}
	item = func(id string) string {
		return fmt.Sprintf(PaaSMKSAuditEventPath, url.PathEscape(scope.Project), url.PathEscape(scope.Cluster), url.PathEscape(id))
	}
	return
}

// ---------------------------------------------------------------------------
// Wire metadata
// ---------------------------------------------------------------------------

// PaaSMKSMetadata is resource metadata on the MKS paas.envmgmt.io API.
// id / projectID / createdAt / modifiedAt are populated by the backend on reads.
type PaaSMKSMetadata struct {
	Name        string            `json:"name,omitempty"`
	Project     string            `json:"project,omitempty"`
	ID          string            `json:"id,omitempty"`
	ProjectID   string            `json:"projectID,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"createdAt,omitempty"`
	ModifiedAt  string            `json:"modifiedAt,omitempty"`
}

func paasMKSMetadataToWire(meta apiv1.ObjectMeta, project string) PaaSMKSMetadata {
	projectName := meta.Project
	if projectName == "" {
		projectName = project
	}
	return PaaSMKSMetadata{
		Name:        meta.Name,
		Project:     projectName,
		Labels:      copyStringMap(meta.Labels),
		Annotations: copyStringMap(meta.Annotations),
	}
}

func paasMKSMetadataFromWire(w PaaSMKSMetadata) apiv1.ObjectMeta {
	return apiv1.ObjectMeta{
		Name:        w.Name,
		Project:     w.Project,
		Labels:      copyStringMap(w.Labels),
		Annotations: copyStringMap(w.Annotations),
	}
}

// ---------------------------------------------------------------------------
// Wire nested types
// ---------------------------------------------------------------------------

type PaaSMKSBlueprint struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type PaaSMKSNetworking struct {
	VPC            string   `json:"vpc,omitempty"`
	Subnet         string   `json:"subnet,omitempty"`
	PodCIDR        string   `json:"podCidr,omitempty"`
	ServiceCIDR    string   `json:"serviceCidr,omitempty"`
	IPFamily       string   `json:"ipFamily,omitempty"`
	PodCIDRV6      string   `json:"podCidrV6,omitempty"`
	ServiceCIDRV6  string   `json:"serviceCidrV6,omitempty"`
	SecurityGroups []string `json:"securityGroups,omitempty"`
}

type PaaSMKSProxy struct {
	HTTPProxy    string `json:"httpProxy,omitempty"`
	HTTPSProxy   string `json:"httpsProxy,omitempty"`
	NoProxy      string `json:"noProxy,omitempty"`
	ProxyRootCA  string `json:"proxyRootCa,omitempty"`
	TLSTerminate *bool  `json:"tlsTerminate,omitempty"`
}

type PaaSMKSStorageBackend struct {
	Type          string            `json:"type,omitempty"`
	AccessMode    string            `json:"accessMode,omitempty"`
	ReclaimPolicy string            `json:"reclaimPolicy,omitempty"`
	Config        map[string]string `json:"config,omitempty"`
}

type PaaSMKSStorage struct {
	Block               *PaaSMKSStorageBackend `json:"block,omitempty"`
	SharedFS            *PaaSMKSStorageBackend `json:"sharedFs,omitempty"`
	Object              *PaaSMKSStorageBackend `json:"object,omitempty"`
	HighSpeed           *PaaSMKSStorageBackend `json:"highSpeed,omitempty"`
	DefaultStorageClass string                 `json:"defaultStorageClass,omitempty"`
}

type PaaSMKSKubeletConfig struct {
	KeyValue map[string]string `json:"keyValue,omitempty"`
	YAML     string            `json:"yaml,omitempty"`
}

type PaaSMKSNodeGroup struct {
	ID              string                `json:"id,omitempty"`
	SKU             string                `json:"sku,omitempty"`
	ScalingMode     string                `json:"scalingMode,omitempty"`
	NodeCount       int32                 `json:"nodeCount,omitempty"`
	MinNodes        int32                 `json:"minNodes,omitempty"`
	MaxNodes        int32                 `json:"maxNodes,omitempty"`
	DesiredNodes    int32                 `json:"desiredNodes,omitempty"`
	PublicIP        *bool                 `json:"publicIp,omitempty"`
	SSHKey          string                `json:"sshKey,omitempty"`
	UserData        string                `json:"userData,omitempty"`
	NodeLabels      map[string]string     `json:"nodeLabels,omitempty"`
	NodeAnnotations map[string]string     `json:"nodeAnnotations,omitempty"`
	KubeletConfig   *PaaSMKSKubeletConfig `json:"kubeletConfig,omitempty"`
}

type PaaSMKSNodeSpec struct {
	Hostname        string                `json:"hostname,omitempty"`
	InventorySource string                `json:"inventorySource,omitempty"`
	Roles           []string              `json:"roles,omitempty"`
	SSHKey          string                `json:"sshKey,omitempty"`
	SSHUserName     string                `json:"sshUserName,omitempty"`
	SSHPort         int32                 `json:"sshPort,omitempty"`
	PrivateIP       string                `json:"privateIp,omitempty"`
	Arch            string                `json:"arch,omitempty"`
	OperatingSystem string                `json:"operatingSystem,omitempty"`
	Interface       string                `json:"interface,omitempty"`
	NodeLabels      map[string]string     `json:"nodeLabels,omitempty"`
	NodeAnnotations map[string]string     `json:"nodeAnnotations,omitempty"`
	NodeTaints      map[string]string     `json:"nodeTaints,omitempty"`
	UserData        string                `json:"userData,omitempty"`
	KubeletConfig   *PaaSMKSKubeletConfig `json:"kubeletConfig,omitempty"`
	NodePool        string                `json:"nodePool,omitempty"`
	SKU             string                `json:"sku,omitempty"`
	PublicIP        *bool                 `json:"publicIp,omitempty"`
}

// ---------------------------------------------------------------------------
// Wire cluster types
// ---------------------------------------------------------------------------

type PaaSMKSClusterSpec struct {
	KubernetesVersion     string             `json:"kubernetesVersion,omitempty"`
	PlatformVersion       string             `json:"platformVersion,omitempty"`
	CNI                   string             `json:"cni,omitempty"`
	CNIVersion            string             `json:"cniVersion,omitempty"`
	OS                    string             `json:"os,omitempty"`
	HAEnabled             *bool              `json:"haEnabled,omitempty"`
	DedicatedControlPlane *bool              `json:"dedicatedControlPlane,omitempty"`
	Location              string             `json:"location,omitempty"`
	Blueprint             *PaaSMKSBlueprint  `json:"blueprint,omitempty"`
	Networking            *PaaSMKSNetworking `json:"networking,omitempty"`
	Proxy                 *PaaSMKSProxy      `json:"proxy,omitempty"`
	Storage               *PaaSMKSStorage    `json:"storage,omitempty"`
	Tags                  map[string]string  `json:"tags,omitempty"`
	ControlPlaneNodeGroup *PaaSMKSNodeGroup  `json:"controlPlaneNodeGroup,omitempty"`
	WorkerNodeGroups      []PaaSMKSNodeGroup `json:"workerNodeGroups,omitempty"`
	Nodes                 []PaaSMKSNodeSpec  `json:"nodes,omitempty"`
}

type PaaSMKSClusterOutput struct {
	APIServerEndpoint string `json:"apiServerEndpoint,omitempty"`
	ClusterIDEdgesrv  string `json:"clusterIdEdgesrv,omitempty"`
}

type PaaSMKSClusterStatus struct {
	Condition       string                `json:"condition,omitempty"`
	ConditionReason string                `json:"conditionReason,omitempty"`
	Output          *PaaSMKSClusterOutput `json:"output,omitempty"`
	Action          string                `json:"action,omitempty"`
}

type PaaSMKSCluster struct {
	APIVersion string               `json:"apiVersion"`
	Kind       string               `json:"kind"`
	Metadata   PaaSMKSMetadata      `json:"metadata"`
	Spec       PaaSMKSClusterSpec   `json:"spec,omitempty"`
	Status     PaaSMKSClusterStatus `json:"status,omitempty"`
}

type PaaSMKSClusterList struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   PaaSListMetadata `json:"metadata,omitempty"`
	Items      []PaaSMKSCluster `json:"items"`
}

// ---------------------------------------------------------------------------
// Wire node / worker node group / audit types
// ---------------------------------------------------------------------------

type PaaSMKSNodeStatus struct {
	Phase  string `json:"phase,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type PaaSMKSNode struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   PaaSMKSMetadata   `json:"metadata"`
	Spec       PaaSMKSNodeSpec   `json:"spec,omitempty"`
	Status     PaaSMKSNodeStatus `json:"status,omitempty"`
}

type PaaSMKSNodeList struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   PaaSListMetadata `json:"metadata,omitempty"`
	Items      []PaaSMKSNode    `json:"items"`
}

type PaaSMKSWorkerNodeGroupSpec struct {
	ClusterName string            `json:"clusterName,omitempty"`
	NodeGroup   *PaaSMKSNodeGroup `json:"nodeGroup,omitempty"`
}

type PaaSMKSWorkerNodeGroup struct {
	APIVersion string                     `json:"apiVersion"`
	Kind       string                     `json:"kind"`
	Metadata   PaaSMKSMetadata            `json:"metadata"`
	Spec       PaaSMKSWorkerNodeGroupSpec `json:"spec,omitempty"`
}

type PaaSMKSWorkerNodeGroupList struct {
	Items []PaaSMKSWorkerNodeGroup `json:"items"`
}

type PaaSMKSAuditEvent struct {
	ID           string          `json:"id,omitempty"`
	ResourceType string          `json:"resourceType,omitempty"`
	ResourceName string          `json:"resourceName,omitempty"`
	Action       string          `json:"action,omitempty"`
	Actor        string          `json:"actor,omitempty"`
	Message      string          `json:"message,omitempty"`
	Timestamp    int64           `json:"timestamp,omitempty"`
	Metadata     PaaSMKSMetadata `json:"metadata,omitempty"`
}

type PaaSMKSAuditEventList struct {
	Items []PaaSMKSAuditEvent `json:"items"`
}

// ---------------------------------------------------------------------------
// Sub-action request wire bodies
// ---------------------------------------------------------------------------

type PaaSMKSUpgradeRequest struct {
	K8sVersion      string `json:"k8sVersion,omitempty"`
	PlatformVersion string `json:"platformVersion,omitempty"`
}

type PaaSMKSScaleNodeGroupRequest struct {
	NodeGroupName string `json:"nodeGroupName"`
	DesiredCount  *int32 `json:"desiredCount,omitempty"`
	MinCount      *int32 `json:"minCount,omitempty"`
	MaxCount      *int32 `json:"maxCount,omitempty"`
}

type PaaSMKSAddNodeGroupRequest struct {
	NodeGroup *PaaSMKSNodeGroup `json:"nodeGroup"`
}

type PaaSMKSRemoveNodeGroupRequest struct {
	NodeGroupName string `json:"nodeGroupName"`
}

type PaaSMKSDrainRequest struct {
	Force              *bool  `json:"force,omitempty"`
	IgnoreDaemonsets   *bool  `json:"ignoreDaemonsets,omitempty"`
	GracePeriodSeconds *int32 `json:"gracePeriodSeconds,omitempty"`
}

// ---------------------------------------------------------------------------
// Shared nested converters
// ---------------------------------------------------------------------------

func boolPtr(b *bool) *bool {
	if b == nil {
		return nil
	}
	v := *b
	return &v
}

func int32Ptr(i *int32) *int32 {
	if i == nil {
		return nil
	}
	v := *i
	return &v
}

func toPaaSMKSKubeletConfig(k *apiv1.MKSKubeletConfig) *PaaSMKSKubeletConfig {
	if k == nil {
		return nil
	}
	return &PaaSMKSKubeletConfig{
		KeyValue: copyStringMap(k.KeyValue),
		YAML:     k.YAML,
	}
}

func fromPaaSMKSKubeletConfig(k *PaaSMKSKubeletConfig) *apiv1.MKSKubeletConfig {
	if k == nil {
		return nil
	}
	return &apiv1.MKSKubeletConfig{
		KeyValue: copyStringMap(k.KeyValue),
		YAML:     k.YAML,
	}
}

func toPaaSMKSNodeGroup(g *apiv1.MKSNodeGroup) *PaaSMKSNodeGroup {
	if g == nil {
		return nil
	}
	out := &PaaSMKSNodeGroup{
		ID:              g.ID,
		SKU:             g.SKU,
		ScalingMode:     g.ScalingMode,
		NodeCount:       g.NodeCount,
		MinNodes:        g.MinNodes,
		MaxNodes:        g.MaxNodes,
		DesiredNodes:    g.DesiredNodes,
		PublicIP:        boolPtr(g.PublicIP),
		SSHKey:          g.SSHKey,
		UserData:        g.UserData,
		NodeLabels:      copyStringMap(g.NodeLabels),
		NodeAnnotations: copyStringMap(g.NodeAnnotations),
		KubeletConfig:   toPaaSMKSKubeletConfig(g.KubeletConfig),
	}
	return out
}

func fromPaaSMKSNodeGroup(g *PaaSMKSNodeGroup) *apiv1.MKSNodeGroup {
	if g == nil {
		return nil
	}
	return &apiv1.MKSNodeGroup{
		ID:              g.ID,
		SKU:             g.SKU,
		ScalingMode:     g.ScalingMode,
		NodeCount:       g.NodeCount,
		MinNodes:        g.MinNodes,
		MaxNodes:        g.MaxNodes,
		DesiredNodes:    g.DesiredNodes,
		PublicIP:        boolPtr(g.PublicIP),
		SSHKey:          g.SSHKey,
		UserData:        g.UserData,
		NodeLabels:      copyStringMap(g.NodeLabels),
		NodeAnnotations: copyStringMap(g.NodeAnnotations),
		KubeletConfig:   fromPaaSMKSKubeletConfig(g.KubeletConfig),
	}
}

func toPaaSMKSNodeSpec(s apiv1.MKSNodeSpec) PaaSMKSNodeSpec {
	out := PaaSMKSNodeSpec{
		Hostname:        s.Hostname,
		InventorySource: s.InventorySource,
		SSHKey:          s.SSHKey,
		SSHUserName:     s.SSHUserName,
		SSHPort:         s.SSHPort,
		PrivateIP:       s.PrivateIP,
		Arch:            s.Arch,
		OperatingSystem: s.OperatingSystem,
		Interface:       s.Interface,
		NodeLabels:      copyStringMap(s.NodeLabels),
		NodeAnnotations: copyStringMap(s.NodeAnnotations),
		NodeTaints:      copyStringMap(s.NodeTaints),
		UserData:        s.UserData,
		KubeletConfig:   toPaaSMKSKubeletConfig(s.KubeletConfig),
		NodePool:        s.NodePool,
		SKU:             s.SKU,
		PublicIP:        boolPtr(s.PublicIP),
	}
	if len(s.Roles) > 0 {
		out.Roles = append([]string(nil), s.Roles...)
	}
	return out
}

func fromPaaSMKSNodeSpec(s PaaSMKSNodeSpec) apiv1.MKSNodeSpec {
	out := apiv1.MKSNodeSpec{
		Hostname:        s.Hostname,
		InventorySource: s.InventorySource,
		SSHKey:          s.SSHKey,
		SSHUserName:     s.SSHUserName,
		SSHPort:         s.SSHPort,
		PrivateIP:       s.PrivateIP,
		Arch:            s.Arch,
		OperatingSystem: s.OperatingSystem,
		Interface:       s.Interface,
		NodeLabels:      copyStringMap(s.NodeLabels),
		NodeAnnotations: copyStringMap(s.NodeAnnotations),
		NodeTaints:      copyStringMap(s.NodeTaints),
		UserData:        s.UserData,
		KubeletConfig:   fromPaaSMKSKubeletConfig(s.KubeletConfig),
		NodePool:        s.NodePool,
		SKU:             s.SKU,
		PublicIP:        boolPtr(s.PublicIP),
	}
	if len(s.Roles) > 0 {
		out.Roles = append([]string(nil), s.Roles...)
	}
	return out
}

func toPaaSMKSStorageBackend(b *apiv1.MKSStorageBackend) *PaaSMKSStorageBackend {
	if b == nil {
		return nil
	}
	return &PaaSMKSStorageBackend{
		Type:          b.Type,
		AccessMode:    b.AccessMode,
		ReclaimPolicy: b.ReclaimPolicy,
		Config:        copyStringMap(b.Config),
	}
}

func fromPaaSMKSStorageBackend(b *PaaSMKSStorageBackend) *apiv1.MKSStorageBackend {
	if b == nil {
		return nil
	}
	return &apiv1.MKSStorageBackend{
		Type:          b.Type,
		AccessMode:    b.AccessMode,
		ReclaimPolicy: b.ReclaimPolicy,
		Config:        copyStringMap(b.Config),
	}
}

// ---------------------------------------------------------------------------
// Cluster converters
// ---------------------------------------------------------------------------

// ToPaaSMKSCluster converts a k8s-style MKSCluster to the paas.envmgmt.io wire format.
func ToPaaSMKSCluster(c *apiv1.MKSCluster, project string) *PaaSMKSCluster {
	if c == nil {
		return nil
	}
	return &PaaSMKSCluster{
		APIVersion: PaaSMKSAPIVersion,
		Kind:       PaaSMKSClusterKind,
		Metadata:   paasMKSMetadataToWire(c.Metadata, project),
		Spec:       toPaaSMKSClusterSpec(c.Spec),
	}
}

func toPaaSMKSClusterSpec(s apiv1.MKSClusterSpec) PaaSMKSClusterSpec {
	out := PaaSMKSClusterSpec{
		KubernetesVersion:     s.KubernetesVersion,
		PlatformVersion:       s.PlatformVersion,
		CNI:                   s.CNI,
		CNIVersion:            s.CNIVersion,
		OS:                    s.OS,
		HAEnabled:             boolPtr(s.HAEnabled),
		DedicatedControlPlane: boolPtr(s.DedicatedControlPlane),
		Location:              s.Location,
		Tags:                  copyStringMap(s.Tags),
		ControlPlaneNodeGroup: toPaaSMKSNodeGroup(s.ControlPlaneNodeGroup),
	}
	if s.Blueprint != nil {
		out.Blueprint = &PaaSMKSBlueprint{Name: s.Blueprint.Name, Version: s.Blueprint.Version}
	}
	if s.Networking != nil {
		nw := &PaaSMKSNetworking{
			VPC:           s.Networking.VPC,
			Subnet:        s.Networking.Subnet,
			PodCIDR:       s.Networking.PodCIDR,
			ServiceCIDR:   s.Networking.ServiceCIDR,
			IPFamily:      s.Networking.IPFamily,
			PodCIDRV6:     s.Networking.PodCIDRV6,
			ServiceCIDRV6: s.Networking.ServiceCIDRV6,
		}
		if len(s.Networking.SecurityGroups) > 0 {
			nw.SecurityGroups = append([]string(nil), s.Networking.SecurityGroups...)
		}
		out.Networking = nw
	}
	if s.Proxy != nil {
		out.Proxy = &PaaSMKSProxy{
			HTTPProxy:    s.Proxy.HTTPProxy,
			HTTPSProxy:   s.Proxy.HTTPSProxy,
			NoProxy:      s.Proxy.NoProxy,
			ProxyRootCA:  s.Proxy.ProxyRootCA,
			TLSTerminate: boolPtr(s.Proxy.TLSTerminate),
		}
	}
	if s.Storage != nil {
		out.Storage = &PaaSMKSStorage{
			Block:               toPaaSMKSStorageBackend(s.Storage.Block),
			SharedFS:            toPaaSMKSStorageBackend(s.Storage.SharedFS),
			Object:              toPaaSMKSStorageBackend(s.Storage.Object),
			HighSpeed:           toPaaSMKSStorageBackend(s.Storage.HighSpeed),
			DefaultStorageClass: s.Storage.DefaultStorageClass,
		}
	}
	if len(s.WorkerNodeGroups) > 0 {
		out.WorkerNodeGroups = make([]PaaSMKSNodeGroup, len(s.WorkerNodeGroups))
		for i := range s.WorkerNodeGroups {
			out.WorkerNodeGroups[i] = *toPaaSMKSNodeGroup(&s.WorkerNodeGroups[i])
		}
	}
	if len(s.Nodes) > 0 {
		out.Nodes = make([]PaaSMKSNodeSpec, len(s.Nodes))
		for i := range s.Nodes {
			out.Nodes[i] = toPaaSMKSNodeSpec(s.Nodes[i])
		}
	}
	return out
}

// FromPaaSMKSCluster converts a wire MKSCluster to the k8s-style type.
func FromPaaSMKSCluster(c *PaaSMKSCluster) *apiv1.MKSCluster {
	if c == nil {
		return nil
	}
	return &apiv1.MKSCluster{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSCluster},
		Metadata: paasMKSMetadataFromWire(c.Metadata),
		Spec:     fromPaaSMKSClusterSpec(c.Spec),
		Status:   fromPaaSMKSClusterStatus(c.Status),
	}
}

func fromPaaSMKSClusterSpec(s PaaSMKSClusterSpec) apiv1.MKSClusterSpec {
	out := apiv1.MKSClusterSpec{
		KubernetesVersion:     s.KubernetesVersion,
		PlatformVersion:       s.PlatformVersion,
		CNI:                   s.CNI,
		CNIVersion:            s.CNIVersion,
		OS:                    s.OS,
		HAEnabled:             boolPtr(s.HAEnabled),
		DedicatedControlPlane: boolPtr(s.DedicatedControlPlane),
		Location:              s.Location,
		Tags:                  copyStringMap(s.Tags),
		ControlPlaneNodeGroup: fromPaaSMKSNodeGroup(s.ControlPlaneNodeGroup),
	}
	if s.Blueprint != nil {
		out.Blueprint = &apiv1.MKSBlueprint{Name: s.Blueprint.Name, Version: s.Blueprint.Version}
	}
	if s.Networking != nil {
		nw := &apiv1.MKSNetworking{
			VPC:           s.Networking.VPC,
			Subnet:        s.Networking.Subnet,
			PodCIDR:       s.Networking.PodCIDR,
			ServiceCIDR:   s.Networking.ServiceCIDR,
			IPFamily:      s.Networking.IPFamily,
			PodCIDRV6:     s.Networking.PodCIDRV6,
			ServiceCIDRV6: s.Networking.ServiceCIDRV6,
		}
		if len(s.Networking.SecurityGroups) > 0 {
			nw.SecurityGroups = append([]string(nil), s.Networking.SecurityGroups...)
		}
		out.Networking = nw
	}
	if s.Proxy != nil {
		out.Proxy = &apiv1.MKSProxy{
			HTTPProxy:    s.Proxy.HTTPProxy,
			HTTPSProxy:   s.Proxy.HTTPSProxy,
			NoProxy:      s.Proxy.NoProxy,
			ProxyRootCA:  s.Proxy.ProxyRootCA,
			TLSTerminate: boolPtr(s.Proxy.TLSTerminate),
		}
	}
	if s.Storage != nil {
		out.Storage = &apiv1.MKSStorage{
			Block:               fromPaaSMKSStorageBackend(s.Storage.Block),
			SharedFS:            fromPaaSMKSStorageBackend(s.Storage.SharedFS),
			Object:              fromPaaSMKSStorageBackend(s.Storage.Object),
			HighSpeed:           fromPaaSMKSStorageBackend(s.Storage.HighSpeed),
			DefaultStorageClass: s.Storage.DefaultStorageClass,
		}
	}
	if len(s.WorkerNodeGroups) > 0 {
		out.WorkerNodeGroups = make([]apiv1.MKSNodeGroup, len(s.WorkerNodeGroups))
		for i := range s.WorkerNodeGroups {
			out.WorkerNodeGroups[i] = *fromPaaSMKSNodeGroup(&s.WorkerNodeGroups[i])
		}
	}
	if len(s.Nodes) > 0 {
		out.Nodes = make([]apiv1.MKSNodeSpec, len(s.Nodes))
		for i := range s.Nodes {
			out.Nodes[i] = fromPaaSMKSNodeSpec(s.Nodes[i])
		}
	}
	return out
}

func fromPaaSMKSClusterStatus(s PaaSMKSClusterStatus) apiv1.MKSClusterStatus {
	out := apiv1.MKSClusterStatus{
		Condition:       s.Condition,
		ConditionReason: s.ConditionReason,
		Action:          s.Action,
	}
	if s.Output != nil {
		out.Output = &apiv1.MKSClusterOutput{
			APIServerEndpoint: s.Output.APIServerEndpoint,
			ClusterIDEdgesrv:  s.Output.ClusterIDEdgesrv,
		}
	}
	return out
}

// FromPaaSMKSClusterList converts a wire cluster list to the SDK list shape.
func FromPaaSMKSClusterList(wire *PaaSMKSClusterList) *apiv1.MKSClusterList {
	out := &apiv1.MKSClusterList{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSCluster + "List"},
	}
	if wire == nil {
		return out
	}
	out.Metadata.Continue = infraListContinue(wire.Metadata, len(wire.Items))
	for i := range wire.Items {
		if c := FromPaaSMKSCluster(&wire.Items[i]); c != nil {
			out.Items = append(out.Items, *c)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Node converters
// ---------------------------------------------------------------------------

// FromPaaSMKSNode converts a wire MKSNode to the k8s-style type.
func FromPaaSMKSNode(n *PaaSMKSNode) *apiv1.MKSNode {
	if n == nil {
		return nil
	}
	return &apiv1.MKSNode{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSNode},
		Metadata: paasMKSMetadataFromWire(n.Metadata),
		Spec:     fromPaaSMKSNodeSpec(n.Spec),
		Status: apiv1.MKSNodeStatus{
			Phase:  n.Status.Phase,
			Reason: n.Status.Reason,
		},
	}
}

// FromPaaSMKSNodeList converts a wire node list to the SDK list shape.
func FromPaaSMKSNodeList(wire *PaaSMKSNodeList) *apiv1.MKSNodeList {
	out := &apiv1.MKSNodeList{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSNode + "List"},
	}
	if wire == nil {
		return out
	}
	out.Metadata.Continue = infraListContinue(wire.Metadata, len(wire.Items))
	for i := range wire.Items {
		if n := FromPaaSMKSNode(&wire.Items[i]); n != nil {
			out.Items = append(out.Items, *n)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Worker node group converters
// ---------------------------------------------------------------------------

// ToPaaSMKSWorkerNodeGroup converts a k8s-style MKSWorkerNodeGroup to the wire format.
func ToPaaSMKSWorkerNodeGroup(g *apiv1.MKSWorkerNodeGroup, project string) *PaaSMKSWorkerNodeGroup {
	if g == nil {
		return nil
	}
	return &PaaSMKSWorkerNodeGroup{
		APIVersion: PaaSMKSAPIVersion,
		Kind:       PaaSMKSWorkerNodeGroupKind,
		Metadata:   paasMKSMetadataToWire(g.Metadata, project),
		Spec: PaaSMKSWorkerNodeGroupSpec{
			ClusterName: g.Spec.ClusterName,
			NodeGroup:   toPaaSMKSNodeGroup(g.Spec.NodeGroup),
		},
	}
}

// FromPaaSMKSWorkerNodeGroup converts a wire MKSWorkerNodeGroup to the k8s-style type.
func FromPaaSMKSWorkerNodeGroup(g *PaaSMKSWorkerNodeGroup) *apiv1.MKSWorkerNodeGroup {
	if g == nil {
		return nil
	}
	return &apiv1.MKSWorkerNodeGroup{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSWorkerNodeGroup},
		Metadata: paasMKSMetadataFromWire(g.Metadata),
		Spec: apiv1.MKSWorkerNodeGroupSpec{
			ClusterName: g.Spec.ClusterName,
			NodeGroup:   fromPaaSMKSNodeGroup(g.Spec.NodeGroup),
		},
	}
}

// FromPaaSMKSWorkerNodeGroupList converts a wire worker node group list to the SDK list shape.
func FromPaaSMKSWorkerNodeGroupList(wire *PaaSMKSWorkerNodeGroupList) *apiv1.MKSWorkerNodeGroupList {
	out := &apiv1.MKSWorkerNodeGroupList{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSWorkerNodeGroup + "List"},
	}
	if wire == nil {
		return out
	}
	for i := range wire.Items {
		if g := FromPaaSMKSWorkerNodeGroup(&wire.Items[i]); g != nil {
			out.Items = append(out.Items, *g)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Audit event converters
// ---------------------------------------------------------------------------

// FromPaaSMKSAuditEvent converts a wire audit event to the SDK type.
func FromPaaSMKSAuditEvent(e *PaaSMKSAuditEvent) *apiv1.MKSAuditEvent {
	if e == nil {
		return nil
	}
	return &apiv1.MKSAuditEvent{
		ID:           e.ID,
		ResourceType: e.ResourceType,
		ResourceName: e.ResourceName,
		Action:       e.Action,
		Actor:        e.Actor,
		Message:      e.Message,
		Timestamp:    e.Timestamp,
		Metadata:     paasMKSMetadataFromWire(e.Metadata),
	}
}

// FromPaaSMKSAuditEventList converts a wire audit event list to the SDK list shape.
func FromPaaSMKSAuditEventList(wire *PaaSMKSAuditEventList) *apiv1.MKSAuditEventList {
	out := &apiv1.MKSAuditEventList{}
	if wire == nil {
		return out
	}
	for i := range wire.Items {
		if e := FromPaaSMKSAuditEvent(&wire.Items[i]); e != nil {
			out.Items = append(out.Items, *e)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Sub-action request converters
// ---------------------------------------------------------------------------

// ToPaaSMKSUpgradeRequest converts the SDK upgrade request to the wire format.
func ToPaaSMKSUpgradeRequest(req *apiv1.MKSUpgradeRequest) *PaaSMKSUpgradeRequest {
	if req == nil {
		return nil
	}
	return &PaaSMKSUpgradeRequest{
		K8sVersion:      req.K8sVersion,
		PlatformVersion: req.PlatformVersion,
	}
}

// ToPaaSMKSScaleNodeGroupRequest converts the SDK scale request to the wire format.
func ToPaaSMKSScaleNodeGroupRequest(req *apiv1.MKSScaleNodeGroupRequest) *PaaSMKSScaleNodeGroupRequest {
	if req == nil {
		return nil
	}
	return &PaaSMKSScaleNodeGroupRequest{
		NodeGroupName: req.NodeGroupName,
		DesiredCount:  int32Ptr(req.DesiredCount),
		MinCount:      int32Ptr(req.MinCount),
		MaxCount:      int32Ptr(req.MaxCount),
	}
}

// ToPaaSMKSAddNodeGroupRequest converts an SDK node group to the addNodeGroup wire body.
func ToPaaSMKSAddNodeGroupRequest(ng *apiv1.MKSNodeGroup) *PaaSMKSAddNodeGroupRequest {
	if ng == nil {
		return nil
	}
	return &PaaSMKSAddNodeGroupRequest{NodeGroup: toPaaSMKSNodeGroup(ng)}
}

// ToPaaSMKSDrainRequest converts the SDK drain request to the wire format.
func ToPaaSMKSDrainRequest(req *apiv1.MKSDrainRequest) *PaaSMKSDrainRequest {
	if req == nil {
		return nil
	}
	return &PaaSMKSDrainRequest{
		Force:              boolPtr(req.Force),
		IgnoreDaemonsets:   boolPtr(req.IgnoreDaemonsets),
		GracePeriodSeconds: int32Ptr(req.GracePeriodSeconds),
	}
}
