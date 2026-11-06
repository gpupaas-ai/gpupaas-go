package v1alpha1_test

import (
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
)

func vmi(status, action string) *apiv1.VMInstance {
	return &apiv1.VMInstance{Status: apiv1.VMInstanceStatus{Status: status, Action: action}}
}

func TestVMInstanceActionDone(t *testing.T) {
	cases := []struct {
		name  string
		verb  string
		polls []*apiv1.VMInstance
		want  []bool
	}{
		{
			name:  "a first run is done once it settled with that action",
			verb:  "Power cycle",
			polls: []*apiv1.VMInstance{vmi("running", "start"), vmi("in_progress", "Power cycle"), vmi("running", "Power cycle")},
			want:  []bool{false, false, true},
		},
		{
			name:  "a different last action does not count",
			verb:  "stop",
			polls: []*apiv1.VMInstance{vmi("running", "start"), vmi("running", "start"), vmi("stopped", "stop")},
			want:  []bool{false, false, true},
		},
		{
			name:  "the same action again waits for the new run to start",
			verb:  "Power cycle",
			polls: []*apiv1.VMInstance{vmi("running", "Power cycle"), vmi("running", "Power cycle"), vmi("in_progress", "Power cycle"), vmi("running", "Power cycle")},
			want:  []bool{false, false, false, true},
		},
		{
			name:  "the same action again, first seen in progress",
			verb:  "Power cycle",
			polls: []*apiv1.VMInstance{vmi("in_progress", "Power cycle"), vmi("running", "Power cycle")},
			want:  []bool{false, true},
		},
		{
			name:  "an instance that is not settled is not done",
			verb:  "reboot",
			polls: []*apiv1.VMInstance{vmi("running", "restart"), vmi("in_progress", "reboot"), vmi("running", "reboot")},
			want:  []bool{false, false, true},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			done := typed.VMInstanceActionDone(c.verb)
			for i, obj := range c.polls {
				got, err := done(obj)
				if err != nil {
					t.Fatalf("poll %d: %v", i, err)
				}
				if got != c.want[i] {
					t.Fatalf("poll %d (%s/%s): done = %v, want %v", i, obj.Status.Status, obj.Status.Action, got, c.want[i])
				}
			}
		})
	}
}

// A run that finishes between two polls is never seen in progress: the repeated action is then
// accepted after a bounded number of settled polls instead of hanging until the timeout.
func TestVMInstanceActionDoneAcceptsAnUnseenRunEventually(t *testing.T) {
	done := typed.VMInstanceActionDone("Power cycle")
	finished := -1
	for i := 0; i < 20; i++ {
		ok, err := done(vmi("running", "Power cycle"))
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			finished = i
			break
		}
	}
	if finished < 1 || finished > 10 {
		t.Fatalf("accepted at poll %d, want after a few settled polls but not never", finished)
	}
}

// A failed state that carries another action is an earlier run's: it must not fail the new action
// before the new action has started.
func TestVMInstanceActionDoneIgnoresAStaleFailureOfAnotherAction(t *testing.T) {
	done := typed.VMInstanceActionDone("Power cycle")
	for i := 0; i < 3; i++ {
		if ok, err := done(vmi("failed", "deploy")); ok || err != nil {
			t.Fatalf("poll %d: got (%v, %v), want (false, nil)", i, ok, err)
		}
	}
	// The new action runs and fails: now the failure counts.
	if ok, err := done(vmi("in_progress", "Power cycle")); ok || err != nil {
		t.Fatalf("got (%v, %v) while the action runs", ok, err)
	}
	if _, err := done(vmi("failed", "Power cycle")); err == nil {
		t.Fatal("a failure of the running action must be reported")
	}
}

// The same action requested again after it failed: the old failed state is not reported at once, but
// a failed state that never changes is reported after a bounded number of polls.
func TestVMInstanceActionDoneRepeatedFailedAction(t *testing.T) {
	done := typed.VMInstanceActionDone("Power cycle")
	var err error
	polls := 0
	for ; polls < 20 && err == nil; polls++ {
		_, err = done(vmi("failed", "Power cycle"))
	}
	if err == nil || polls < 2 || polls > 10 {
		t.Fatalf("reported after %d polls (err %v), want after a few polls", polls, err)
	}
}

func TestVMInstanceActionDoneFailsWhenTheInstanceFails(t *testing.T) {
	done := typed.VMInstanceActionDone("Power cycle")
	if _, err := done(vmi("running", "start")); err != nil {
		t.Fatalf("a running instance must not fail: %v", err)
	}
	// The action is a new one (the last action was start) and the instance reports it failed.
	if _, err := done(vmi("failed", "Power cycle")); err == nil {
		t.Fatal("a failed instance must fail the action")
	}
}
