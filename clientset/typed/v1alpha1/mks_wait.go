package v1alpha1

import (
	"context"
	"fmt"
	"strings"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// MKSClusterFailedError is returned by the wait helpers when a cluster ends in a failed phase.
type MKSClusterFailedError struct {
	Name   string
	Phase  string
	Action string
	Reason string
	// Cluster is the last observed state.
	Cluster *apiv1.MKSCluster
}

func (e *MKSClusterFailedError) Error() string {
	return fmt.Sprintf("mks cluster %q: %s failed (phase %q): %s", e.Name, e.Action, e.Phase, e.Reason)
}

// MKSWaitOptions configures WaitForMKSCluster.
type MKSWaitOptions struct {
	// Interval between polls. Defaults to 10 seconds.
	Interval time.Duration
	// Done reports whether the observed cluster is in the wanted state. Return a non-nil
	// error to stop and fail (for example MKSClusterFailedError).
	Done func(obj *apiv1.MKSCluster) (bool, error)
	// Stable is how many consecutive polls Done must return true before the wait ends.
	// The backend can report RUNNING, flip to UPDATING and come back to RUNNING while it
	// finishes a write, so a single RUNNING read is not proof the cluster settled.
	// Defaults to 1.
	Stable int
	// GoneIsDone ends the wait successfully when the cluster no longer exists.
	GoneIsDone bool
	// MaxConsecutiveErrors is how many consecutive transient API errors (anything other
	// than not found) are tolerated. Defaults to 10.
	MaxConsecutiveErrors int
}

// WaitForMKSCluster polls Get until opts.Done has returned true for opts.Stable consecutive
// polls, the cluster is gone (when GoneIsDone is set), an error is returned, or ctx ends (use
// context.WithTimeout for a deadline). It returns the last observed cluster, which is nil
// when the cluster is gone.
func WaitForMKSCluster(ctx context.Context, c MKSClusterInterface, name string, opts MKSWaitOptions) (*apiv1.MKSCluster, error) {
	if opts.Done == nil && !opts.GoneIsDone {
		return nil, fmt.Errorf("WaitForMKSCluster: set Done or GoneIsDone, or the wait would poll until the context ends")
	}
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
		maxErrs = 10 // about 100 s at the default 10 s interval: a short API outage must not fail a wait that takes an hour
	}
	var last *apiv1.MKSCluster
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
				return last, mksCtxError(ctx, name, last)
			}
			errs++
			streak = 0 // the stability guard needs consecutive fresh reads
			if errs >= maxErrs {
				return last, err
			}
		}
		select {
		case <-ctx.Done():
			return last, mksCtxError(ctx, name, last)
		case <-time.After(interval):
		}
	}
}

func mksCtxError(ctx context.Context, name string, last *apiv1.MKSCluster) error {
	phase, action := "", ""
	if last != nil {
		phase, action = last.Status.Phase(), last.Status.Action
	}
	return fmt.Errorf("waiting for mks cluster %q: %w (last phase %q, action %q)", name, ctx.Err(), phase, action)
}

func mksFailed(o *apiv1.MKSCluster) error {
	return &MKSClusterFailedError{Name: o.Metadata.Name, Phase: o.Status.Phase(), Action: o.Status.Action, Reason: o.Status.Reason, Cluster: o}
}

// MKSClusterRunning is a MKSWaitOptions.Done predicate that completes when the cluster is
// RUNNING and fails when it reaches ERROR or DESTROYFAILED.
func MKSClusterRunning(obj *apiv1.MKSCluster) (bool, error) {
	if obj.Status.Failed() {
		return false, mksFailed(obj)
	}
	return obj.Status.Running(), nil
}

// MKSClusterMatches returns a Done predicate that completes when the cluster is RUNNING and
// its observed spec satisfies want (see MKSSpecSatisfied). Use it after an update: the
// backend can still show RUNNING with the old spec for a moment after accepting a write.
func MKSClusterMatches(want apiv1.MKSClusterSpec) func(*apiv1.MKSCluster) (bool, error) {
	return func(obj *apiv1.MKSCluster) (bool, error) {
		if obj.Status.Failed() {
			return false, mksFailed(obj)
		}
		return obj.Status.Running() && MKSSpecSatisfied(want, obj.Spec), nil
	}
}

// MKSClusterGone is a Done predicate for delete: it never completes by itself (use GoneIsDone)
// and fails when teardown ends in DESTROYFAILED or ERROR.
func MKSClusterGone(obj *apiv1.MKSCluster) (bool, error) {
	if obj.Status.Failed() {
		return false, mksFailed(obj)
	}
	return false, nil
}

// MKSSpecSatisfied reports whether the observed spec has the Kubernetes version, control-plane
// size and worker pools (by name, with their counts) the wanted spec asks for. A wanted
// version of "1.36" is satisfied by "v1.36.4"; a leading "v" is ignored.
func MKSSpecSatisfied(want, got apiv1.MKSClusterSpec) bool {
	if w := normalizeK8sVersion(want.KubernetesVersion); w != "" {
		g := normalizeK8sVersion(got.KubernetesVersion)
		if g != w && !strings.HasPrefix(g, w+".") {
			return false
		}
	}
	if want.ControlPlane != nil && want.ControlPlane.Scaling != nil {
		if got.ControlPlane == nil || !scalingSatisfied(want.ControlPlane.Scaling, got.ControlPlane.Scaling) {
			return false
		}
	}
	if len(want.WorkerPools) != len(got.WorkerPools) {
		return false
	}
	byName := map[string]apiv1.MKSNodeGroup{}
	for _, p := range got.WorkerPools {
		byName[p.Name] = p
	}
	for _, p := range want.WorkerPools {
		g, ok := byName[p.Name]
		if !ok {
			return false
		}
		if !scalingSatisfied(p.Scaling, g.Scaling) {
			return false
		}
	}
	return true
}

// scalingSatisfied compares the wanted and observed scaling of a node group. A manual group must
// have the wanted count. An automatic group must have the wanted bounds: its current size is chosen
// by the autoscaler between them, so it is not compared.
func scalingSatisfied(want, got *apiv1.MKSScaling) bool {
	if want == nil {
		return true
	}
	if got == nil {
		return false
	}
	if strings.EqualFold(want.Mode, apiv1.MKSScalingAutomatic) {
		if got.Mode != "" && !strings.EqualFold(got.Mode, apiv1.MKSScalingAutomatic) {
			return false
		}
		return got.Min == want.Min && got.Max == want.Max
	}
	if strings.EqualFold(got.Mode, apiv1.MKSScalingAutomatic) {
		return false
	}
	return got.Count == want.Count
}

func normalizeK8sVersion(v string) string { return strings.TrimPrefix(strings.TrimSpace(v), "v") }
