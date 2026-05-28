package convert

// Backend REST paths (not Kubernetes-style /apis/... for all resources).
const (
	AuthProjectsPath                = "/auth/v1/projects/"
	AuthProjectPath                 = "/auth/v1/projects/%s/"
	PaaSWorkspacesPath              = "/apis/paas.envmgmt.io/v1/projects/%s/workspaces"
	PaaSWorkspacePath               = "/apis/paas.envmgmt.io/v1/projects/%s/workspaces/%s"
	PaaSWorkspaceCollaboratorsPath  = "/apis/paas.envmgmt.io/v1/projects/%s/workspaces/%s/collaborators"
	PaaSWorkspaceAssignCollabPath   = "/apis/paas.envmgmt.io/v1/projects/%s/workspaces/%s/assigncollaborators"
	PaaSWorkspaceUnassignCollabPath = "/apis/paas.envmgmt.io/v1/projects/%s/workspaces/%s/unassigncollaborators"
	PaaSWorkspaceAPIVer             = "paas.envmgmt.io/v1"
	PaaSWorkspaceKind               = "Workspace"
	PaaSWorkspaceListKind           = "WorkspaceList"

	DevAPIVersion = "dev.envmgmt.io/v1"

	DevProjectVMsPath      = "/apis/dev.envmgmt.io/v1/projects/%s/virtualmachines"
	DevProjectVMPath       = "/apis/dev.envmgmt.io/v1/projects/%s/virtualmachines/%s"
	DevProjectVMStatusPath = "/apis/dev.envmgmt.io/v1/projects/%s/virtualmachines/%s/status"
	DevProjectVMActionPath = "/apis/dev.envmgmt.io/v1/projects/%s/virtualmachines/%s/action/%s"

	DevWorkspaceVMsPath      = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/virtualmachines"
	DevWorkspaceVMPath       = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/virtualmachines/%s"
	DevWorkspaceVMStatusPath = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/virtualmachines/%s/status"
	DevWorkspaceVMActionPath = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/virtualmachines/%s/action/%s"

	DevProjectStoragesPath   = "/apis/dev.envmgmt.io/v1/projects/%s/storages"
	DevProjectStoragePath    = "/apis/dev.envmgmt.io/v1/projects/%s/storages/%s"
	DevWorkspaceStoragesPath = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/storages"
	DevWorkspaceStoragePath  = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/storages/%s"

	DevProjectSecurityGroupsPath   = "/apis/dev.envmgmt.io/v1/projects/%s/securitygroups"
	DevProjectSecurityGroupPath    = "/apis/dev.envmgmt.io/v1/projects/%s/securitygroups/%s"
	DevWorkspaceSecurityGroupsPath = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/securitygroups"
	DevWorkspaceSecurityGroupPath  = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/securitygroups/%s"

	DevProjectSshKeysPath   = "/apis/dev.envmgmt.io/v1/projects/%s/sshkeys"
	DevProjectSshKeyPath    = "/apis/dev.envmgmt.io/v1/projects/%s/sshkeys/%s"
	DevWorkspaceSshKeysPath = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/sshkeys"
	DevWorkspaceSshKeyPath  = "/apis/dev.envmgmt.io/v1/projects/%s/workspaces/%s/sshkeys/%s"

	// Infra API group for project-scoped baremetal resources.
	InfraAPIVersion = "infra.k8smgmt.io/v3"

	InfraBaremetalMachinesPath              = "/apis/infra.k8smgmt.io/v3/projects/%s/baremetalmachines"
	InfraBaremetalMachinePath               = "/apis/infra.k8smgmt.io/v3/projects/%s/baremetalmachines/%s"
	InfraBaremetalMachinePowerOnPath        = "/apis/infra.k8smgmt.io/v3/projects/%s/baremetalmachines/%s/powerOn"
	InfraBaremetalMachinePowerOffPath       = "/apis/infra.k8smgmt.io/v3/projects/%s/baremetalmachines/%s/powerOff"
	InfraBaremetalMachineRebootPath         = "/apis/infra.k8smgmt.io/v3/projects/%s/baremetalmachines/%s/reboot"
	InfraBaremetalMachineProvisionPath      = "/apis/infra.k8smgmt.io/v3/projects/%s/baremetalmachines/%s/provision"
	InfraBaremetalMachineReinstallOSPath    = "/apis/infra.k8smgmt.io/v3/projects/%s/baremetalmachines/%s/reinstallOS"
	InfraBaremetalMachineConsoleSessionPath = "/apis/infra.k8smgmt.io/v3/projects/%s/baremetalmachines/%s/consoleSessions"
	InfraBaremetalMachineStatusPath         = "/apis/infra.k8smgmt.io/v3/projects/%s/baremetalmachines/%s/status"
)
