package engine

import (
	"testing"
	"time"

	"QueueForge/internal/config"
	"QueueForge/internal/model"
)

type fakeClock struct{ current time.Time }

func (f *fakeClock) Now() time.Time { return f.current }

func testConfig(t *testing.T) config.Config {
	t.Helper()
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	cfg.SnapshotEvery = 2
	cfg.HeartbeatGraceSeconds = 0
	return cfg
}

func TestLifecycleRetryAndComplete(t *testing.T) {
	clock := &fakeClock{current: time.Unix(1_700_000_000, 0).UTC()}
	queue, err := OpenWithClock(testConfig(t), clock)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()
	result, err := queue.Enqueue(model.EnqueueRequest{ID: "job-a", Type: "email", Payload: []byte(`{"to":"a"}`), MaxAttempts: 2, Backoff: &model.BackoffPolicy{Kind: "fixed", BaseSeconds: 5, MaxSeconds: 5}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Job.State != model.StateReady {
		t.Fatalf("state = %s", result.Job.State)
	}
	worker := model.Worker{ID: "w1", Queues: []string{"default"}, Capacity: model.Resources{CPU: 1, MemoryMB: 64, Slots: 1}}
	claimed, err := queue.Claim(model.ClaimRequest{Worker: worker})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: jobs=%d err=%v", len(claimed), err)
	}
	failed, err := queue.Fail(model.FailRequest{JobID: "job-a", LeaseToken: claimed[0].Lease.Token, Code: "temporary", Message: "try later", Retryable: true})
	if err != nil {
		t.Fatal(err)
	}
	if failed.State != model.StateRetryWait {
		t.Fatalf("state = %s", failed.State)
	}
	clock.current = clock.current.Add(5 * time.Second)
	claimed, err = queue.Claim(model.ClaimRequest{Worker: worker})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("second claim: jobs=%d err=%v", len(claimed), err)
	}
	completed, err := queue.Complete(model.CompleteRequest{JobID: "job-a", LeaseToken: claimed[0].Lease.Token, Result: []byte(`{"ok":true}`)})
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != model.StateSucceeded || completed.Attempts != 2 {
		t.Fatalf("completed=%+v", completed)
	}
}

func TestFailRejectsLeaseAfterExpiryGrace(t *testing.T) {
	clock := &fakeClock{current: time.Unix(1_700_000_000, 0).UTC()}
	cfg := testConfig(t)
	cfg.LeaseSeconds = 10
	cfg.HeartbeatGraceSeconds = 2
	queue, err := OpenWithClock(cfg, clock)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()

	if _, err := queue.Enqueue(model.EnqueueRequest{ID: "expired-fail", Type: "work"}); err != nil {
		t.Fatal(err)
	}
	worker := model.Worker{ID: "worker", Queues: []string{"default"}, Capacity: model.Resources{CPU: 1, MemoryMB: 64, Slots: 1}}
	claimed, err := queue.Claim(model.ClaimRequest{Worker: worker})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: jobs=%d err=%v", len(claimed), err)
	}
	token := claimed[0].Lease.Token
	clock.current = claimed[0].Lease.ExpiresAt.Add(cfg.HeartbeatGrace()).Add(time.Nanosecond)

	if job, err := queue.Fail(model.FailRequest{JobID: "expired-fail", LeaseToken: token, Code: "temporary", Message: "retry", Retryable: true}); err == nil || job != nil {
		t.Fatalf("Fail() = (%+v, %v), want nil job and expired-lease error", job, err)
	}
	job, ok := queue.Job("expired-fail")
	if !ok {
		t.Fatal("job disappeared")
	}
	if job.State != model.StateLeased || job.Lease == nil || job.Lease.Token != token {
		t.Fatalf("expired rejection mutated leased state: %+v", job)
	}
	if job.LastError != nil {
		t.Fatalf("expired rejection recorded failure: %+v", job.LastError)
	}
}
