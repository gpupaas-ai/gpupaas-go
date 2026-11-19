package convert

// Backend REST paths (not Kubernetes-style /apis/... for all resources).
const (
	PaaSAPIVersion = "paas.envmgmt.io/v1"
	DevAPIVersion  = "dev.envmgmt.io/v1"

	// First-class virtual machines (dev.envmgmt.io): instances have project and workspace scope;
	// the SKU catalog (VmProfile) is project-scoped and read-only. Actions are
	// POST .../{name}/action/{verb}.
	DevProjectVMsPath   = "/apis/dev.envmgmt.io/v1/projects/%s/virtualmachines"
	DevProjectVMPath    = "/apis/dev.envmgmt.io/v1/projects/%s/virtualmachines/%s"
	DevWorkspaceVMsPath = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/virtualmachines"
	DevWorkspaceVMPath  = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/virtualmachines/%s"
	DevVMProfilesPath   = "/apis/dev.envmgmt.io/v1/projects/%s/vmprofiles"
	DevVMProfilePath    = "/apis/dev.envmgmt.io/v1/projects/%s/vmprofiles/%s"

	// First-class bare metal (dev.envmgmt.io): instances have project and workspace scope; the SKU
	// catalog (BmProfile) is project-scoped and read-only. Actions are POST .../{name}/action/{verb}.
	DevProjectBareMetalsPath   = "/apis/dev.envmgmt.io/v1/projects/%s/baremetals"
	DevProjectBareMetalPath    = "/apis/dev.envmgmt.io/v1/projects/%s/baremetals/%s"
	DevWorkspaceBareMetalsPath = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/baremetals"
	DevWorkspaceBareMetalPath  = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/baremetals/%s"
	DevBmProfilesPath          = "/apis/dev.envmgmt.io/v1/projects/%s/bmprofiles"
	DevBmProfilePath           = "/apis/dev.envmgmt.io/v1/projects/%s/bmprofiles/%s"

	// SSH keys and security groups (dev.envmgmt.io), project and workspace scope. Both are created
	// and deleted; the platform has no update for them.
	DevProjectSshKeysPath          = "/apis/dev.envmgmt.io/v1/projects/%s/sshkeys"
	DevProjectSshKeyPath           = "/apis/dev.envmgmt.io/v1/projects/%s/sshkeys/%s"
	DevWorkspaceSshKeysPath        = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/sshkeys"
	DevWorkspaceSshKeyPath         = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/sshkeys/%s"
	DevProjectSecurityGroupsPath   = "/apis/dev.envmgmt.io/v1/projects/%s/securitygroups"
	DevProjectSecurityGroupPath    = "/apis/dev.envmgmt.io/v1/projects/%s/securitygroups/%s"
	DevWorkspaceSecurityGroupsPath = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/securitygroups"
	DevWorkspaceSecurityGroupPath  = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/securitygroups/%s"

	// Storage (dev.envmgmt.io): one kind for high-speed and object storage; the profile named in spec.type
	// decides which. POST on the collection creates or updates by name.
	DevProjectStoragesPath   = "/apis/dev.envmgmt.io/v1/projects/%s/storages"
	DevProjectStoragePath    = "/apis/dev.envmgmt.io/v1/projects/%s/storages/%s"
	DevWorkspaceStoragesPath = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/storages"
	DevWorkspaceStoragePath  = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/storages/%s"

	// Workspaces (paas.envmgmt.io/v1), project-scoped. POST on the collection creates one.
	PaaSWorkspacesPath    = "/apis/paas.envmgmt.io/v1/projects/%s/workspaces"
	PaaSWorkspacePath     = "/apis/paas.envmgmt.io/v1/projects/%s/workspaces/%s"
	PaaSWorkspaceAPIVer   = "paas.envmgmt.io/v1"
	PaaSWorkspaceKind     = "Workspace"
	PaaSWorkspaceListKind = "WorkspaceList"

	// First-class managed Kubernetes (MKS) under the paas.k8smgmt.io/v3 API group. Clusters
	// have project and workspace scope; profiles and nodes are project-scoped.
	MKSAPIVersion = "paas.k8smgmt.io/v3"

	MKSClustersPath          = "/apis/paas.k8smgmt.io/v3/projects/%s/clusters"
	MKSNodePoolPath          = "/apis/paas.k8smgmt.io/v3/projects/%s/nodepools/%s/%s"
	MKSClusterPath           = "/apis/paas.k8smgmt.io/v3/projects/%s/clusters/%s"
	MKSWorkspaceClustersPath = "/apis/paas.k8smgmt.io/v3/projects/%s/workspaces/%s/clusters"
	MKSWorkspaceClusterPath  = "/apis/paas.k8smgmt.io/v3/projects/%s/workspaces/%s/clusters/%s"
	MKSClusterProfilesPath   = "/apis/paas.k8smgmt.io/v3/projects/%s/clusterprofiles"
	MKSClusterProfilePath    = "/apis/paas.k8smgmt.io/v3/projects/%s/clusterprofiles/%s"
	MKSClusterNodesPath      = "/apis/paas.k8smgmt.io/v3/projects/%s/clusternodes"
	// Day-2, status and discovery routes of a cluster, node or Kubernetes SKU, always under the project
	// (cluster names are unique per organization, so a workspace cluster needs no workspace segment).
	MKSClusterSubPath      = "/apis/paas.k8smgmt.io/v3/projects/%s/clusters/%s/%s"
	MKSClusterNodePoolPath = "/apis/paas.k8smgmt.io/v3/projects/%s/clusters/%s/nodepools/%s"
	MKSClusterNodePath     = "/apis/paas.k8smgmt.io/v3/projects/%s/clusternodes/%s"
	MKSClusterNodeVerbPath = "/apis/paas.k8smgmt.io/v3/projects/%s/clusternodes/%s/%s"
	MKSMatchmakerPath      = "/apis/paas.k8smgmt.io/v3/projects/%s/clustermatchmakers/%s"
	MKSClusterProfileSub   = "/apis/paas.k8smgmt.io/v3/projects/%s/clusterprofiles/%s/%s"

	// Compute instances (paas.envmgmt.io/v1), read-only: the instance behind a first-class VM, bare
	// metal or MKS object. Project and workspace scope.
	PaaSComputeInstancesPath          = "/apis/paas.envmgmt.io/v1/projects/%s/computeinstances"
	PaaSComputeInstancePath           = "/apis/paas.envmgmt.io/v1/projects/%s/computeinstances/%s"
	PaaSWorkspaceComputeInstancesPath = "/apis/paas.envmgmt.io/v1/projects/%s/workspaces/%s/computeinstances"
	PaaSWorkspaceComputeInstancePath  = "/apis/paas.envmgmt.io/v1/projects/%s/workspaces/%s/computeinstances/%s"

	// EaaS environment run details used to read provisioning logs of a compute instance.
	EaaSEnvironmentEventPath         = "/apis/eaas.envmgmt.io/v1/projects/%s/environments/%s/events/%s"
	EaaSEnvironmentActivityStdoutFmt = "/apis/eaas.envmgmt.io/v1/projects/%s/environments/%s/events/%s/activity/%s/stdout"
)
