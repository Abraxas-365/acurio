package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Abraxas-365/relay/pkg/logx"
	"github.com/Abraxas-365/relay/recruitment/resume"
	"github.com/Abraxas-365/relay/recruitment/resume/resumesrv"
)

type ResumeWorker struct {
	service *resumesrv.Service
	queue   resume.JobQueue
	workers int
}

func NewResumeWorker(service *resumesrv.Service, queue resume.JobQueue, workers int) *ResumeWorker {
	return &ResumeWorker{
		service: service,
		queue:   queue,
		workers: workers,
	}
}

func (w *ResumeWorker) Start(ctx context.Context) {
	logx.Infof("Starting %d resume workers", w.workers)

	// Start delayed job mover
	go w.moveDelayedJobs(ctx)

	// Start worker pool
	for i := 0; i < w.workers; i++ {
		go w.processJobs(ctx, i)
	}
}

func (w *ResumeWorker) processJobs(ctx context.Context, workerID int) {
	logx.Infof("Worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			logx.Infof("Worker %d stopping", workerID)
			return
		default:
			// Dequeue with 5 second timeout
			data, err := w.queue.Dequeue(ctx, 5*time.Second)
			if err != nil {
				if err.Error() != "redis: nil" { // Timeout is not an error
					logx.Errorf("Worker %d dequeue error: %v", workerID, err)
				}
				continue
			}

			// Check if data is empty (queue timeout - no jobs available)
			if data == nil || len(data) == 0 {
				continue
			}

			// Parse job
			var job resume.ResumeProcessingJob
			if err := json.Unmarshal(data, &job); err != nil {
				logx.Errorf("Worker %d unmarshal error: %v (data: %s)", workerID, err, string(data))
				continue
			}

			// Process job
			logx.Infof("Worker %d processing job: %s", workerID, job.ID)
			if err := w.service.ProcessResumeJob(ctx, &job); err != nil {
				logx.Errorf("Worker %d job failed: %v", workerID, err)
			}
		}
	}
}

func (w *ResumeWorker) moveDelayedJobs(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := w.queue.MoveDelayedToReady(ctx)
			if err != nil {
				logx.Errorf("Failed to move delayed jobs: %v", err)
			} else if count > 0 {
				logx.Infof("Moved %d delayed jobs to ready queue", count)
			}
		}
	}
}

