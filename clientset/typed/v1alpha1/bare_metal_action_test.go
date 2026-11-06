package v1alpha1_test

import (
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
)

func bm(status, action string) *apiv1.BareMetal {
	return &apiv1.BareMetal{Status: apiv1.BareMetalStatus{Status: status, Action: action}}
}

func TestBareMetalActionDone(t *testing.T) {
	cases := []struct {
		name  string
		verb  string
		polls []*apiv1.BareMetal
		want  []bool
	}{
		{
			name:  "a first run is done once it settled with that action",
			verb:  "Power cycle",
			polls: []*apiv1.BareMetal{bm("running", "start"), bm("in_progress", "Power cycle"), bm("running", "Power cycle")},
			want:  []bool{false, false, true},
		},
		{
			name:  "a different last action does not count",
			verb:  "stop",
			polls: []*apiv1.BareMetal{bm("running", "start"), bm("running", "start"), bm("stopped", "stop")},
			want:  []bool{false, false, true},
		},
		{
			name:  "the same action again waits for the new run to start",
			verb:  "Power cycle",
			polls: []*apiv1.BareMetal{bm("running", "Power cycle"), bm("running", "Power cycle"), bm("in_progress", "Power cycle"), bm("running", "Power cycle")},
			want:  []bool{false, false, false, true},
		},
		{
			name:  "the same action again, first seen in progress",
			verb:  "Power cycle",
			polls: []*apiv1.BareMetal{bm("in_progress", "Power cycle"), bm("running", "Power cycle")},
			want:  []bool{false, true},
		},
		{
			name:  "an instance that is not settled is not done",
			verb:  "reboot",
			polls: []*apiv1.BareMetal{bm("running", "restart"), bm("in_progress", "reboot"), bm("running", "reboot")},
			want:  []bool{false, false, true},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			done := typed.BareMetalActionDone(c.verb)
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
func TestBareMetalActionDoneAcceptsAnUnseenRunEventually(t *testing.T) {
	done := typed.BareMetalActionDone("Power cycle")
	finished := -1
	for i := 0; i < 20; i++ {
		ok, err := done(bm("running", "Power cycle"))
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
func TestBareMetalActionDoneIgnoresAStaleFailureOfAnotherAction(t *testing.T) {
	done := typed.BareMetalActionDone("Power cycle")
	for i := 0; i < 3; i++ {
		if ok, err := done(bm("failed", "deploy")); ok || err != nil {
			t.Fatalf("poll %d: got (%v, %v), want (false, nil)", i, ok, err)
		}
	}
	// The new action runs and fails: now the failure counts.
	if ok, err := done(bm("in_progress", "Power cycle")); ok || err != nil {
		t.Fatalf("got (%v, %v) while the action runs", ok, err)
	}
	if _, err := done(bm("failed", "Power cycle")); err == nil {
		t.Fatal("a failure of the running action must be reported")
	}
}

// The same action requested again after it failed: the old failed state is not reported at once, but
// a failed state that never changes is reported after a bounded number of polls.
func TestBareMetalActionDoneRepeatedFailedAction(t *testing.T) {
	done := typed.BareMetalActionDone("Power cycle")
	var err error
	polls := 0
	for ; polls < 20 && err == nil; polls++ {
		_, err = done(bm("failed", "Power cycle"))
	}
	if err == nil || polls < 2 || polls > 10 {
		t.Fatalf("reported after %d polls (err %v), want after a few polls", polls, err)
	}
}

func TestBareMetalActionDoneFailsWhenTheInstanceFails(t *testing.T) {
	done := typed.BareMetalActionDone("Power cycle")
	if _, err := done(bm("running", "start")); err != nil {
		t.Fatalf("a running instance must not fail: %v", err)
	}
	// The action is a new one (the last action was start) and the instance reports it failed.
	if _, err := done(bm("failed", "Power cycle")); err == nil {
		t.Fatal("a failed instance must fail the action")
	}
}
