package convert_test

import (
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func TestRoleForSpec(t *testing.T) {
	if got := convert.RoleForSpec(apiv1.WorkspaceRoleCollaborator); got != apiv1.WorkspaceRoleCollaborator {
		t.Fatalf("got %q want %q", got, apiv1.WorkspaceRoleCollaborator)
	}
}

func TestValidWorkspaceCollaboratorRole(t *testing.T) {
	if !apiv1.ValidWorkspaceCollaboratorRole(apiv1.WorkspaceRoleCollaboratorReadOnly) {
		t.Fatal("expected read-only role to be valid")
	}
	if apiv1.ValidWorkspaceCollaboratorRole("Workspace Admin") {
		t.Fatal("legacy admin role should not be valid")
	}
}
