package v1alpha1

import (
	"context"
	"fmt"
	"strings"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// VMInstanceFailedError is returned by the wait helpers when an instance ends in a failed phase.
type VMInstanceFailedError struct {
	Name   string
	Phase  string
	Action string
	Reason string
	// VMInstance is the last observed state.
	VMInstance *apiv1.VMInstance
}

func (e *VMInstanceFailedError) Error() string {
	return fmt.Sprintf("virtual machine %q: %s failed (phase %q): %s", e.Name, e.Action, e.Phase, e.Reason)
}

// VMInstanceWaitOptions configures WaitForVMInstance.
type VMInstanceWaitOptions struct {
	// Interval between polls. Defaults to 10 seconds.
	Interval time.Duration
	// Done reports whether the observed instance is in the wanted state. Return a non-nil error to
	// stop and fail (for example VMInstanceFailedError).
	Done func(obj *apiv1.VMInstance) (bool, error)
	// Stable is how many consecutive polls Done must return true before the wait ends. Defaults to 1.
	Stable int
	// GoneIsDone ends the wait successfully when the instance no longer exists.
	GoneIsDone bool
	// MaxConsecutiveErrors is how many consecutive transient API errors (anything other than not
	// found) are tolerated. Defaults to 10.
	MaxConsecutiveErrors int
}

// WaitForVMInstance polls Get until opts.Done has returned true for opts.Stable consecutive polls,
// the instance is gone (when GoneIsDone is set), an error is returned, or ctx ends (use
// context.WithTimeout for a deadline). It returns the last observed instance, which is nil when
// the instance is gone.
func WaitForVMInstance(ctx context.Context, c VMInstanceInterface, name string, opts VMInstanceWaitOptions) (*apiv1.VMInstance, error) {
	return pollUntil(ctx, pollParams[apiv1.VMInstance]{
		fn:         "WaitForVMInstance",
		get:        func(ctx context.Context) (*apiv1.VMInstance, error) { return c.Get(ctx, name, gpupaas.GetOptions{}) },
		done:       opts.Done,
		ctxError:   func(ctx context.Context, last *apiv1.VMInstance) error { return vmInstanceCtxError(ctx, name, last) },
		interval:   opts.Interval,
		stable:     opts.Stable,
		maxErrs:    opts.MaxConsecutiveErrors,
		goneIsDone: opts.GoneIsDone,
	})
}

func vmInstanceCtxError(ctx context.Context, name string, last *apiv1.VMInstance) error {
	phase, action := "", ""
	if last != nil {
		phase, action = last.Status.Phase(), last.Status.Action
	}
	return fmt.Errorf("waiting for virtual machine %q: %w (last phase %q, action %q)", name, ctx.Err(), phase, action)
}

func vmInstanceFailed(o *apiv1.VMInstance) error {
	return &VMInstanceFailedError{Name: o.Metadata.Name, Phase: o.Status.Phase(), Action: o.Status.Action, Reason: o.Status.Reason, VMInstance: o}
}

// VMInstanceRunning is a VMInstanceWaitOptions.Done predicate that completes when the machine is
// running and fails when provisioning fails.
func VMInstanceRunning(obj *apiv1.VMInstance) (bool, error) {
	if obj.Status.Failed() {
		return false, vmInstanceFailed(obj)
	}
	return obj.Status.Running(), nil
}

// VMInstanceGone is a Done predicate for delete: it never completes by itself (use GoneIsDone) and
// fails when the delete fails or is parked for the operator (delete_pending).
func VMInstanceGone(obj *apiv1.VMInstance) (bool, error) {
	if obj.Status.Failed() {
		return false, vmInstanceFailed(obj)
	}
	return false, nil
}

// VMInstanceActionDone returns a Done predicate for an action: it completes when the instance has
// settled (running or stopped) with verb as its last action, and fails when the instance fails.
// "restart" and "reboot" are the same action. Use it with Stable >= 2: right after the request the
// backend can still show the previous state.
//
// When the instance's last action already was verb (the same action requested again), a settled
// state proves nothing: it may be the previous run. The predicate then needs to see the new run
// start (an in-progress state) first. If it never does (the run finished between two polls), it
// accepts the settled state after actionUnseenPolls polls.
func VMInstanceActionDone(verb string) func(*apiv1.VMInstance) (bool, error) {
	want := normalizeVMInstanceVerb(verb)
	var (
		started      bool // the first poll has been seen
		repeated     bool // the instance's last action, before this request, already was verb
		sawActive    bool // an in-progress state has been seen
		settledPolls int  // settled polls of a repeated action that never showed the new run
		failedPolls  int  // failed polls of a repeated action that never showed the new run
	)
	return func(obj *apiv1.VMInstance) (bool, error) {
		if !started {
			started = true
			repeated = normalizeVMInstanceVerb(obj.Status.Action) == want
		}
		if obj.Status.Phase() == apiv1.VMPhaseInProgress {
			sawActive = true
		}
		// A failure counts as this action's only when it carries this action, or after the action
		// was seen running: right after the request the backend can still show an earlier run's
		// failed state, which must not fail the new action before it started.
		if obj.Status.Failed() {
			sameAction := normalizeVMInstanceVerb(obj.Status.Action) == want
			switch {
			case sawActive, sameAction && !repeated:
				return false, vmInstanceFailed(obj)
			case sameAction:
				// The same action again: the failed state may be the previous run's. If it never
				// changes, the new run failed between two polls.
				if failedPolls++; failedPolls >= actionUnseenPolls {
					return false, vmInstanceFailed(obj)
				}
			}
			return false, nil
		}
		if normalizeVMInstanceVerb(obj.Status.Action) != want {
			return false, nil
		}
		switch obj.Status.Phase() {
		case apiv1.VMPhaseRunning, apiv1.VMPhaseStopped:
			if repeated && !sawActive {
				if settledPolls++; settledPolls < actionUnseenPolls {
					return false, nil
				}
			}
			return true, nil
		}
		return false, nil
	}
}

func normalizeVMInstanceVerb(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "restart" {
		return "reboot"
	}
	return strings.ReplaceAll(v, " ", "")
}
