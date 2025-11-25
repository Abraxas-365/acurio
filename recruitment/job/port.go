package job

import (
	"context"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

type Repository interface {
	// Create creates a new job
	Create(ctx context.Context, job *Job) error

	// Update updates an existing job
	Update(ctx context.Context, id kernel.JobID, job *Job) error

	// GetByID retrieves a job by ID
	GetByID(ctx context.Context, id kernel.JobID) (*Job, error)

	// Delete deletes a job by ID
	Delete(ctx context.Context, id kernel.JobID) error

	// List retrieves all jobs with pagination
	List(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[Job], error)

	// ListByUserID retrieves jobs posted by a specific user
	ListByUserID(ctx context.Context, userID kernel.UserID, pagination kernel.PaginationOptions) (*kernel.Paginated[Job], error)

	// Search searches jobs by various criteria
	Search(ctx context.Context, req SearchJobsRequest) (*kernel.Paginated[Job], error)

	// GetByTitle retrieves jobs by title (exact match or partial)
	GetByTitle(ctx context.Context, title kernel.JobTitle) ([]*Job, error)

	// Exists checks if a job exists by ID
	Exists(ctx context.Context, id kernel.JobID) (bool, error)

	// CountByUserID counts the number of jobs posted by a user
	CountByUserID(ctx context.Context, userID kernel.UserID) (int64, error)

	// Archive archives a job (soft delete alternative)
	Archive(ctx context.Context, id kernel.JobID) error

	// Unarchive unarchives a job
	Unarchive(ctx context.Context, id kernel.JobID) error

	// ListArchived retrieves archived jobs with pagination
	ListArchived(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[Job], error)

	// Publish marks a job as published/active
	Publish(ctx context.Context, id kernel.JobID) error

	// Unpublish marks a job as unpublished/draft
	Unpublish(ctx context.Context, id kernel.JobID) error

	// ListPublished retrieves only published jobs
	ListPublished(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[Job], error)

	CountApplications(ctx context.Context, jobID kernel.JobID) (int64, error)
}
