package resume

import (
	"github.com/Abraxas-365/relay/pkg/kernel"
	"time"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

type ProcessingStep string

const (
	StepUploading ProcessingStep = "uploading"
	StepParsing   ProcessingStep = "parsing"
	StepEmbedding ProcessingStep = "embedding"
	StepSaving    ProcessingStep = "saving"
)

type ResumeProcessingJob struct {
	ID       kernel.JobID     `db:"id" json:"id"`
	TenantID kernel.TenantID  `db:"tenant_id" json:"tenant_id"`
	ResumeID *kernel.ResumeID `db:"resume_id" json:"resume_id,omitempty"`

	Status   JobStatus `db:"status" json:"status"`
	FilePath string    `db:"file_path" json:"file_path"`
	FileName string    `db:"file_name" json:"file_name"`
	FileType string    `db:"file_type" json:"file_type"`
	Title    string    `db:"title" json:"title"`

	AttemptCount int `db:"attempt_count" json:"attempt_count"`
	MaxAttempts  int `db:"max_attempts" json:"max_attempts"`

	ErrorMessage string         `db:"error_message" json:"error_message,omitempty"`
	ErrorDetails map[string]any `db:"error_details" json:"error_details,omitempty"`

	CurrentStep        *ProcessingStep `db:"current_step" json:"current_step,omitempty"`
	ProgressPercentage int             `db:"progress_percentage" json:"progress_percentage"`

	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	StartedAt   *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	FailedAt    *time.Time `db:"failed_at" json:"failed_at,omitempty"`
	NextRetryAt *time.Time `db:"next_retry_at" json:"next_retry_at,omitempty"`

	RequestPayload ParseResumeRequest `db:"request_payload" json:"request_payload"`
}

// JobStatusResponse - Response for job status queries
type JobStatusResponse struct {
	JobID       kernel.JobID     `json:"job_id"`
	TenantID    kernel.TenantID  `json:"tenant_id"`
	Status      JobStatus        `json:"status"`
	Message     string           `json:"message"`
	Progress    int              `json:"progress"`
	CurrentStep *ProcessingStep  `json:"current_step,omitempty"`
	ResumeID    *kernel.ResumeID `json:"resume_id,omitempty"`
	Error       *JobError        `json:"error,omitempty"`

	AttemptCount int        `json:"attempt_count,omitempty"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`

	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
}

// JobError - Error details for failed jobs
type JobError struct {
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// JobStatsResponse - Statistics about jobs for a tenant
type JobStatsResponse struct {
	TenantID         kernel.TenantID `json:"tenant_id"`
	TotalJobs        int             `json:"total_jobs"`
	PendingJobs      int             `json:"pending_jobs"`
	ProcessingJobs   int             `json:"processing_jobs"`
	CompletedJobs    int             `json:"completed_jobs"`
	FailedJobs       int             `json:"failed_jobs"`
	AverageProgress  float64         `json:"average_progress"`
	OldestPendingJob *time.Time      `json:"oldest_pending_job,omitempty"`
	LastCompletedJob *time.Time      `json:"last_completed_job,omitempty"`
}
