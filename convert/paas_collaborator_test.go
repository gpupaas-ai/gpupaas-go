package convert_test

import (
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func TestToPaaSWorkspaceCollaborator(t *testing.T) {
	in := &apiv1.WorkspaceCollaborator{
		Metadata: apiv1.ObjectMeta{
			Name:      "guest@example.com",
			Project:   "demo",
			Workspace: "ws1",
		},
		Spec: apiv1.WorkspaceCollaboratorSpec{
			Email:     "guest@example.com",
			FirstName: "Guest",
			LastName:  "User",
			Role:      apiv1.WorkspaceRoleCollaborator,
			UserType:  "Console",
		},
	}
	wire := convert.ToPaaSWorkspaceCollaborator(in, "demo", "ws1")
	if wire.Kind != convert.PaaSWorkspaceCollaboratorKind {
		t.Fatalf("kind: %q", wire.Kind)
	}
	if wire.Spec.Email != "guest@example.com" || wire.Spec.Role != apiv1.WorkspaceRoleCollaborator {
		t.Fatalf("spec: %+v", wire.Spec)
	}
}

func TestToPaaSWorkspaceAddCollaborators(t *testing.T) {
	in := &apiv1.WorkspaceCollaborator{
		Metadata: apiv1.ObjectMeta{
			Name:      "alice",
			Project:   "demo",
			Workspace: "ws1",
		},
		Spec: apiv1.WorkspaceCollaboratorSpec{
			Username:  "alice",
			Role:      apiv1.WorkspaceRoleCollaboratorReadOnly,
			IsSSOUser: true,
		},
	}
	wire := convert.ToPaaSWorkspaceAddCollaborators(in, "demo", "ws1")
	if len(wire.Spec.Usernames) != 1 || wire.Spec.Usernames[0] != "alice" {
		t.Fatalf("usernames: %+v", wire.Spec.Usernames)
	}
	if wire.Spec.Role != apiv1.WorkspaceRoleCollaboratorReadOnly {
		t.Fatalf("role: %q", wire.Spec.Role)
	}
}

func TestFromPaaSWorkspaceCollaboratorList(t *testing.T) {
	list := convert.FromPaaSWorkspaceCollaboratorList(&convert.PaaSWorkspaceCollaboratorList{
		APIVersion: convert.PaaSWorkspaceAPIVer,
		Kind:       convert.PaaSWorkspaceCollaboratorListKind,
		Items: []convert.PaaSWorkspaceCollaborator{
			{
				Metadata: convert.PaaSMetadata{Name: "alice", Project: "demo"},
				Spec:     convert.PaaSWorkspaceCollaboratorSpec{Role: apiv1.WorkspaceRoleCollaborator},
				Status:   convert.PaaSStatus{ConditionStatus: "StatusOK"},
			},
		},
	}, "ws1")
	if len(list.Items) != 1 {
		t.Fatalf("items: %d", len(list.Items))
	}
	item := list.Items[0]
	if item.Metadata.Workspace != "ws1" || item.Metadata.Name != "alice" {
		t.Fatalf("item metadata: %+v", item.Metadata)
	}
	if item.Spec.Role != apiv1.WorkspaceRoleCollaborator || item.Status.Phase != "StatusOK" {
		t.Fatalf("item: %+v", item)
	}
	if item.Status.Role != apiv1.WorkspaceRoleCollaborator {
		t.Fatalf("status role: %q", item.Status.Role)
	}
}
