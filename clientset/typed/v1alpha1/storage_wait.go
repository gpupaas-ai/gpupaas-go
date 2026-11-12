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
	return pollUntil(ctx, pollParams[apiv1.Storage]{
		fn:         "WaitForStorage",
		get:        func(ctx context.Context) (*apiv1.Storage, error) { return c.Get(ctx, name, gpupaas.GetOptions{}) },
		done:       opts.Done,
		ctxError:   func(ctx context.Context, last *apiv1.Storage) error { return storageCtxError(ctx, name, last) },
		interval:   opts.Interval,
		stable:     opts.Stable,
		maxErrs:    opts.MaxConsecutiveErrors,
		goneIsDone: opts.GoneIsDone,
	})
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
