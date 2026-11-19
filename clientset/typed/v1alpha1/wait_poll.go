package v1alpha1

import (
	"context"
	"fmt"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
)

// pollParams describes one wait. Every WaitFor* function fills it from its own options type and shares pollUntil, so the
// retry budget, the stable-poll rule and the not-found handling cannot drift between resource kinds.
type pollParams[T any] struct {
	// fn names the public function for the error that reports an empty wait.
	fn string
	// get reads the object once.
	get func(ctx context.Context) (*T, error)
	// done reports whether the object is in the wanted state; a non-nil error stops the wait and fails it.
	done func(obj *T) (bool, error)
	// ctxError builds the error returned when the context ends, from the last object seen.
	ctxError func(ctx context.Context, last *T) error

	interval   time.Duration
	stable     int
	maxErrs    int
	goneIsDone bool
}

// pollUntil polls get until done has returned true for stable consecutive polls, the object is gone (when goneIsDone
// is set), a hard error is returned, or ctx ends. It returns the last object seen, which is nil when it is gone.
func pollUntil[T any](ctx context.Context, p pollParams[T]) (*T, error) {
	if p.done == nil && !p.goneIsDone {
		return nil, fmt.Errorf("%s: set Done or GoneIsDone, or the wait would poll until the context ends", p.fn)
	}
	interval := p.interval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	stable := p.stable
	if stable <= 0 {
		stable = 1
	}
	maxErrs := p.maxErrs
	if maxErrs <= 0 {
		maxErrs = 10 // about 100 s at the default 10 s interval: a short API outage must not fail a wait that takes an hour
	}
	var last *T
	errs, streak := 0, 0
	for {
		obj, err := p.get(ctx)
		switch {
		case err == nil:
			errs = 0
			last = obj
			if p.done != nil {
				done, derr := p.done(obj)
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
			if p.goneIsDone {
				return nil, nil
			}
			return last, err
		default:
			if ctx.Err() != nil {
				return last, p.ctxError(ctx, last)
			}
			errs++
			streak = 0 // the stability guard needs consecutive fresh reads
			if errs >= maxErrs {
				return last, err
			}
		}
		select {
		case <-ctx.Done():
			return last, p.ctxError(ctx, last)
		case <-time.After(interval):
		}
	}
}
