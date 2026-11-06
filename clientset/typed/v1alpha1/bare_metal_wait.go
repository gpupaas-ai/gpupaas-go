package v1alpha1

import (
	"context"
	"fmt"
	"strings"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// BareMetalFailedError is returned by the wait helpers when an instance ends in a failed phase.
type BareMetalFailedError struct {
	Name   string
	Phase  string
	Action string
	Reason string
	// BareMetal is the last observed state.
	BareMetal *apiv1.BareMetal
}

func (e *BareMetalFailedError) Error() string {
	return fmt.Sprintf("bare metal %q: %s failed (phase %q): %s", e.Name, e.Action, e.Phase, e.Reason)
}

// BareMetalWaitOptions configures WaitForBareMetal.
type BareMetalWaitOptions struct {
	// Interval between polls. Defaults to 10 seconds.
	Interval time.Duration
	// Done reports whether the observed instance is in the wanted state. Return a non-nil error to
	// stop and fail (for example BareMetalFailedError).
	Done func(obj *apiv1.BareMetal) (bool, error)
	// Stable is how many consecutive polls Done must return true before the wait ends. Defaults to 1.
	Stable int
	// GoneIsDone ends the wait successfully when the instance no longer exists.
	GoneIsDone bool
	// MaxConsecutiveErrors is how many consecutive transient API errors (anything other than not
	// found) are tolerated. Defaults to 10.
	MaxConsecutiveErrors int
}

// WaitForBareMetal polls Get until opts.Done has returned true for opts.Stable consecutive polls,
// the instance is gone (when GoneIsDone is set), an error is returned, or ctx ends (use
// context.WithTimeout for a deadline). It returns the last observed instance, which is nil when
// the instance is gone.
func WaitForBareMetal(ctx context.Context, c BareMetalInterface, name string, opts BareMetalWaitOptions) (*apiv1.BareMetal, error) {
	interval := opts.Interval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	stable := opts.Stable
	if stable <= 0 {
		stable = 1
	}
	maxErrs := opts.MaxConsecutiveErrors
	if maxErrs <= 0 {
		maxErrs = 10 // about 2.5 minutes at a 15 s interval: a short API outage must not fail a create that takes an hour
	}
	var last *apiv1.BareMetal
	errs, streak := 0, 0
	for {
		obj, err := c.Get(ctx, name, gpupaas.GetOptions{})
		switch {
		case err == nil:
			errs = 0
			last = obj
			if opts.Done != nil {
				done, derr := opts.Done(obj)
				if derr != nil {
					return last, derr
				}
				if done {
					streak++
					if streak >= stable {
						return last, nil
					}
				} else {
					streak = 0
				}
			}
		case gpupaas.IsNotFound(err):
			if opts.GoneIsDone {
				return nil, nil
			}
			return last, err
		default:
			if ctx.Err() != nil {
				return last, bareMetalCtxError(ctx, name, last)
			}
			errs++
			if errs >= maxErrs {
				return last, err
			}
		}
		select {
		case <-ctx.Done():
			return last, bareMetalCtxError(ctx, name, last)
		case <-time.After(interval):
		}
	}
}

func bareMetalCtxError(ctx context.Context, name string, last *apiv1.BareMetal) error {
	phase, action := "", ""
	if last != nil {
		phase, action = last.Status.Phase(), last.Status.Action
	}
	return fmt.Errorf("waiting for bare metal %q: %w (last phase %q, action %q)", name, ctx.Err(), phase, action)
}

func bareMetalFailed(o *apiv1.BareMetal) error {
	return &BareMetalFailedError{Name: o.Metadata.Name, Phase: o.Status.Phase(), Action: o.Status.Action, Reason: o.Status.Reason, BareMetal: o}
}

// BareMetalRunning is a BareMetalWaitOptions.Done predicate that completes when the machine is
// running and fails when provisioning fails.
func BareMetalRunning(obj *apiv1.BareMetal) (bool, error) {
	if obj.Status.Failed() {
		return false, bareMetalFailed(obj)
	}
	return obj.Status.Running(), nil
}

// BareMetalGone is a Done predicate for delete: it never completes by itself (use GoneIsDone) and
// fails when the delete fails or is parked for the operator (delete_pending).
func BareMetalGone(obj *apiv1.BareMetal) (bool, error) {
	if obj.Status.Failed() {
		return false, bareMetalFailed(obj)
	}
	return false, nil
}

// BareMetalActionDone returns a Done predicate for an action: it completes when the instance has
// settled (running or stopped) with verb as its last action, and fails when the instance fails.
// "restart" and "reboot" are the same action. Use it with Stable >= 2: right after the request the
// backend can still show the previous state.
//
// When the instance's last action already was verb (the same action requested again), a settled
// state proves nothing: it may be the previous run. The predicate then needs to see the new run
// start (an in-progress state) first. If it never does (the run finished between two polls), it
// accepts the settled state after actionUnseenPolls polls.
func BareMetalActionDone(verb string) func(*apiv1.BareMetal) (bool, error) {
	want := normalizeBareMetalVerb(verb)
	var (
		started      bool // the first poll has been seen
		repeated     bool // the instance's last action, before this request, already was verb
		sawActive    bool // an in-progress state has been seen
		settledPolls int  // settled polls of a repeated action that never showed the new run
		failedPolls  int  // failed polls of a repeated action that never showed the new run
	)
	return func(obj *apiv1.BareMetal) (bool, error) {
		if !started {
			started = true
			repeated = normalizeBareMetalVerb(obj.Status.Action) == want
		}
		if obj.Status.Phase() == apiv1.BareMetalPhaseInProgress {
			sawActive = true
		}
		// A failure counts as this action's only when it carries this action, or after the action
		// was seen running: right after the request the backend can still show an earlier run's
		// failed state, which must not fail the new action before it started.
		if obj.Status.Failed() {
			sameAction := normalizeBareMetalVerb(obj.Status.Action) == want
			switch {
			case sawActive, sameAction && !repeated:
				return false, bareMetalFailed(obj)
			case sameAction:
				// The same action again: the failed state may be the previous run's. If it never
				// changes, the new run failed between two polls.
				if failedPolls++; failedPolls >= actionUnseenPolls {
					return false, bareMetalFailed(obj)
				}
			}
			return false, nil
		}
		if normalizeBareMetalVerb(obj.Status.Action) != want {
			return false, nil
		}
		switch obj.Status.Phase() {
		case apiv1.BareMetalPhaseRunning, apiv1.BareMetalPhaseStopped:
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

// actionUnseenPolls is how many settled polls of a repeated action are accepted without ever seeing
// the new run in progress.
const actionUnseenPolls = 8

func normalizeBareMetalVerb(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "restart" {
		return "reboot"
	}
	return strings.ReplaceAll(v, " ", "")
}
