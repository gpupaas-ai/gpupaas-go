package v1alpha1_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/mock/devapi"
)

func storageFixture(t *testing.T) (*devapi.Server, typed.Interface) {
	t.Helper()
	srv, cs := newDevFixture(t)
	srv.SeedStorageProfile("weka-fs", apiv1.StorageServiceTypeHighSpeed)
	srv.SeedStorageProfile("s3-small", apiv1.StorageServiceTypeObject)
	return srv, cs.V1alpha1()
}

func volume(name, profile, size string) *apiv1.Storage {
	return &apiv1.Storage{Metadata: apiv1.ObjectMeta{Name: name}, Spec: apiv1.StorageSpec{Type: profile, Size: size, Datacenter: "dc1"}}
}

var fast = typed.StorageWaitOptions{Interval: time.Millisecond}

func ready(opts typed.StorageWaitOptions) typed.StorageWaitOptions {
	opts.Done, opts.Stable = typed.StorageReady, 2
	return opts
}

func TestStorageLifecycleCreateWaitUpdateDelete(t *testing.T) {
	srv, cs := storageFixture(t)
	ctx := waitCtx(t)
	vols := cs.Workspaces(testProject).Storages("team-a")

	created, err := vols.Create(ctx, volume("data", "weka-fs", "50"), gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !created.Status.InProgress() || created.Metadata.Workspace != "team-a" || created.Spec.Type != "weka-fs" {
		t.Fatalf("a new volume provisions asynchronously: %#v", created)
	}
	if srv.Storage(testProject, "team-a", "data") == nil {
		t.Fatal("the volume must be stored at workspace scope")
	}

	done, err := typed.WaitForStorage(ctx, vols, "data", ready(fast))
	if err != nil || !done.Status.Ready() {
		t.Fatalf("wait: %#v %v", done, err)
	}
	// the output arrives as strings, the size as an integer without an exponent
	if done.Status.Output["volume_size"] != "50000000000" || done.Status.Output["filesystem_servers"] == "" {
		t.Fatalf("output: %#v", done.Status.Output)
	}

	// an update applies the new size in place and runs again
	up := *done
	up.Spec.Size = "80"
	if _, err := vols.Update(ctx, &up, gpupaas.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	done, err = typed.WaitForStorage(ctx, vols, "data", ready(fast))
	if err != nil || done.Spec.Size != "80" || done.Status.Output["volume_size"] != "80000000000" || done.Status.Action != "update" {
		t.Fatalf("after update: %#v %v", done, err)
	}

	if err := vols.Delete(ctx, "data", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	gone, err := typed.WaitForStorage(ctx, vols, "data", typed.StorageWaitOptions{Interval: time.Millisecond, Done: typed.StorageGone, GoneIsDone: true})
	if err != nil || gone != nil {
		t.Fatalf("the volume must be gone: %#v %v", gone, err)
	}
	if err := vols.Delete(ctx, "data", gpupaas.DeleteOptions{IgnoreNotFound: true}); err != nil {
		t.Fatalf("deleting a missing volume with IgnoreNotFound: %v", err)
	}
}

func TestStorageCreateFailsWithAConflictWhenTheNameIsTaken(t *testing.T) {
	_, cs := storageFixture(t)
	ctx := waitCtx(t)
	vols := cs.Storages(testProject)
	if _, err := vols.Create(ctx, volume("data", "weka-fs", "10"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := vols.Create(ctx, volume("data", "weka-fs", "10"), gpupaas.CreateOptions{}); !gpupaas.IsConflict(err) {
		t.Fatalf("a taken name must fail with 409, got %v", err)
	}
	// the same name is free in another scope
	if _, err := cs.Workspaces(testProject).Storages("w").Create(ctx, volume("data", "weka-fs", "10"), gpupaas.CreateOptions{}); err != nil {
		t.Fatalf("workspace scope: %v", err)
	}
}

func TestStorageUnknownProfileIsRejected(t *testing.T) {
	_, cs := storageFixture(t)
	_, err := cs.Storages(testProject).Create(waitCtx(t), volume("data", "nope", "10"), gpupaas.CreateOptions{})
	var apiErr *gpupaas.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 400 || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("an unknown profile must fail with 400 naming it, got %v", err)
	}
}

func TestStorageListFiltersByServiceTypeAndProjectListShowsWorkspaceVolumes(t *testing.T) {
	_, cs := storageFixture(t)
	ctx := waitCtx(t)
	must := func(_ *apiv1.Storage, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(cs.Storages(testProject).Create(ctx, volume("proj-fs", "weka-fs", "10"), gpupaas.CreateOptions{}))
	must(cs.Workspaces(testProject).Storages("w1").Create(ctx, volume("ws-fs", "weka-fs", "10"), gpupaas.CreateOptions{}))
	must(cs.Workspaces(testProject).Storages("w1").Create(ctx, volume("ws-bucket", "s3-small", "5"), gpupaas.CreateOptions{}))

	names := func(l *apiv1.StorageList, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		var n []string
		for _, i := range l.Items {
			n = append(n, i.Metadata.Name)
		}
		return strings.Join(n, ",")
	}
	if got := names(cs.Storages(testProject).List(ctx, gpupaas.ListOptions{})); got != "proj-fs,ws-bucket,ws-fs" {
		t.Errorf("a project list shows its workspaces' volumes too, got %s", got)
	}
	if got := names(cs.Storages(testProject).List(ctx, gpupaas.ListOptions{ServiceType: apiv1.StorageServiceTypeHighSpeed})); got != "proj-fs,ws-fs" {
		t.Errorf("high-speed filter: %s", got)
	}
	if got := names(cs.Storages(testProject).List(ctx, gpupaas.ListOptions{ServiceType: apiv1.StorageServiceTypeObject})); got != "ws-bucket" {
		t.Errorf("object filter: %s", got)
	}
	if got := names(cs.Workspaces(testProject).Storages("w1").List(ctx, gpupaas.ListOptions{})); got != "ws-bucket,ws-fs" {
		t.Errorf("workspace list: %s", got)
	}
}

func TestStorageProvisioningFailureCarriesThePlatformReason(t *testing.T) {
	srv, cs := storageFixture(t)
	ctx := waitCtx(t)
	srv.FailStorage("data")
	vols := cs.Storages(testProject)
	if _, err := vols.Create(ctx, volume("data", "weka-fs", "10"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	last, err := typed.WaitForStorage(ctx, vols, "data", ready(fast))
	var failed *typed.StorageFailedError
	if !errors.As(err, &failed) || failed.Phase != "failed" || failed.Action != "deploy" || !strings.Contains(failed.Reason, "provisioning is temporarily unavailable") || failed.Storage == nil {
		t.Fatalf("want a StorageFailedError with the reason, got %v", err)
	}
	if last == nil || last.Status.Output != nil {
		t.Fatalf("a failed volume has no output: %#v", last)
	}
}

func TestStorageDeleteParkedForAnAdministratorEndsTheWaitAndRefusesAnotherDelete(t *testing.T) {
	srv, cs := storageFixture(t)
	srv.ParkStorageDeletes()
	ctx := waitCtx(t)
	vols := cs.Storages(testProject)
	if _, err := vols.Create(ctx, volume("data", "weka-fs", "10"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForStorage(ctx, vols, "data", ready(fast)); err != nil {
		t.Fatal(err)
	}
	if err := vols.Delete(ctx, "data", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err := typed.WaitForStorage(ctx, vols, "data", typed.StorageWaitOptions{Interval: time.Millisecond, Done: typed.StorageGone, GoneIsDone: true})
	var failed *typed.StorageFailedError
	if !errors.As(err, &failed) || failed.Phase != apiv1.StoragePhaseDeletePending {
		t.Fatalf("a parked delete must end the wait with delete_pending, got %v", err)
	}
	var apiErr *gpupaas.APIError
	if err := vols.Delete(ctx, "data", gpupaas.DeleteOptions{}); !errors.As(err, &apiErr) || apiErr.StatusCode != 412 {
		t.Fatalf("a second delete from the end user must answer 412, got %v", err)
	}
}

func TestStorageUpdateThatTheControllerIgnoresLeavesTheOldSize(t *testing.T) {
	srv, cs := storageFixture(t)
	ctx := waitCtx(t)
	vols := cs.Storages(testProject)
	if _, err := vols.Create(ctx, volume("data", "weka-fs", "50"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	cur, err := typed.WaitForStorage(ctx, vols, "data", ready(fast))
	if err != nil {
		t.Fatal(err)
	}
	srv.IgnoreStorageSizeChange()
	cur.Spec.Size = "80"
	if _, err := vols.Update(ctx, cur, gpupaas.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	got, err := vols.Get(ctx, "data", gpupaas.GetOptions{})
	if err != nil || got.Spec.Size != "50" || !got.Status.Ready() {
		t.Fatalf("the caller must be able to see that the size did not change: %#v %v", got, err)
	}
}

func TestStorageWaitStopsWhenTheContextEnds(t *testing.T) {
	srv, cs := storageFixture(t)
	srv.SetStoragePolls(1000)
	ctx := waitCtx(t)
	vols := cs.Storages(testProject)
	if _, err := vols.Create(ctx, volume("data", "weka-fs", "10"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	short, cancel := contextWithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	_, err := typed.WaitForStorage(short, vols, "data", ready(fast))
	if err == nil || !strings.Contains(err.Error(), "waiting for storage") || !strings.Contains(err.Error(), "in_progress") {
		t.Fatalf("want a timeout naming the last phase, got %v", err)
	}
}

func contextWithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}

// scripted answers Get from a script; every other call is unused.
type scripted struct {
	typed.StorageInterface
	steps []any // *apiv1.Storage or error
	gets  int
}

func (s *scripted) Get(_ context.Context, _ string, _ gpupaas.GetOptions) (*apiv1.Storage, error) {
	step := s.steps[min(s.gets, len(s.steps)-1)]
	s.gets++
	if err, ok := step.(error); ok {
		return nil, err
	}
	return step.(*apiv1.Storage), nil
}

func TestStorageWaitNeedsConsecutiveReadsAfterATransientError(t *testing.T) {
	ok := &apiv1.Storage{Metadata: apiv1.ObjectMeta{Name: "data"}, Status: apiv1.StorageStatus{Status: "success"}}
	stub := &scripted{steps: []any{ok, &gpupaas.APIError{StatusCode: 500, Message: "blip"}, ok, ok}}
	done, err := typed.WaitForStorage(waitCtx(t), stub, "data", ready(fast))
	if err != nil || done == nil {
		t.Fatalf("wait: %v %v", done, err)
	}
	// ready, error, ready, ready: the first ready does not count towards the stable pair
	if stub.gets != 4 {
		t.Fatalf("finished after %d reads, want 4: an error between two settled reads must reset the streak", stub.gets)
	}
}

// Observed live: a delete of a failed volume ends failed on destroy once; deleting again works.
func TestStorageDeleteThatFailsOnceWorksOnRetry(t *testing.T) {
	srv, cs := storageFixture(t)
	ctx := waitCtx(t)
	vols := cs.Storages(testProject)
	if _, err := vols.Create(ctx, volume("data", "weka-fs", "10"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForStorage(ctx, vols, "data", ready(fast)); err != nil {
		t.Fatal(err)
	}
	srv.FailNextStorageDelete("data")
	gone := typed.StorageWaitOptions{Interval: time.Millisecond, Done: typed.StorageGone, GoneIsDone: true}
	if err := vols.Delete(ctx, "data", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err := typed.WaitForStorage(ctx, vols, "data", gone)
	var failed *typed.StorageFailedError
	if !errors.As(err, &failed) || failed.Phase != "failed" || failed.Action != "destroy" || !strings.Contains(failed.Reason, "Please try again") {
		t.Fatalf("the first delete must end failed on destroy, got %v", err)
	}
	if err := vols.Delete(ctx, "data", gpupaas.DeleteOptions{}); err != nil {
		t.Fatalf("a failed teardown is retryable (not parked): %v", err)
	}
	if left, err := typed.WaitForStorage(ctx, vols, "data", gone); err != nil || left != nil {
		t.Fatalf("the retry must remove the volume: %v %v", left, err)
	}
}
