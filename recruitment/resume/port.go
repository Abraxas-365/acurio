package resume

import (
	"context"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

type Repository interface {
	// Create creates a new resume
	Create(ctx context.Context, resume *Resume) error

	// Update updates an existing resume
	Update(ctx context.Context, id kernel.ResumeID, resume *Resume) error

	// GetByID retrieves a resume by ID
	GetByID(ctx context.Context, id kernel.ResumeID) (*Resume, error)

	// GetByTenantID retrieves the default resume for a tenant (backward compatibility)
	GetByTenantID(ctx context.Context, tenantID kernel.TenantID) (*Resume, error)

	// ListByTenantID retrieves all resumes for a tenant
	ListByTenantID(ctx context.Context, tenantID kernel.TenantID) ([]*Resume, error)

	// GetActiveByTenantID retrieves all active resumes for a tenant
	GetActiveByTenantID(ctx context.Context, tenantID kernel.TenantID) ([]*Resume, error)

	// GetDefaultByTenantID retrieves the default resume for a tenant
	GetDefaultByTenantID(ctx context.Context, tenantID kernel.TenantID) (*Resume, error)

	// SetDefault sets a resume as the default for a tenant (unsets others)
	SetDefault(ctx context.Context, id kernel.ResumeID, tenantID kernel.TenantID) error

	// ToggleActive activates or deactivates a resume
	ToggleActive(ctx context.Context, id kernel.ResumeID, isActive bool) error

	// Delete deletes a resume
	Delete(ctx context.Context, id kernel.ResumeID) error

	// Exists checks if any resume exists for a tenant
	Exists(ctx context.Context, tenantID kernel.TenantID) (bool, error)

	// CountByTenantID counts resumes for a tenant
	CountByTenantID(ctx context.Context, tenantID kernel.TenantID) (int64, error)

	// SemanticSearch performs vector similarity search
	SemanticSearch(ctx context.Context, req SearchResumesRequest) ([]ResumeMatchResult, error)

	// UpdateEmbeddings updates only the embeddings for a resume
	UpdateEmbeddings(ctx context.Context, id kernel.ResumeID, embeddings ResumeEmbeddings) error

	// List retrieves all resumes with pagination
	List(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[Resume], error)

	// ListByTenantIDWithPagination retrieves resumes for a tenant with pagination
	ListByTenantIDWithPagination(ctx context.Context, tenantID kernel.TenantID, pagination kernel.PaginationOptions) (*kernel.Paginated[Resume], error)

	// SearchByTenant performs semantic search within a specific tenant
	SearchByTenant(ctx context.Context, tenantID kernel.TenantID, req SearchResumesRequest) ([]ResumeMatchResult, error)
}

type JobRepository interface {
	Create(ctx context.Context, job *ResumeProcessingJob) error
	Update(ctx context.Context, job *ResumeProcessingJob) error
	GetByID(ctx context.Context, jobID kernel.JobID) (*ResumeProcessingJob, error)
	GetByTenantID(ctx context.Context, tenantID kernel.TenantID, pagination kernel.PaginationOptions) (*kernel.Paginated[ResumeProcessingJob], error)

	// For retry mechanism
	GetFailedJobsForRetry(ctx context.Context, limit int) ([]*ResumeProcessingJob, error)

	// Update status helpers
	MarkAsProcessing(ctx context.Context, jobID kernel.JobID) error
	MarkAsCompleted(ctx context.Context, jobID kernel.JobID, resumeID kernel.ResumeID) error
	MarkAsFailed(ctx context.Context, jobID kernel.JobID, errorMsg string, errorDetails map[string]any) error
	UpdateProgress(ctx context.Context, jobID kernel.JobID, step ProcessingStep, percentage int) error
}

// Queue defines the interface for job queue operations
type JobQueue interface {
	// Enqueue adds a job to the queue
	Enqueue(ctx context.Context, jobID kernel.JobID, payload any) error

	// Dequeue gets a job from the queue (blocking with timeout)
	Dequeue(ctx context.Context, timeout time.Duration) ([]byte, error)

	// EnqueueDelayed schedules a job for later processing (for retries)
	EnqueueDelayed(ctx context.Context, jobID kernel.JobID, payload any, delay time.Duration) error

	// MoveDelayedToReady moves delayed jobs that are ready to the main queue
	MoveDelayedToReady(ctx context.Context) (int, error)

	// GetQueueSize returns the number of jobs in the queue
	GetQueueSize(ctx context.Context) (int64, error)

	// GetDelayedQueueSize returns the number of delayed jobs
	GetDelayedQueueSize(ctx context.Context) (int64, error)

	// Clear removes all jobs from the queue (use with caution)
	Clear(ctx context.Context) error
}
