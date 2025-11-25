package application

import (
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// CreateApplicationRequest - DTO for creating a new application
type CreateApplicationRequest struct {
	JobID           kernel.JobID           `json:"job_id" validate:"required"`
	CandidateID     kernel.CandidateID     `json:"candidate_id" validate:"required"`
	ResumeSummary   kernel.ResumeSummary   `json:"resume_summary" validate:"required"`
	ResumeEmbedding kernel.ResumeEmbedding `json:"resume_embedding,omitempty"`
	ResumeBucketUrl kernel.BucketURL       `json:"resume_bucket_url" validate:"required"`
	CreatedBy       *kernel.UserID         `json:"created_by,omitempty"`
}

// UpdateApplicationRequest - DTO for updating an existing application
type UpdateApplicationRequest struct {
	ResumeSummary   *kernel.ResumeSummary   `json:"resume_summary,omitempty"`
	ResumeEmbedding *kernel.ResumeEmbedding `json:"resume_embedding,omitempty"`
	ResumeBucketUrl *kernel.BucketURL       `json:"resume_bucket_url,omitempty"`
}

// ApplicationWithDetailsResponse - DTO for returning application with candidate and job details
type ApplicationWithDetailsResponse struct {
	ID              kernel.ApplicationID `json:"id"`
	JobID           kernel.JobID         `json:"job_id"`
	JobTitle        kernel.JobTitle      `json:"job_title"`
	CandidateID     kernel.CandidateID   `json:"candidate_id"`
	CandidateName   string               `json:"candidate_name"`
	CandidateEmail  kernel.Email         `json:"candidate_email"`
	ResumeSummary   kernel.ResumeSummary `json:"resume_summary"`
	ResumeBucketUrl kernel.BucketURL     `json:"resume_bucket_url"`
}

// GetApplicationRequest - DTO for getting an application by ID
type GetApplicationRequest struct {
	ID kernel.ApplicationID `json:"id" validate:"required"`
}

// DeleteApplicationRequest - DTO for deleting an application
type DeleteApplicationRequest struct {
	ID kernel.ApplicationID `json:"id" validate:"required"`
}

// ListApplicationsByJobRequest - DTO for listing applications by job
type ListApplicationsByJobRequest struct {
	JobID      kernel.JobID             `json:"job_id" validate:"required"`
	Pagination kernel.PaginationOptions `json:"pagination"`
}

// ListApplicationsByCandidateRequest - DTO for listing applications by candidate
type ListApplicationsByCandidateRequest struct {
	CandidateID kernel.CandidateID       `json:"candidate_id" validate:"required"`
	Pagination  kernel.PaginationOptions `json:"pagination"`
}

// ListApplicationsRequest - DTO for listing all applications
type ListApplicationsRequest struct {
	Pagination kernel.PaginationOptions `json:"pagination"`
}

// UploadResumeRequest - DTO for uploading resume
type UploadResumeRequest struct {
	ApplicationID kernel.ApplicationID `json:"application_id" validate:"required"`
	FileData      []byte               `json:"-"` // File data, not serialized
	FileName      string               `json:"file_name" validate:"required"`
	FileSize      int64                `json:"file_size" validate:"required,max=10485760"` // 10MB max
	ContentType   string               `json:"content_type" validate:"required"`
}

// Response type aliases for paginated applications
type PaginatedApplicationsResponse = kernel.Paginated[ApplicationResponse]
type PaginatedApplicationsWithDetailsResponse = kernel.Paginated[ApplicationWithDetailsResponse]

// ApplicationResponse - DTO for returning application data
type ApplicationResponse struct {
	ID              kernel.ApplicationID   `json:"id"`
	JobID           kernel.JobID           `json:"job_id"`
	CandidateID     kernel.CandidateID     `json:"candidate_id"`
	ResumeSummary   kernel.ResumeSummary   `json:"resume_summary"`
	ResumeEmbedding kernel.ResumeEmbedding `json:"resume_embedding,omitempty"`
	ResumeBucketUrl kernel.BucketURL       `json:"resume_bucket_url"`
	Status          ApplicationStatus      `json:"status"`
	ReviewerID      *kernel.UserID         `json:"reviewer_id,omitempty"`
	SubmittedBy     *kernel.UserID         `json:"submitted_by"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// UploadResumeResponse - Response after uploading resume
type UploadResumeResponse struct {
	ApplicationID kernel.ApplicationID `json:"application_id"`
	BucketURL     kernel.BucketURL     `json:"bucket_url"`
	FileName      string               `json:"file_name"`
	FileSize      int64                `json:"file_size"`
	UploadedAt    time.Time            `json:"uploaded_at"`
	UploadedBy    kernel.UserID        `json:"uploaded_by"`
}

// AssignReviewerRequest - Request to assign reviewer
type AssignReviewerRequest struct {
	ReviewerID kernel.UserID `json:"reviewer_id" validate:"required"`
}

// UpdateStatusRequest - Request to update application status
type UpdateStatusRequest struct {
	Status ApplicationStatus `json:"status" validate:"required"`
	Reason string            `json:"reason,omitempty"`
}

// ApplicationStatsResponse - Statistics for an application
type ApplicationStatsResponse struct {
	ApplicationID         kernel.ApplicationID `json:"application_id"`
	Status                ApplicationStatus    `json:"status"`
	IsArchived            bool                 `json:"is_archived"`
	HasResume             bool                 `json:"has_resume"`
	HasReviewer           bool                 `json:"has_reviewer"`
	DaysSinceSubmission   int                  `json:"days_since_submission"`
	DaysSinceLastUpdate   int                  `json:"days_since_last_update"`
	DaysSinceStatusChange *int                 `json:"days_since_status_change,omitempty"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
}

// BulkApplicationOperationResponse - Result of bulk operations
type BulkApplicationOperationResponse struct {
	Successful []kernel.ApplicationID          `json:"successful"`
	Failed     map[kernel.ApplicationID]string `json:"failed"`
	Total      int                             `json:"total"`
}

// BulkArchiveApplicationsRequest - Request to archive multiple applications
type BulkArchiveApplicationsRequest struct {
	ApplicationIDs []kernel.ApplicationID `json:"application_ids" validate:"required,min=1"`
}

// BulkUpdateStatusRequest - Request to update status for multiple applications
type BulkUpdateStatusRequest struct {
	ApplicationIDs []kernel.ApplicationID `json:"application_ids" validate:"required,min=1"`
	Status         ApplicationStatus      `json:"status" validate:"required"`
}

// WithdrawApplicationRequest - Request to withdraw an application
type WithdrawApplicationRequest struct {
	Reason string `json:"reason,omitempty"`
}
