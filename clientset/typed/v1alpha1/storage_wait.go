package v1alpha1

import (
	"context"
	"fmt"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// StorageFailedError is returned by the wait helpers when a volume ends in a failed phase.
type StorageFailedError struct {
	Name   string
	Phase  string
	Action string
	Reason string
	// Storage is the last observed state.
	Storage *apiv1.Storage
}

func (e *StorageFailedError) Error() string {
	return fmt.Sprintf("storage %q: %s failed (phase %q): %s", e.Name, e.Action, e.Phase, e.Reason)
}

// StorageWaitOptions configures WaitForStorage.
type StorageWaitOptions struct {
	// Interval between polls. Defaults to 10 seconds.
	Interval time.Duration
	// Done reports whether the observed volume is in the wanted state. Return a non-nil error to stop and
	// fail (for example StorageFailedError).
	Done func(obj *apiv1.Storage) (bool, error)
	// Stable is how many consecutive polls Done must return true before the wait ends. Defaults to 1.
	Stable int
	// GoneIsDone ends the wait successfully when the volume no longer exists.
	GoneIsDone bool
	// MaxConsecutiveErrors is how many consecutive transient API errors (anything other than not found)
	// are tolerated. Defaults to 10.
	MaxConsecutiveErrors int
}

// WaitForStorage polls Get until opts.Done has returned true for opts.Stable consecutive polls, the volume
// is gone (when GoneIsDone is set), an error is returned, or ctx ends (use context.WithTimeout for a
// deadline). It returns the last observed volume, which is nil when the volume is gone.
func WaitForStorage(ctx context.Context, c StorageInterface, name string, opts StorageWaitOptions) (*apiv1.Storage, error) {
	if opts.Done == nil && !opts.GoneIsDone {
		return nil, fmt.Errorf("WaitForStorage: set Done or GoneIsDone, or the wait would poll until the context ends")
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
		maxErrs = 10
	}
	var last *apiv1.Storage
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
				return last, storageCtxError(ctx, name, last)
			}
			errs++
			streak = 0 // the stability guard needs consecutive fresh reads
			if errs >= maxErrs {
				return last, err
			}
		}
		select {
		case <-ctx.Done():
			return last, storageCtxError(ctx, name, last)
		case <-time.After(interval):
		}
	}
}

func storageCtxError(ctx context.Context, name string, last *apiv1.Storage) error {
	phase, action := "", ""
	if last != nil {
		phase, action = last.Status.Phase(), last.Status.Action
	}
	return fmt.Errorf("waiting for storage %q: %w (last phase %q, action %q)", name, ctx.Err(), phase, action)
}

func storageFailed(o *apiv1.Storage) error {
	return &StorageFailedError{Name: o.Metadata.Name, Phase: o.Status.Phase(), Action: o.Status.Action, Reason: o.Status.Reason, Storage: o}
}

// StorageReady is a StorageWaitOptions.Done predicate that completes when the volume is provisioned and
// fails when provisioning fails.
func StorageReady(obj *apiv1.Storage) (bool, error) {
	if obj.Status.Failed() {
		return false, storageFailed(obj)
	}
	return obj.Status.Ready(), nil
}

// StorageGone is a Done predicate for delete: it never completes by itself (use GoneIsDone) and fails when
// the delete fails or is parked for the operator (delete_pending).
func StorageGone(obj *apiv1.Storage) (bool, error) {
	if obj.Status.Failed() {
		return false, storageFailed(obj)
	}
	return false, nil
}
