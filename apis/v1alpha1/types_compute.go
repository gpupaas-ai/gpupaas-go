package v1alpha1

import (
	"encoding/json"
	"strings"

	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

// KindComputeInstance is the kind of a compute instance on the paas.envmgmt.io API. The SDK reads
// compute instances only: every first-class VM, bare metal and MKS cluster (and node) has one behind
// it, which holds the workflow error and the provisioning log when the first-class record fails.
const KindComputeInstance = "ComputeInstance"

// Normalized compute instance phases. The backend reports lowercase strings with
// optional underscores (for example "not_deployed", "success", "actioncomplete");
// NormalizeComputeStatus maps every spelling to these constants.
const (
	ComputeStatusNotDeployed    = "notdeployed"
	ComputeStatusPending        = "pending"
	ComputeStatusInProgress     = "inprogress"
	ComputeStatusSuccess        = "success"
	ComputeStatusFailed         = "failed"
	ComputeStatusCancelPending  = "cancelpending"
	ComputeStatusCancelling     = "cancelling"
	ComputeStatusCancelled      = "cancelled"
	ComputeStatusCancelFailed   = "cancelfailed"
	ComputeStatusUpdatePending  = "updatepending"
	ComputeStatusUpdating       = "updating"
	ComputeStatusUpdateFailed   = "updatefailed"
	ComputeStatusDestroyPending = "destroypending"
	ComputeStatusDestroying     = "destroying"
	ComputeStatusDestroyFailed  = "destroyfailed"
	ComputeStatusDestroyed      = "destroyed"
	ComputeStatusActionPending  = "actionpending"
	ComputeStatusActionRunning  = "actioninprogress"
	ComputeStatusActionFailed   = "actionfailed"
	ComputeStatusActionComplete = "actioncomplete"
)

// NormalizeComputeStatus lowercases a backend status and strips separators and
// an optional "Status_" prefix, so "not_deployed", "NOT_DEPLOYED" and
// "Status_NOT_DEPLOYED" all become "notdeployed".
func NormalizeComputeStatus(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.NewReplacer("_", "", "-", "", " ", "").Replace(s)
	return strings.TrimPrefix(s, "status")
}

// ComputeEnvironmentRef identifies the EaaS environment that provisions the instance.
type ComputeEnvironmentRef struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	ID   string `json:"id,omitempty" yaml:"id,omitempty"`
}

// ComputeProvisionedResource identifies the provisioning run behind the instance.
type ComputeProvisionedResource struct {
	Provisioner  string `json:"provisioner,omitempty" yaml:"provisioner,omitempty"`
	ResourceID   string `json:"resourceId,omitempty" yaml:"resourceId,omitempty"`
	ResourceName string `json:"resourceName,omitempty" yaml:"resourceName,omitempty"`
	// TriggerID identifies the latest publish run. The backend reports it only
	// after a publish completes (it is absent while running and after actions), so
	// do not use it to detect a new run; use Status.Action and Status.Phase.
	TriggerID string `json:"triggerId,omitempty" yaml:"triggerId,omitempty"`
}

// ComputeInstanceStatus holds observed compute instance state (read-only).
type ComputeInstanceStatus struct {
	// Status is the raw backend status (for example "success", "not_deployed").
	// Use Phase for the normalized value.
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
	// Action is the last action run: publish, start, stop, reboot, destroy, ...
	Action              string                      `json:"action,omitempty" yaml:"action,omitempty"`
	WorkflowStatus      map[string]any              `json:"workflowStatus,omitempty" yaml:"workflowStatus,omitempty"`
	Environment         *ComputeEnvironmentRef      `json:"environment,omitempty" yaml:"environment,omitempty"`
	ProvisionedResource *ComputeProvisionedResource `json:"provisionedResource,omitempty" yaml:"provisionedResource,omitempty"`
}

// Phase returns the normalized status (one of the ComputeStatus* constants).
func (s ComputeInstanceStatus) Phase() string { return NormalizeComputeStatus(s.Status) }

// TriggerID returns the id of the latest provisioning run, or "".
func (s ComputeInstanceStatus) TriggerID() string {
	if s.ProvisionedResource == nil {
		return ""
	}
	return s.ProvisionedResource.TriggerID
}

// InProgress reports whether a provisioning, update, destroy or action run is pending or active.
func (s ComputeInstanceStatus) InProgress() bool {
	switch s.Phase() {
	case ComputeStatusPending, ComputeStatusInProgress, ComputeStatusCancelPending, ComputeStatusCancelling,
		ComputeStatusUpdatePending, ComputeStatusUpdating, ComputeStatusDestroyPending, ComputeStatusDestroying,
		ComputeStatusActionPending, ComputeStatusActionRunning:
		return true
	}
	return false
}

// Failed reports whether the last run failed.
func (s ComputeInstanceStatus) Failed() bool {
	switch s.Phase() {
	case ComputeStatusFailed, ComputeStatusCancelFailed, ComputeStatusUpdateFailed,
		ComputeStatusDestroyFailed, ComputeStatusActionFailed:
		return true
	}
	return false
}

// ComputeFailureLog holds the diagnostic lines of the failed activities of the
// last run of an instance. Only WARNING and ERROR lines are kept, shortened.
type ComputeFailureLog struct {
	Environment string   `json:"environment,omitempty" yaml:"environment,omitempty"`
	Activities  []string `json:"activities,omitempty" yaml:"activities,omitempty"`
	Lines       []string `json:"lines,omitempty" yaml:"lines,omitempty"`
}

// ComputeInstance is the compute resource (VM, baremetal, Kubernetes, ...) behind a first-class
// object. It is project- or workspace-scoped and read-only in this SDK: only its metadata and
// status are modelled.
type ComputeInstance struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta            `json:"metadata" yaml:"metadata"`
	Status   ComputeInstanceStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (c *ComputeInstance) GetAPIVersion() string { return c.APIVersion }
func (c *ComputeInstance) GetKind() string       { return c.Kind }
func (c *ComputeInstance) GetName() string       { return c.Metadata.Name }
func (c *ComputeInstance) GetProject() string    { return c.Metadata.Project }
func (c *ComputeInstance) GetWorkspace() string  { return c.Metadata.Workspace }
func (c *ComputeInstance) SetProject(v string)   { c.Metadata.Project = v }
func (c *ComputeInstance) SetWorkspace(v string) { c.Metadata.Workspace = v }
func (c *ComputeInstance) DeepCopyObject() runtime.Object {
	cp := *c
	cp.Metadata = copyObjectMeta(c.Metadata)
	cp.Status = copyComputeInstanceStatus(c.Status)
	return &cp
}

// ComputeInstanceList is a list of ComputeInstance resources.
type ComputeInstanceList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta          `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []ComputeInstance `json:"items" yaml:"items"`
}

func (l *ComputeInstanceList) GetAPIVersion() string { return l.APIVersion }
func (l *ComputeInstanceList) GetKind() string       { return l.Kind }
func (l *ComputeInstanceList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *ComputeInstanceList) SetItems(items []runtime.Object) {
	l.Items = make([]ComputeInstance, len(items))
	for i, item := range items {
		if c, ok := item.(*ComputeInstance); ok {
			l.Items[i] = *c
		}
	}
}

func copyComputeInstanceStatus(s ComputeInstanceStatus) ComputeInstanceStatus {
	out := s
	out.WorkflowStatus = copyAnyMap(s.WorkflowStatus)
	if s.Environment != nil {
		e := *s.Environment
		out.Environment = &e
	}
	if s.ProvisionedResource != nil {
		p := *s.ProvisionedResource
		out.ProvisionedResource = &p
	}
	return out
}

// copyAnyMap deep-copies a JSON-shaped map.
func copyAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	b, err := json.Marshal(in)
	if err != nil {
		return in
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return in
	}
	return out
}
