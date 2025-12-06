package resume

import (
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Request DTOs
// ============================================================================

// ParseResumeRequest - Request to parse and create a resume
type ParseResumeRequest struct {
	TenantID  kernel.TenantID `json:"tenant_id" validate:"required"`
	FilePath  string          `json:"file_path" validate:"required"`
	FileName  string          `json:"file_name" validate:"required"`
	FileType  string          `json:"file_type" validate:"required,oneof=pdf jpg jpeg png"`
	Title     string          `json:"title" validate:"required"` // Resume title/name
	IsActive  bool            `json:"is_active"`                 // Set as active
	IsDefault bool            `json:"is_default"`                // Set as default
}

// CreateResumeRequest - Manual resume creation (rare)
type CreateResumeRequest struct {
	TenantID            kernel.TenantID       `json:"tenant_id" validate:"required"`
	Title               string                `json:"title" validate:"required"`
	PersonalInfo        PersonalInfo          `json:"personal_info" validate:"required"`
	WorkExperience      []WorkExperience      `json:"work_experience,omitempty"`
	Education           []Education           `json:"education,omitempty"`
	Skills              Skills                `json:"skills,omitempty"`
	Languages           []Language            `json:"languages,omitempty"`
	Certifications      []Certification       `json:"certifications,omitempty"`
	Projects            []Project             `json:"projects,omitempty"`
	Achievements        []string              `json:"achievements,omitempty"`
	VolunteerWork       []VolunteerExperience `json:"volunteer_work,omitempty"`
	ProfessionalSummary string                `json:"professional_summary,omitempty"`
	PersonalStatement   *PersonalStatement    `json:"personal_statement,omitempty"`
	IsActive            bool                  `json:"is_active"`
	IsDefault           bool                  `json:"is_default"`
}

// UpdateResumeRequest - Update resume information
type UpdateResumeRequest struct {
	Title               *string                `json:"title,omitempty"`
	PersonalInfo        *PersonalInfo          `json:"personal_info,omitempty"`
	WorkExperience      *[]WorkExperience      `json:"work_experience,omitempty"`
	Education           *[]Education           `json:"education,omitempty"`
	Skills              *Skills                `json:"skills,omitempty"`
	Languages           *[]Language            `json:"languages,omitempty"`
	Certifications      *[]Certification       `json:"certifications,omitempty"`
	Projects            *[]Project             `json:"projects,omitempty"`
	Achievements        *[]string              `json:"achievements,omitempty"`
	VolunteerWork       *[]VolunteerExperience `json:"volunteer_work,omitempty"`
	ProfessionalSummary *string                `json:"professional_summary,omitempty"`
	PersonalStatement   *PersonalStatement     `json:"personal_statement,omitempty"`
}

// AddPersonalStatementRequest - Add/update personal statement
type AddPersonalStatementRequest struct {
	WhyThisCompany string `json:"why_this_company,omitempty" validate:"max=1000"`
	WhyThisRole    string `json:"why_this_role,omitempty" validate:"max=1000"`
	CareerGoals    string `json:"career_goals,omitempty" validate:"max=1000"`
	UniqueValue    string `json:"unique_value,omitempty" validate:"max=1000"`
	Essay          string `json:"essay,omitempty" validate:"max=2000"`
}

// SetDefaultResumeRequest - Set a resume as default
type SetDefaultResumeRequest struct {
	ResumeID kernel.ResumeID `json:"resume_id" validate:"required"`
}

// ToggleActiveRequest - Activate/deactivate a resume
type ToggleActiveRequest struct {
	IsActive bool `json:"is_active"`
}

// ListResumesRequest - List resumes for a tenant
type ListResumesRequest struct {
	TenantID   kernel.TenantID          `json:"tenant_id" validate:"required"`
	OnlyActive bool                     `json:"only_active"`
	Pagination kernel.PaginationOptions `json:"pagination"`
}

// SearchResumesRequest - Semantic search request
type SearchResumesRequest struct {
	Query              string                   `json:"query" validate:"required"`
	TopK               int                      `json:"top_k" validate:"min=1,max=100"`
	MinYearsExperience *float64                 `json:"min_years_experience,omitempty"`
	MaxYearsExperience *float64                 `json:"max_years_experience,omitempty"`
	RequiredSkills     []string                 `json:"required_skills,omitempty"`
	PreferredSkills    []string                 `json:"preferred_skills,omitempty"`
	Locations          []string                 `json:"locations,omitempty"`
	EducationLevel     *string                  `json:"education_level,omitempty"`
	Languages          []string                 `json:"languages,omitempty"`
	Industries         []string                 `json:"industries,omitempty"`
	OnlyActive         bool                     `json:"only_active"`
	TenantID           *kernel.TenantID         `json:"tenant_id,omitempty"` // Optional: search within specific tenant
	Pagination         kernel.PaginationOptions `json:"pagination"`
}

// GetResumeRequest - Get resume by ID
type GetResumeRequest struct {
	ResumeID kernel.ResumeID `json:"resume_id" validate:"required"`
}

// DeleteResumeRequest - Delete resume by ID
type DeleteResumeRequest struct {
	ResumeID kernel.ResumeID `json:"resume_id" validate:"required"`
}

// UpdateEmbeddingsRequest - Update embeddings for a resume
type UpdateEmbeddingsRequest struct {
	ResumeID   kernel.ResumeID  `json:"resume_id" validate:"required"`
	Embeddings ResumeEmbeddings `json:"embeddings" validate:"required"`
}

// BulkUpdateEmbeddingsRequest - Update embeddings for multiple resumes
type BulkUpdateEmbeddingsRequest struct {
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	Force    bool            `json:"force"` // Force regeneration even if embeddings exist
}

// ============================================================================
// Response DTOs
// ============================================================================

// ResumeResponse - Full resume response
type ResumeResponse struct {
	ID                  kernel.ResumeID       `json:"id"`
	TenantID            kernel.TenantID       `json:"tenant_id"`
	Title               string                `json:"title"`
	IsActive            bool                  `json:"is_active"`
	IsDefault           bool                  `json:"is_default"`
	Version             int                   `json:"version"`
	PersonalInfo        PersonalInfo          `json:"personal_info"`
	WorkExperience      []WorkExperience      `json:"work_experience"`
	Education           []Education           `json:"education"`
	Skills              Skills                `json:"skills"`
	Languages           []Language            `json:"languages"`
	Certifications      []Certification       `json:"certifications"`
	Projects            []Project             `json:"projects,omitempty"`
	Achievements        []string              `json:"achievements,omitempty"`
	VolunteerWork       []VolunteerExperience `json:"volunteer_work,omitempty"`
	ProfessionalSummary string                `json:"professional_summary,omitempty"`
	PersonalStatement   PersonalStatement     `json:"personal_statement,omitempty"`
	FileURL             string                `json:"file_url"`
	FileName            string                `json:"file_name"`
	FileType            string                `json:"file_type"`
	TotalYearsExp       float64               `json:"total_years_experience"`
	HasEmbeddings       bool                  `json:"has_embeddings"`
	ParsedAt            time.Time             `json:"parsed_at"`
	LastUpdatedAt       time.Time             `json:"last_updated_at"`
	CreatedAt           time.Time             `json:"created_at"`
}

// ResumeSummaryResponse - Lightweight resume summary
type ResumeSummaryResponse struct {
	ID                   kernel.ResumeID `json:"id"`
	TenantID             kernel.TenantID `json:"tenant_id"`
	Title                string          `json:"title"`
	IsActive             bool            `json:"is_active"`
	IsDefault            bool            `json:"is_default"`
	Version              int             `json:"version"`
	FullName             string          `json:"full_name"`
	Email                string          `json:"email"`
	Phone                string          `json:"phone,omitempty"`
	Location             string          `json:"location,omitempty"`
	LatestPosition       string          `json:"latest_position,omitempty"`
	LatestCompany        string          `json:"latest_company,omitempty"`
	TotalYearsExp        float64         `json:"total_years_experience"`
	TopSkills            []string        `json:"top_skills,omitempty"`
	HighestEducation     string          `json:"highest_education,omitempty"`
	HasPersonalStatement bool            `json:"has_personal_statement"`
	ParsedAt             time.Time       `json:"parsed_at"`
	LastUpdatedAt        time.Time       `json:"last_updated_at"`
}

// ResumeMatchResult - Single resume match with similarity score
type ResumeMatchResult struct {
	Resume           ResumeSummaryResponse `json:"resume"`
	SimilarityScore  float64               `json:"similarity_score"`
	MatchExplanation string                `json:"match_explanation"`
	MatchedSkills    []string              `json:"matched_skills"`
	MatchedSections  MatchedSections       `json:"matched_sections"`
}

// MatchedSections - Shows which sections matched in search
type MatchedSections struct {
	ExperienceScore        float64 `json:"experience_score"`
	EducationScore         float64 `json:"education_score"`
	SkillsScore            float64 `json:"skills_score"`
	PersonalStatementScore float64 `json:"personal_statement_score,omitempty"`
}

// SearchResumesResponse - Results from semantic search with pagination
type SearchResumesResponse struct {
	Results       kernel.Paginated[ResumeMatchResult] `json:"results"`
	SearchQuery   string                              `json:"search_query"`
	Filters       SearchFilters                       `json:"filters,omitempty"`
	ExecutionTime string                              `json:"execution_time"`
}

// SearchFilters - Applied filters in search
type SearchFilters struct {
	MinYearsExperience *float64 `json:"min_years_experience,omitempty"`
	MaxYearsExperience *float64 `json:"max_years_experience,omitempty"`
	RequiredSkills     []string `json:"required_skills,omitempty"`
	PreferredSkills    []string `json:"preferred_skills,omitempty"`
	Locations          []string `json:"locations,omitempty"`
	EducationLevel     *string  `json:"education_level,omitempty"`
	Languages          []string `json:"languages,omitempty"`
	Industries         []string `json:"industries,omitempty"`
	OnlyActive         bool     `json:"only_active"`
}

// ListResumesResponse - List of tenant's resumes with pagination
type ListResumesResponse struct {
	Resumes       kernel.Paginated[ResumeSummaryResponse] `json:"resumes"`
	ActiveCount   int                                     `json:"active_count"`
	InactiveCount int                                     `json:"inactive_count"`
	DefaultResume *kernel.ResumeID                        `json:"default_resume,omitempty"`
}

// ResumeStatsResponse - Statistics about resumes
type ResumeStatsResponse struct {
	TenantID              kernel.TenantID `json:"tenant_id"`
	TotalResumes          int             `json:"total_resumes"`
	ActiveResumes         int             `json:"active_resumes"`
	InactiveResumes       int             `json:"inactive_resumes"`
	ResumesWithEmbeddings int             `json:"resumes_with_embeddings"`
	AverageVersion        float64         `json:"average_version"`
	LastUpdated           *time.Time      `json:"last_updated,omitempty"`
}

// EmbeddingsStatusResponse - Status of embeddings generation
type EmbeddingsStatusResponse struct {
	ResumeID      kernel.ResumeID `json:"resume_id"`
	HasEmbeddings bool            `json:"has_embeddings"`
	ModelUsed     string          `json:"model_used,omitempty"`
	EmbeddingDim  int             `json:"embedding_dim,omitempty"`
	GeneratedAt   *time.Time      `json:"generated_at,omitempty"`
}

// BulkOperationResponse - Response for bulk operations
type BulkOperationResponse struct {
	TotalProcessed int      `json:"total_processed"`
	SuccessCount   int      `json:"success_count"`
	FailureCount   int      `json:"failure_count"`
	Errors         []string `json:"errors,omitempty"`
	ExecutionTime  string   `json:"execution_time"`
}

// ============================================================================
// Mapper Functions
// ============================================================================

// ToResumeResponse converts a Resume domain model to ResumeResponse DTO
func ToResumeResponse(r *Resume) *ResumeResponse {
	return &ResumeResponse{
		ID:                  r.ID,
		TenantID:            r.TenantID,
		Title:               r.Title,
		IsActive:            r.IsActive,
		IsDefault:           r.IsDefault,
		Version:             r.Version,
		PersonalInfo:        r.PersonalInfo,
		WorkExperience:      r.WorkExperience,
		Education:           r.Education,
		Skills:              r.Skills,
		Languages:           r.Languages,
		Certifications:      r.Certifications,
		Projects:            r.Projects,
		Achievements:        r.Achievements,
		VolunteerWork:       r.VolunteerWork,
		ProfessionalSummary: r.ProfessionalSummary,
		PersonalStatement:   r.PersonalStatement,
		FileURL:             r.FileURL,
		FileName:            r.FileName,
		FileType:            r.FileType,
		TotalYearsExp:       r.TotalYearsOfExperience(),
		HasEmbeddings:       r.HasEmbeddings(),
		ParsedAt:            r.ParsedAt,
		LastUpdatedAt:       r.LastUpdatedAt,
		CreatedAt:           r.CreatedAt,
	}
}

// ToResumeSummaryResponse converts a Resume to ResumeSummaryResponse
func ToResumeSummaryResponse(r *Resume) *ResumeSummaryResponse {
	summary := &ResumeSummaryResponse{
		ID:                   r.ID,
		TenantID:             r.TenantID,
		Title:                r.Title,
		IsActive:             r.IsActive,
		IsDefault:            r.IsDefault,
		Version:              r.Version,
		FullName:             r.PersonalInfo.FullName,
		Email:                r.PersonalInfo.Email,
		Phone:                r.PersonalInfo.Phone,
		TotalYearsExp:        r.TotalYearsOfExperience(),
		HasPersonalStatement: r.HasPersonalStatement(),
		ParsedAt:             r.ParsedAt,
		LastUpdatedAt:        r.LastUpdatedAt,
	}

	// Location
	if r.PersonalInfo.Location.City != "" {
		summary.Location = r.PersonalInfo.Location.City
		if r.PersonalInfo.Location.Country != "" {
			summary.Location += ", " + r.PersonalInfo.Location.Country
		}
	}

	// Latest position
	if latest := r.GetLatestPosition(); latest != nil {
		summary.LatestPosition = latest.Title
		summary.LatestCompany = latest.Company
	}

	// Highest education
	if edu := r.GetHighestEducation(); edu != nil {
		summary.HighestEducation = edu.Degree + " in " + edu.Field
	}

	// Top skills (limit to 10)
	allSkills := r.GetAllSkills()
	if len(allSkills) > 10 {
		summary.TopSkills = allSkills[:10]
	} else {
		summary.TopSkills = allSkills
	}

	return summary
}

// ToListResumesResponse creates a paginated list response
func ToListResumesResponse(
	resumes []*Resume,
	page, pageSize, total int,
	activeCount, inactiveCount int,
	defaultResumeID *kernel.ResumeID,
) *ListResumesResponse {
	summaries := make([]ResumeSummaryResponse, len(resumes))
	for i, r := range resumes {
		summaries[i] = *ToResumeSummaryResponse(r)
	}

	return &ListResumesResponse{
		Resumes:       kernel.NewPaginated(summaries, page, pageSize, total),
		ActiveCount:   activeCount,
		InactiveCount: inactiveCount,
		DefaultResume: defaultResumeID,
	}
}

// ToSearchResumesResponse creates a paginated search response
func ToSearchResumesResponse(
	matches []ResumeMatchResult,
	page, pageSize, total int,
	query string,
	filters SearchFilters,
	executionTime string,
) *SearchResumesResponse {
	return &SearchResumesResponse{
		Results:       kernel.NewPaginated(matches, page, pageSize, total),
		SearchQuery:   query,
		Filters:       filters,
		ExecutionTime: executionTime,
	}
}
