package resumesrv

import (
	"context"
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/internal/ai/resumeparser"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/pkg/logx"
	"github.com/Abraxas-365/relay/recruitment/resume"
	"github.com/google/uuid"
)

// ParseResumeAsync - Queue the resume for background processing
func (s *Service) ParseResumeAsync(ctx context.Context, req resume.ParseResumeRequest) (*resume.JobStatusResponse, error) {
	logx.Infof("Queueing resume for async processing: TenantID=%s, File=%s", req.TenantID, req.FileName)

	// Check if tenant has reached max resumes limit
	count, err := s.repo.CountByTenantID(ctx, req.TenantID)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", req.TenantID)
	}
	if count >= MaxResumesPerTenant {
		return nil, resume.ErrMaxResumesExceeded().
			WithDetail("tenant_id", req.TenantID).
			WithDetail("current_count", count).
			WithDetail("max_allowed", MaxResumesPerTenant)
	}

	// Create job record
	jobID := kernel.NewJobID(uuid.NewString())
	job := &resume.ResumeProcessingJob{
		ID:                 jobID,
		TenantID:           req.TenantID,
		Status:             resume.JobStatusPending,
		FilePath:           req.FilePath,
		FileName:           req.FileName,
		FileType:           req.FileType,
		Title:              req.Title,
		AttemptCount:       0,
		MaxAttempts:        3,
		ProgressPercentage: 0,
		CreatedAt:          time.Now(),
		RequestPayload:     req,
	}

	// Save job to database
	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, resume.ErrJobCreationFailed().
			WithDetail("tenant_id", req.TenantID).
			WithDetail("file_name", req.FileName).
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	// Enqueue to Redis
	if err := s.queue.Enqueue(ctx, jobID, job); err != nil {
		// Mark job as failed if we can't queue it
		_ = s.jobRepo.MarkAsFailed(ctx, jobID, "failed to enqueue", map[string]any{
			"error": err.Error(),
		})

		return nil, resume.ErrQueueEnqueueFailed().
			WithDetail("job_id", jobID).
			WithDetail("tenant_id", req.TenantID).
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	logx.Infof("Job queued successfully: JobID=%s", jobID)

	return &resume.JobStatusResponse{
		JobID:    jobID,
		TenantID: req.TenantID,
		Status:   resume.JobStatusPending,
		Message:  "Resume queued for processing",
		Progress: 0,
	}, nil
}

// ProcessResumeJob - Worker function to process a job
func (s *Service) ProcessResumeJob(ctx context.Context, job *resume.ResumeProcessingJob) error {
	logx.Infof("Processing job: JobID=%s, Attempt=%d/%d", job.ID, job.AttemptCount+1, job.MaxAttempts)

	// Mark as processing
	if err := s.jobRepo.MarkAsProcessing(ctx, job.ID); err != nil {
		return resume.ErrJobUpdateFailed().
			WithDetail("job_id", job.ID).
			WithDetail("status", "processing").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	// Update progress: Parsing
	_ = s.jobRepo.UpdateProgress(ctx, job.ID, resume.StepParsing, 25)

	// Read file
	fileData, err := s.fileReader.ReadFile(ctx, job.FilePath)
	if err != nil {
		return s.handleJobError(ctx, job, "file_read_failed", err)
	}

	// Parse resume
	var parsedData *resumeparser.ResumeData
	switch job.FileType {
	case "pdf":
		parsedData, err = s.parsePDFResume(ctx, fileData)
	case "jpg", "jpeg", "png":
		parsedData, err = s.parseImageResume(ctx, fileData)
	default:
		return s.handleJobError(ctx, job, "invalid_file_type",
			fmt.Errorf("unsupported file type: %s", job.FileType))
	}

	if err != nil {
		return s.handleJobError(ctx, job, "parsing_failed", err)
	}

	// Update progress: Generating embeddings
	_ = s.jobRepo.UpdateProgress(ctx, job.ID, resume.StepEmbedding, 50)

	// Convert to domain model
	resumeModel := s.convertParsedDataToDomain(parsedData, job.RequestPayload)

	// Generate embeddings
	embeddings, err := s.generateResumeEmbeddings(ctx, resumeModel)
	if err != nil {
		return s.handleJobError(ctx, job, "embedding_generation_failed", err)
	}
	resumeModel.Embeddings = *embeddings

	// Update progress: Saving
	_ = s.jobRepo.UpdateProgress(ctx, job.ID, resume.StepSaving, 75)

	// Handle default resume logic
	if job.RequestPayload.IsDefault {
		_ = s.unsetOtherDefaults(ctx, job.TenantID)
	}

	// Save resume
	if err := s.repo.Create(ctx, resumeModel); err != nil {
		return s.handleJobError(ctx, job, "save_failed", err)
	}

	// Mark as completed
	if err := s.jobRepo.MarkAsCompleted(ctx, job.ID, resumeModel.ID); err != nil {
		logx.Errorf("Failed to mark job as completed: %v", err)
		// Don't fail the job if we can't update status - resume was created successfully
	}

	_ = s.jobRepo.UpdateProgress(ctx, job.ID, resume.StepSaving, 100)

	logx.Infof("Job completed successfully: JobID=%s, ResumeID=%s", job.ID, resumeModel.ID)
	return nil
}

// handleJobError handles job processing errors with retry logic
func (s *Service) handleJobError(ctx context.Context, job *resume.ResumeProcessingJob, errorType string, err error) error {
	job.AttemptCount++

	errorDetails := map[string]any{
		"error":        err.Error(),
		"error_type":   errorType,
		"attempt":      job.AttemptCount,
		"max_attempts": job.MaxAttempts,
		"file_path":    job.FilePath,
		"file_name":    job.FileName,
	}

	// Check if we should retry
	if job.AttemptCount < job.MaxAttempts {
		// Calculate exponential backoff: 2^attempt minutes
		retryDelay := time.Duration(1<<uint(job.AttemptCount)) * time.Minute
		nextRetry := time.Now().Add(retryDelay)
		job.NextRetryAt = &nextRetry

		logx.Warnf("Job failed, will retry: JobID=%s, Attempt=%d/%d, NextRetry=%v, Error=%s",
			job.ID, job.AttemptCount, job.MaxAttempts, nextRetry, errorType)

		// Enqueue for retry
		if queueErr := s.queue.EnqueueDelayed(ctx, job.ID, job, retryDelay); queueErr != nil {
			logx.Errorf("Failed to enqueue for retry: %v", queueErr)

			// If we can't enqueue, mark as failed
			_ = s.jobRepo.MarkAsFailed(ctx, job.ID,
				fmt.Sprintf("%s (retry enqueue failed)", errorType),
				errorDetails)

			return resume.ErrJobRetryFailed().
				WithDetail("job_id", job.ID).
				WithDetail("error_type", errorType).
				WithDetails(errorDetails)
		}

		// Update job with retry info
		job.ErrorMessage = fmt.Sprintf("%s (will retry)", errorType)
		job.ErrorDetails = errorDetails
		job.Status = resume.JobStatusPending // Reset to pending for retry

		if updateErr := s.jobRepo.Update(ctx, job); updateErr != nil {
			logx.Errorf("Failed to update job for retry: %v", updateErr)
		}

		return resume.ErrJobFailed().
			WithDetail("job_id", job.ID).
			WithDetail("error_type", errorType).
			WithDetail("will_retry", true).
			WithDetail("next_retry_at", nextRetry).
			WithDetails(errorDetails)
	}

	// Max attempts reached - mark as permanently failed
	logx.Errorf("Job permanently failed: JobID=%s, Error=%s, Attempts=%d/%d",
		job.ID, errorType, job.AttemptCount, job.MaxAttempts)

	_ = s.jobRepo.MarkAsFailed(ctx, job.ID, errorType, errorDetails)

	return resume.ErrJobMaxRetriesReached().
		WithDetail("job_id", job.ID).
		WithDetail("error_type", errorType).
		WithDetail("final_attempt", job.AttemptCount).
		WithDetails(errorDetails)
}

// GetJobStatus retrieves the current status of a job
func (s *Service) GetJobStatus(ctx context.Context, jobID kernel.JobID) (*resume.JobStatusResponse, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, resume.ErrJobNotFound().
			WithDetail("job_id", jobID)
	}

	response := &resume.JobStatusResponse{
		JobID:     job.ID,
		TenantID:  job.TenantID,
		Status:    job.Status,
		Progress:  job.ProgressPercentage,
		CreatedAt: job.CreatedAt,
	}

	// Set message based on status
	switch job.Status {
	case resume.JobStatusPending:
		if job.AttemptCount > 0 {
			response.Message = fmt.Sprintf("Job pending retry (attempt %d/%d)", job.AttemptCount, job.MaxAttempts)
		} else {
			response.Message = "Job queued and waiting to be processed"
		}
		if job.NextRetryAt != nil {
			response.NextRetryAt = job.NextRetryAt
		}

	case resume.JobStatusProcessing:
		response.Message = fmt.Sprintf("Processing resume: %v", job.CurrentStep)
		response.CurrentStep = job.CurrentStep
		response.StartedAt = job.StartedAt

	case resume.JobStatusCompleted:
		response.Message = "Resume processed successfully"
		response.ResumeID = job.ResumeID
		response.CompletedAt = job.CompletedAt

	case resume.JobStatusFailed:
		response.Message = job.ErrorMessage
		response.Error = &resume.JobError{
			Message: job.ErrorMessage,
			Details: job.ErrorDetails,
		}
		response.FailedAt = job.FailedAt
		response.AttemptCount = job.AttemptCount
	}

	return response, nil
}

// ListJobsByTenant retrieves all jobs for a tenant
func (s *Service) ListJobsByTenant(ctx context.Context, tenantID kernel.TenantID, pagination kernel.PaginationOptions) (*kernel.Paginated[resume.ResumeProcessingJob], error) {
	jobs, err := s.jobRepo.GetByTenantID(ctx, tenantID, pagination)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeJobNotFound, err).
			WithDetail("tenant_id", tenantID)
	}

	return jobs, nil
}

// CancelJob cancels a pending or processing job
func (s *Service) CancelJob(ctx context.Context, jobID kernel.JobID, tenantID kernel.TenantID) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return resume.ErrJobNotFound().
			WithDetail("job_id", jobID)
	}

	// Verify tenant ownership
	if job.TenantID != tenantID {
		return resume.ErrTenantMismatch().
			WithDetail("job_id", jobID).
			WithDetail("job_tenant_id", job.TenantID).
			WithDetail("requested_tenant_id", tenantID)
	}

	// Can only cancel pending or failed jobs
	if job.Status == resume.JobStatusCompleted {
		return resume.ErrJobAlreadyCompleted().
			WithDetail("job_id", jobID)
	}

	if job.Status == resume.JobStatusProcessing {
		// Note: This won't stop an actively running job, just marks it
		logx.Warnf("Attempting to cancel job that is currently processing: %s", jobID)
	}

	// Mark as failed with cancellation message
	now := time.Now()
	job.Status = resume.JobStatusFailed
	job.FailedAt = &now
	job.ErrorMessage = "Job cancelled by user"
	job.ErrorDetails = map[string]any{
		"cancelled_at": now,
		"tenant_id":    tenantID,
	}

	if err := s.jobRepo.Update(ctx, job); err != nil {
		return resume.ErrJobUpdateFailed().
			WithDetail("job_id", jobID).
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	logx.Infof("Job cancelled: JobID=%s, TenantID=%s", jobID, tenantID)
	return nil
}

// RetryFailedJob manually retries a failed job
func (s *Service) RetryFailedJob(ctx context.Context, jobID kernel.JobID, tenantID kernel.TenantID) (*resume.JobStatusResponse, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, resume.ErrJobNotFound().
			WithDetail("job_id", jobID)
	}

	// Verify tenant ownership
	if job.TenantID != tenantID {
		return nil, resume.ErrTenantMismatch().
			WithDetail("job_id", jobID).
			WithDetail("job_tenant_id", job.TenantID).
			WithDetail("requested_tenant_id", tenantID)
	}

	// Can only retry failed jobs
	if job.Status != resume.JobStatusFailed {
		return nil, resume.ErrInvalidJobStatus().
			WithDetail("job_id", jobID).
			WithDetail("current_status", job.Status).
			WithDetail("required_status", resume.JobStatusFailed)
	}

	// Reset job for retry
	job.Status = resume.JobStatusPending
	job.AttemptCount = 0 // Reset attempt count for manual retry
	job.ErrorMessage = ""
	job.ErrorDetails = nil
	job.FailedAt = nil
	job.NextRetryAt = nil
	job.ProgressPercentage = 0
	job.CurrentStep = nil

	// Update in database
	if err := s.jobRepo.Update(ctx, job); err != nil {
		return nil, resume.ErrJobUpdateFailed().
			WithDetail("job_id", jobID).
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	// Re-enqueue
	if err := s.queue.Enqueue(ctx, jobID, job); err != nil {
		// Mark as failed again
		_ = s.jobRepo.MarkAsFailed(ctx, jobID, "failed to re-enqueue", map[string]any{
			"error": err.Error(),
		})

		return nil, resume.ErrQueueEnqueueFailed().
			WithDetail("job_id", jobID).
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	logx.Infof("Job manually retried: JobID=%s", jobID)

	return &resume.JobStatusResponse{
		JobID:    jobID,
		TenantID: job.TenantID,
		Status:   resume.JobStatusPending,
		Message:  "Job requeued for processing",
		Progress: 0,
	}, nil
}

// GetJobStats returns statistics about jobs for a tenant
func (s *Service) GetJobStats(ctx context.Context, tenantID kernel.TenantID) (*resume.JobStatsResponse, error) {
	// Get all jobs for tenant (without pagination)
	allJobs, err := s.jobRepo.GetByTenantID(ctx, tenantID, kernel.PaginationOptions{
		Page:     1,
		PageSize: 1000, // Get a large number
	})
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeJobNotFound, err).
			WithDetail("tenant_id", tenantID)
	}

	stats := &resume.JobStatsResponse{
		TenantID:        tenantID,
		TotalJobs:       len(allJobs.Items),
		PendingJobs:     0,
		ProcessingJobs:  0,
		CompletedJobs:   0,
		FailedJobs:      0,
		AverageProgress: 0,
	}

	totalProgress := 0
	var oldestPending *time.Time
	var newestCompleted *time.Time

	for _, job := range allJobs.Items {
		switch job.Status {
		case resume.JobStatusPending:
			stats.PendingJobs++
			if oldestPending == nil || job.CreatedAt.Before(*oldestPending) {
				oldestPending = &job.CreatedAt
			}
		case resume.JobStatusProcessing:
			stats.ProcessingJobs++
		case resume.JobStatusCompleted:
			stats.CompletedJobs++
			if job.CompletedAt != nil && (newestCompleted == nil || job.CompletedAt.After(*newestCompleted)) {
				newestCompleted = job.CompletedAt
			}
		case resume.JobStatusFailed:
			stats.FailedJobs++
		}

		totalProgress += job.ProgressPercentage
	}

	if len(allJobs.Items) > 0 {
		stats.AverageProgress = float64(totalProgress) / float64(len(allJobs.Items))
	}

	stats.OldestPendingJob = oldestPending
	stats.LastCompletedJob = newestCompleted

	return stats, nil
}

