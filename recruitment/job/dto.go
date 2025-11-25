package job

import (
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// CreateJobRequest - DTO for creating a new job
type CreateJobRequest struct {
	Title               kernel.JobTitle         `json:"job_title" validate:"required"`
	Description         kernel.JobDescription   `json:"job_description" validate:"required"`
	Position            kernel.JobPosition      `json:"job_position" validate:"required"`
	GeneralRequirements []kernel.JobRequirement `json:"general_requirements,omitempty"`
	Benefits            []kernel.JobBenefit     `json:"benefits,omitempty"`
	PostedBy            kernel.UserID           `json:"posted_by" validate:"required"`
}

// UpdateJobRequest - DTO for updating an existing job
type UpdateJobRequest struct {
	Title               *kernel.JobTitle         `json:"job_title,omitempty"`
	Description         *kernel.JobDescription   `json:"job_description,omitempty"`
	Position            *kernel.JobPosition      `json:"job_position,omitempty"`
	GeneralRequirements *[]kernel.JobRequirement `json:"general_requirements,omitempty"`
	Benefits            *[]kernel.JobBenefit     `json:"benefits,omitempty"`
}

// GetJobRequest - DTO for getting a job by ID
type GetJobRequest struct {
	ID kernel.JobID `json:"id" validate:"required"`
}

// DeleteJobRequest - DTO for deleting a job
type DeleteJobRequest struct {
	ID kernel.JobID `json:"id" validate:"required"`
}

// ListJobsRequest - DTO for listing all jobs
type ListJobsRequest struct {
	Pagination kernel.PaginationOptions `json:"pagination"`
}

// ListJobsByUserRequest - DTO for listing jobs by user
type ListJobsByUserRequest struct {
	PostedBy   kernel.UserID            `json:"posted_by" validate:"required"`
	Pagination kernel.PaginationOptions `json:"pagination"`
}

// SearchJobsRequest - DTO for searching jobs
type SearchJobsRequest struct {
	Query      string                   `json:"query,omitempty"`
	Title      string                   `json:"title,omitempty"`
	Position   string                   `json:"position,omitempty"`
	PostedBy   string                   `json:"posted_by,omitempty"`
	Pagination kernel.PaginationOptions `json:"pagination"`
}

// Response type alias for paginated jobs
type PaginatedJobsResponse = kernel.Paginated[JobResponse]

// JobResponse - DTO for returning job data
type JobResponse struct {
	ID                  kernel.JobID            `json:"id"`
	Title               kernel.JobTitle         `json:"job_title"`
	Description         kernel.JobDescription   `json:"job_description"`
	Position            kernel.JobPosition      `json:"job_position"`
	GeneralRequirements []kernel.JobRequirement `json:"general_requirements"`
	Benefits            []kernel.JobBenefit     `json:"benefits"`
	PostedBy            kernel.UserID           `json:"posted_by"`
	Status              JobStatus               `json:"status"`
	PublishedAt         *time.Time              `json:"published_at,omitempty"`
	ArchivedAt          *time.Time              `json:"archived_at,omitempty"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
}

// JobStatsResponse - Statistics for a job
type JobStatsResponse struct {
	JobID              kernel.JobID    `json:"job_id"`
	Title              kernel.JobTitle `json:"title"`
	Status             JobStatus       `json:"status"`
	TotalApplications  int64           `json:"total_applications"`
	IsPublished        bool            `json:"is_published"`
	IsArchived         bool            `json:"is_archived"`
	DaysSincePublished *int            `json:"days_since_published,omitempty"`
	DaysSinceArchived  *int            `json:"days_since_archived,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

// BulkJobOperationResponse - Result of bulk operations
type BulkJobOperationResponse struct {
	Successful []kernel.JobID          `json:"successful"`
	Failed     map[kernel.JobID]string `json:"failed"`
	Total      int                     `json:"total"`
}
