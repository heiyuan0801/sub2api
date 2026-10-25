package service

import (
	"context"
	"time"
)

type BatchImageDurableRecoverySource interface {
	ListRecoverableBatchImageJobs(context.Context, int64, int) ([]*BatchImageJob, error)
}
type BatchImageQueueRepair interface {
	EnsureQueued(context.Context, string) (bool, error)
}

// Rebuild Redis queue membership from durable provider job IDs. It never
// resubmits created/uploading jobs whose provider acceptance is uncertain.
func RecoverBatchImageQueue(ctx context.Context, repo BatchImageRepository, queue BatchImageQueue, after int64) (int64, int, error) {
	source, ok := repo.(BatchImageDurableRecoverySource)
	if !ok {
		return 0, 0, nil
	}
	repair, ok := queue.(BatchImageQueueRepair)
	if !ok {
		return 0, 0, nil
	}
	jobs, err := source.ListRecoverableBatchImageJobs(ctx, after, 100)
	if err != nil {
		return after, 0, err
	}
	if len(jobs) == 0 {
		return 0, 0, nil
	}
	recovered := 0
	for _, job := range jobs {
		added, e := repair.EnsureQueued(ctx, job.BatchID)
		if e != nil {
			return after, recovered, e
		}
		if added {
			recovered++
		}
		after = job.ID
	}
	return after, recovered, nil
}
func (r *BatchImageWorkerRuntime) runDurableQueueRecovery(ctx context.Context) {
	if r.cfg == nil || !r.cfg.Cluster.HAEnabled || r.billingRecovery == nil {
		return
	}
	var cursor int64
	for {
		next, _, err := RecoverBatchImageQueue(ctx, r.billingRecovery.Repo, r.billingRecovery.Queue, cursor)
		if err == nil {
			cursor = next
		}
		if ctx.Err() != nil {
			return
		}
		sleepOrDone(ctx, 15*time.Second)
	}
}
