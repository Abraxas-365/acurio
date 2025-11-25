package candidate

import (
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// CreateCandidateRequest - DTO for creating a new candidate
type CreateCandidateRequest struct {
	Email     kernel.Email     `json:"email" validate:"required,email"`
	Phone     kernel.Phone     `json:"phone" validate:"required"`
	FirstName kernel.FirstName `json:"first_name" validate:"required"`
	LastName  kernel.LastName  `json:"last_name" validate:"required"`
	DNI       kernel.DNI       `json:"dni" validate:"required"`
}

// UpdateCandidateRequest - DTO for updating an existing candidate
type UpdateCandidateRequest struct {
	Email     *kernel.Email     `json:"email,omitempty" validate:"omitempty,email"`
	Phone     *kernel.Phone     `json:"phone,omitempty"`
	FirstName *kernel.FirstName `json:"first_name,omitempty"`
	LastName  *kernel.LastName  `json:"last_name,omitempty"`
	DNI       *kernel.DNI       `json:"dni,omitempty"`
}

// GetCandidateRequest - DTO for getting a candidate by ID
type GetCandidateRequest struct {
	ID kernel.CandidateID `json:"id" validate:"required"`
}

// DeleteCandidateRequest - DTO for deleting a candidate
type DeleteCandidateRequest struct {
	ID kernel.CandidateID `json:"id" validate:"required"`
}

// SearchCandidatesRequest - DTO for searching candidates
type SearchCandidatesRequest struct {
	Query      string                   `json:"query,omitempty"`
	Email      string                   `json:"email,omitempty"`
	Phone      string                   `json:"phone,omitempty"`
	DNIType    string                   `json:"dni_type,omitempty"`
	Pagination kernel.PaginationOptions `json:"pagination"`
}

// ListCandidatesRequest - DTO for listing all candidates
type ListCandidatesRequest struct {
	Pagination kernel.PaginationOptions `json:"pagination"`
}

// Response type alias for paginated candidates
type PaginatedCandidatesResponse = kernel.Paginated[CandidateResponse]

// CandidateResponse - DTO for returning candidate data
type CandidateResponse struct {
	ID        kernel.CandidateID `json:"id"`
	Email     kernel.Email       `json:"email"`
	Phone     kernel.Phone       `json:"phone"`
	FirstName kernel.FirstName   `json:"first_name"`
	LastName  kernel.LastName    `json:"last_name"`
	DNI       kernel.DNI         `json:"dni"`
	Status    CandidateStatus    `json:"status"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// CandidateStatsResponse - Statistics for a candidate
type CandidateStatsResponse struct {
	CandidateID           kernel.CandidateID `json:"candidate_id"`
	FullName              string             `json:"full_name"`
	Email                 kernel.Email       `json:"email"`
	Status                CandidateStatus    `json:"status"`
	TotalApplications     int64              `json:"total_applications"`
	IsArchived            bool               `json:"is_archived"`
	DaysSinceRegistration int                `json:"days_since_registration"`
	DaysSinceLastUpdate   int                `json:"days_since_last_update"`
	DaysSinceArchived     *int               `json:"days_since_archived,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
}

// BulkCandidateOperationResponse - Result of bulk operations
type BulkCandidateOperationResponse struct {
	Successful []kernel.CandidateID          `json:"successful"`
	Failed     map[kernel.CandidateID]string `json:"failed"`
	Total      int                           `json:"total"`
}

// ExportCandidatesRequest - Request for exporting candidates
type ExportCandidatesRequest struct {
	CandidateIDs    []kernel.CandidateID `json:"candidate_ids,omitempty"`                         // Specific candidates, or empty for all
	Format          string               `json:"format" validate:"required,oneof=csv json excel"` // Export format
	IncludeArchived bool                 `json:"include_archived"`                                // Include archived candidates
}

// CandidateExportData - Single candidate data for export
type CandidateExportData struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	DNIType   string    `json:"dni_type"`
	DNINumber string    `json:"dni_number"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ExportCandidatesResponse - Response for export operation
type ExportCandidatesResponse struct {
	Data       []CandidateExportData `json:"data"`
	TotalCount int                   `json:"total_count"`
	ExportedAt time.Time             `json:"exported_at"`
	ExportedBy kernel.UserID         `json:"exported_by"`
}

// ArchiveCandidateRequest - Request to archive a candidate
type ArchiveCandidateRequest struct {
	Reason string `json:"reason,omitempty"`
}

// BulkArchiveCandidatesRequest - Request to archive multiple candidates
type BulkArchiveCandidatesRequest struct {
	CandidateIDs []kernel.CandidateID `json:"candidate_ids" validate:"required,min=1"`
	Reason       string               `json:"reason,omitempty"`
}

// BulkDeleteCandidatesRequest - Request to delete multiple candidates
type BulkDeleteCandidatesRequest struct {
	CandidateIDs []kernel.CandidateID `json:"candidate_ids" validate:"required,min=1"`
}

// CandidateDetailsResponse - Extended candidate information with applications
type CandidateDetailsResponse struct {
	Candidate          CandidateResponse `json:"candidate"`
	ApplicationCount   int64             `json:"application_count"`
	LastApplicationAt  *time.Time        `json:"last_application_at,omitempty"`
	ActiveApplications int64             `json:"active_applications"`
}

type UpdateProfileRequest struct {
	FirstName   *string `json:"first_name,omitempty" validate:"omitempty,min=2"`
	LastName    *string `json:"last_name,omitempty" validate:"omitempty,min=2"`
	Phone       *string `json:"phone,omitempty"`
	LinkedInURL *string `json:"linkedin_url,omitempty"`
}

type UploadResumeRequest struct {
	FileName string `json:"file_name" validate:"required"`
	FileSize int64  `json:"file_size" validate:"required"`
	FileType string `json:"file_type" validate:"required"`
}
