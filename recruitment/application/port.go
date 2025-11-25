package application

import (
	"context"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

type Repository interface {
	// Create creates a new application
	Create(ctx context.Context, application *Application) error

	// Update updates an existing application
	Update(ctx context.Context, id kernel.ApplicationID, application *Application) error

	// GetByID retrieves an application by ID
	GetByID(ctx context.Context, id kernel.ApplicationID) (*Application, error)

	// GetWithDetails retrieves an application with candidate and job details
	GetWithDetails(ctx context.Context, id kernel.ApplicationID) (*ApplicationWithDetailsResponse, error)

	// Delete deletes an application by ID
	Delete(ctx context.Context, id kernel.ApplicationID) error

	// List retrieves all applications with pagination
	List(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[Application], error)

	// ListByJobID retrieves applications for a specific job
	ListByJobID(ctx context.Context, jobID kernel.JobID, pagination kernel.PaginationOptions) (*kernel.Paginated[Application], error)

	// ListByCandidateID retrieves applications for a specific candidate
	ListByCandidateID(ctx context.Context, candidateID kernel.CandidateID, pagination kernel.PaginationOptions) (*kernel.Paginated[Application], error)

	// ListWithDetailsByJobID retrieves applications with details for a specific job
	ListWithDetailsByJobID(ctx context.Context, jobID kernel.JobID, pagination kernel.PaginationOptions) (*kernel.Paginated[ApplicationWithDetailsResponse], error)

	// ListWithDetailsByCandidateID retrieves applications with details for a specific candidate
	ListWithDetailsByCandidateID(ctx context.Context, candidateID kernel.CandidateID, pagination kernel.PaginationOptions) (*kernel.Paginated[ApplicationWithDetailsResponse], error)

	// Exists checks if an application exists by ID
	Exists(ctx context.Context, id kernel.ApplicationID) (bool, error)

	// ExistsByJobAndCandidate checks if an application exists for a job and candidate
	ExistsByJobAndCandidate(ctx context.Context, jobID kernel.JobID, candidateID kernel.CandidateID) (bool, error)

	// UpdateResumeBucketUrl updates the resume bucket URL
	UpdateResumeBucketUrl(ctx context.Context, id kernel.ApplicationID, url kernel.BucketURL) error

	// CountByJobID counts applications for a specific job
	CountByJobID(ctx context.Context, jobID kernel.JobID) (int64, error)

	// CountByCandidateID counts applications for a specific candidate
	CountByCandidateID(ctx context.Context, candidateID kernel.CandidateID) (int64, error)

	// AssignReviewer assigns a reviewer to an application
	AssignReviewer(ctx context.Context, id kernel.ApplicationID, reviewerID kernel.UserID) error

	// GetApplicationsByReviewer retrieves applications assigned to a reviewer
	GetApplicationsByReviewer(ctx context.Context, reviewerID kernel.UserID, pagination kernel.PaginationOptions) (*kernel.Paginated[Application], error)

	// Archive archives an application
	Archive(ctx context.Context, id kernel.ApplicationID) error

	// Unarchive unarchives an application
	Unarchive(ctx context.Context, id kernel.ApplicationID) error
}
