package resumesrv

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Abraxas-365/relay/internal/ai/embeddings"
	"github.com/Abraxas-365/relay/internal/ai/resumeparser"
	"github.com/Abraxas-365/relay/internal/pdf"
	"github.com/Abraxas-365/relay/pkg/fsx"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/pkg/logx"
	"github.com/Abraxas-365/relay/recruitment/resume"
	"github.com/google/uuid"
)

const (
	MaxResumesPerTenant = 20
	EmbeddingModel      = "text-embedding-3-small"
	EmbeddingDimension  = 1536
)

type Service struct {
	repo       resume.Repository
	jobRepo    resume.JobRepository
	parser     *resumeparser.ResumeParser
	embedGen   *embeddings.EmbeddingsGenerator
	fileReader fsx.FileReader
	queue      resume.JobQueue
}

// NewService creates a new resume service
func NewService(
	repo resume.Repository,
	parser *resumeparser.ResumeParser,
	embedGen *embeddings.EmbeddingsGenerator,
	jobRepo resume.JobRepository,
	fileReader fsx.FileReader,
	queue resume.JobQueue,
) *Service {
	return &Service{
		repo:       repo,
		parser:     parser,
		embedGen:   embedGen,
		jobRepo:    jobRepo,
		fileReader: fileReader,
		queue:      queue,
	}
}

// ============================================================================
// Upload & Parse Resume
// ============================================================================

// ParseAndCreateResume uploads, parses, and creates a resume with embeddings
func (s *Service) ParseAndCreateResume(ctx context.Context, req resume.ParseResumeRequest) (*resume.ResumeResponse, error) {
	logx.Infof("Starting ParseAndCreateResume for TenantID: %s, FilePath: %s", req.TenantID, req.FilePath)
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

	// Read file from storage
	fileData, err := s.fileReader.ReadFile(ctx, req.FilePath)
	if err != nil {
		return nil, resume.ErrFileReadFailed().
			WithDetail("file_path", req.FilePath).
			WithDetails(map[string]interface{}{
				"tenant_id": req.TenantID,
				"error":     err.Error(),
			})
	}

	logx.Infof("File read successfully for TenantID: %s, FilePath: %s", req.TenantID, req.FilePath)
	// Parse resume based on file type
	var parsedData *resumeparser.ResumeData
	switch strings.ToLower(req.FileType) {
	case "pdf":
		parsedData, err = s.parsePDFResume(ctx, fileData)
	case "jpg", "jpeg", "png":
		parsedData, err = s.parseImageResume(ctx, fileData)
	default:
		return nil, resume.ErrInvalidFileFormat().
			WithDetail("file_type", req.FileType).
			WithDetail("supported_formats", []string{"pdf", "jpg", "jpeg", "png"})
	}

	if err != nil {
		return nil, resume.ErrResumeParseFailed().
			WithDetail("file_path", req.FilePath).
			WithDetail("file_type", req.FileType).
			WithDetails(map[string]interface{}{
				"tenant_id": req.TenantID,
				"error":     err.Error(),
			})
	}

	// Convert parsed data to domain model
	resumeModel := s.convertParsedDataToDomain(parsedData, req)

	logx.Infof("Resume parsed successfully for TenantID: %s, FilePath: %s", req.TenantID, req.FilePath)
	logx.Infof("Resume Title: %s, Parsed Name: %s, Parsed Email: %s", resumeModel.Title, resumeModel.PersonalInfo.FullName, resumeModel.PersonalInfo.Email)
	// Generate embeddings
	embeddings, err := s.generateResumeEmbeddings(ctx, resumeModel)
	if err != nil {
		return nil, resume.ErrEmbeddingGenerationFailed().
			WithDetail("resume_title", req.Title).
			WithDetails(map[string]interface{}{
				"tenant_id": req.TenantID,
				"error":     err.Error(),
			})
	}
	resumeModel.Embeddings = *embeddings

	// Handle default resume logic
	if req.IsDefault {
		existingDefault, err := s.repo.GetDefaultByTenantID(ctx, req.TenantID)
		if err == nil && existingDefault != nil {
			existingDefault.UnsetAsDefault()
			_ = s.repo.Update(ctx, existingDefault.ID, existingDefault)
		}
	}

	// Create resume
	if err := s.repo.Create(ctx, resumeModel); err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeAlreadyExists, err).
			WithDetail("tenant_id", req.TenantID).
			WithDetail("title", req.Title)
	}

	return resume.ToResumeResponse(resumeModel), nil
}

// parsePDFResume converts PDF to images and parses
func (s *Service) parsePDFResume(ctx context.Context, pdfData []byte) (*resumeparser.ResumeData, error) {
	// Convert PDF pages to images
	images, err := pdf.ConvertPDFToImages(pdfData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert PDF: %w", err)
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("PDF contains no pages")
	}

	// Parse using multi-page or single-page parser
	if len(images) > 1 {
		return s.parser.ParseResumeFromMultiplePages(ctx, images)
	}
	return s.parser.ParseResumeFromImage(ctx, images[0])
}

// parseImageResume parses a single image resume
func (s *Service) parseImageResume(ctx context.Context, imageData []byte) (*resumeparser.ResumeData, error) {
	// Detect and convert image format if needed
	format, err := pdf.DetectImageFormat(imageData)
	if err != nil {
		return nil, fmt.Errorf("invalid image format: %w", err)
	}

	// Convert to JPEG if not already
	if format != "jpeg" && format != "jpg" {
		imageData, err = pdf.ConvertImageToJPEG(imageData)
		if err != nil {
			return nil, fmt.Errorf("failed to convert image: %w", err)
		}
	}

	return s.parser.ParseResumeFromImage(ctx, imageData)
}

// ============================================================================
// CRUD Operations
// ============================================================================

// CreateResume creates a resume manually (without parsing)
func (s *Service) CreateResume(ctx context.Context, req resume.CreateResumeRequest) (*resume.ResumeResponse, error) {
	// Check resume limit
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

	// Build resume model
	now := time.Now()
	resumeModel := &resume.Resume{
		ID:                  kernel.NewResumeID(uuid.NewString()),
		TenantID:            req.TenantID,
		Title:               req.Title,
		IsActive:            req.IsActive,
		IsDefault:           req.IsDefault,
		Version:             1,
		PersonalInfo:        req.PersonalInfo,
		WorkExperience:      req.WorkExperience,
		Education:           req.Education,
		Skills:              req.Skills,
		Languages:           req.Languages,
		Certifications:      req.Certifications,
		Projects:            req.Projects,
		Achievements:        req.Achievements,
		VolunteerWork:       req.VolunteerWork,
		ProfessionalSummary: req.ProfessionalSummary,
		ParsedAt:            now,
		LastUpdatedAt:       now,
		CreatedAt:           now,
	}

	if req.PersonalStatement != nil {
		resumeModel.PersonalStatement = *req.PersonalStatement
	}

	// Validate completeness
	if !resumeModel.IsComplete() {
		return nil, resume.ErrResumeIncomplete().
			WithDetail("tenant_id", req.TenantID).
			WithDetail("title", req.Title).
			WithDetails(map[string]interface{}{
				"has_name":       resumeModel.PersonalInfo.FullName != "",
				"has_email":      resumeModel.PersonalInfo.Email != "",
				"has_experience": resumeModel.HasWorkExperience(),
				"has_education":  resumeModel.HasEducation(),
			})
	}

	// Generate embeddings
	embeddings, err := s.generateResumeEmbeddings(ctx, resumeModel)
	if err != nil {
		return nil, resume.ErrEmbeddingGenerationFailed().
			WithDetail("tenant_id", req.TenantID).
			WithDetail("title", req.Title).
			WithDetails(map[string]interface{}{
				"error": err.Error(),
			})
	}
	resumeModel.Embeddings = *embeddings

	// Handle default resume
	if req.IsDefault {
		if err := s.unsetOtherDefaults(ctx, req.TenantID); err != nil {
			return nil, err
		}
	}

	// Create
	if err := s.repo.Create(ctx, resumeModel); err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeAlreadyExists, err).
			WithDetail("tenant_id", req.TenantID).
			WithDetail("title", req.Title)
	}

	return resume.ToResumeResponse(resumeModel), nil
}

// GetResume retrieves a resume by ID
func (s *Service) GetResume(ctx context.Context, id kernel.ResumeID) (*resume.ResumeResponse, error) {
	resumeModel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, resume.ErrResumeNotFound().
			WithDetail("resume_id", id)
	}

	return resume.ToResumeResponse(resumeModel), nil
}

// UpdateResume updates resume information
func (s *Service) UpdateResume(ctx context.Context, id kernel.ResumeID, req resume.UpdateResumeRequest) (*resume.ResumeResponse, error) {
	// Get existing resume
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, resume.ErrResumeNotFound().
			WithDetail("resume_id", id)
	}

	// Apply updates
	needsEmbeddingUpdate := false
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.PersonalInfo != nil {
		existing.PersonalInfo = *req.PersonalInfo
	}
	if req.WorkExperience != nil {
		existing.WorkExperience = *req.WorkExperience
		needsEmbeddingUpdate = true
	}
	if req.Education != nil {
		existing.Education = *req.Education
		needsEmbeddingUpdate = true
	}
	if req.Skills != nil {
		existing.Skills = *req.Skills
		needsEmbeddingUpdate = true
	}
	if req.Languages != nil {
		existing.Languages = *req.Languages
		needsEmbeddingUpdate = true
	}
	if req.Certifications != nil {
		existing.Certifications = *req.Certifications
	}
	if req.Projects != nil {
		existing.Projects = *req.Projects
	}
	if req.Achievements != nil {
		existing.Achievements = *req.Achievements
	}
	if req.VolunteerWork != nil {
		existing.VolunteerWork = *req.VolunteerWork
	}
	if req.ProfessionalSummary != nil {
		existing.ProfessionalSummary = *req.ProfessionalSummary
	}
	if req.PersonalStatement != nil {
		existing.PersonalStatement = *req.PersonalStatement
		needsEmbeddingUpdate = true
	}

	existing.Version++
	existing.LastUpdatedAt = time.Now()

	// Regenerate embeddings if content changed
	if needsEmbeddingUpdate {
		embeddings, err := s.generateResumeEmbeddings(ctx, existing)
		if err != nil {
			return nil, resume.ErrEmbeddingGenerationFailed().
				WithDetail("resume_id", id).
				WithDetail("tenant_id", existing.TenantID).
				WithDetails(map[string]interface{}{
					"error": err.Error(),
				})
		}
		existing.Embeddings = *embeddings
	}

	// Update
	if err := s.repo.Update(ctx, id, existing); err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id).
			WithDetail("tenant_id", existing.TenantID)
	}

	return resume.ToResumeResponse(existing), nil
}

// DeleteResume deletes a resume
func (s *Service) DeleteResume(ctx context.Context, id kernel.ResumeID) error {
	// Get resume to check if it's default
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return resume.ErrResumeNotFound().
			WithDetail("resume_id", id)
	}

	// Check if there are other resumes
	count, err := s.repo.CountByTenantID(ctx, existing.TenantID)
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", existing.TenantID)
	}

	// If this is the default and there are other resumes, prevent deletion
	if existing.IsDefault && count > 1 {
		return resume.ErrDefaultResumeRequired().
			WithDetail("resume_id", id).
			WithDetail("tenant_id", existing.TenantID).
			WithDetail("message", "set another resume as default before deleting")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id)
	}

	return nil
}

// ============================================================================
// List & Search Operations
// ============================================================================

// ListResumes lists all resumes for a tenant
func (s *Service) ListResumes(ctx context.Context, req resume.ListResumesRequest) (*resume.ListResumesResponse, error) {
	if req.OnlyActive {
		resumes, err := s.repo.GetActiveByTenantID(ctx, req.TenantID)
		if err != nil {
			return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
				WithDetail("tenant_id", req.TenantID)
		}

		// Calculate stats
		activeCount := len(resumes)
		var defaultResumeID *kernel.ResumeID

		for _, r := range resumes {
			if r.IsDefault {
				defaultResumeID = &r.ID
				break
			}
		}

		return resume.ToListResumesResponse(
			resumes,
			req.Pagination.Page,
			req.Pagination.PageSize,
			len(resumes),
			activeCount,
			0,
			defaultResumeID,
		), nil
	}

	// Get paginated results
	paginated, err := s.repo.ListByTenantIDWithPagination(ctx, req.TenantID, req.Pagination)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", req.TenantID)
	}

	// Calculate counts
	activeCount := 0
	inactiveCount := 0
	var defaultResumeID *kernel.ResumeID

	for _, r := range paginated.Items {
		if r.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}
		if r.IsDefault {
			defaultResumeID = &r.ID
		}
	}

	return resume.ToListResumesResponse(
		convertSliceToPointers(paginated.Items),
		paginated.Page.Number,
		paginated.Page.Size,
		paginated.Page.Total,
		activeCount,
		inactiveCount,
		defaultResumeID,
	), nil
}

// SearchResumes performs semantic search on resumes
func (s *Service) SearchResumes(ctx context.Context, req resume.SearchResumesRequest) (*resume.SearchResumesResponse, error) {
	startTime := time.Now()

	// Search
	matches, err := s.repo.SemanticSearch(ctx, req)
	if err != nil {
		return nil, resume.ErrSearchFailed().
			WithDetail("query", req.Query).
			WithDetails(map[string]interface{}{
				"error":   err.Error(),
				"top_k":   req.TopK,
				"filters": req,
			})
	}

	executionTime := time.Since(startTime).String()

	// Build filters
	filters := resume.SearchFilters{
		MinYearsExperience: req.MinYearsExperience,
		MaxYearsExperience: req.MaxYearsExperience,
		RequiredSkills:     req.RequiredSkills,
		PreferredSkills:    req.PreferredSkills,
		Locations:          req.Locations,
		EducationLevel:     req.EducationLevel,
		Languages:          req.Languages,
		Industries:         req.Industries,
		OnlyActive:         req.OnlyActive,
	}

	return resume.ToSearchResumesResponse(
		matches,
		req.Pagination.Page,
		req.Pagination.PageSize,
		len(matches),
		req.Query,
		filters,
		executionTime,
	), nil
}

// ============================================================================
// Resume Management
// ============================================================================

// SetDefaultResume sets a resume as the default for a tenant
func (s *Service) SetDefaultResume(ctx context.Context, tenantID kernel.TenantID, resumeID kernel.ResumeID) error {
	// Verify resume belongs to tenant
	resumeModel, err := s.repo.GetByID(ctx, resumeID)
	if err != nil {
		return resume.ErrResumeNotFound().
			WithDetail("resume_id", resumeID)
	}

	if resumeModel.TenantID != tenantID {
		return resume.ErrTenantMismatch().
			WithDetail("resume_id", resumeID).
			WithDetail("resume_tenant_id", resumeModel.TenantID).
			WithDetail("requested_tenant_id", tenantID)
	}

	if err := s.repo.SetDefault(ctx, resumeID, tenantID); err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", resumeID).
			WithDetail("tenant_id", tenantID)
	}

	return nil
}

// ToggleActive activates or deactivates a resume
func (s *Service) ToggleActive(ctx context.Context, resumeID kernel.ResumeID, isActive bool) error {
	if err := s.repo.ToggleActive(ctx, resumeID, isActive); err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", resumeID).
			WithDetail("is_active", isActive)
	}
	return nil
}

// AddPersonalStatement adds or updates a personal statement
func (s *Service) AddPersonalStatement(ctx context.Context, resumeID kernel.ResumeID, req resume.AddPersonalStatementRequest) (*resume.ResumeResponse, error) {
	// Get resume
	resumeModel, err := s.repo.GetByID(ctx, resumeID)
	if err != nil {
		return nil, resume.ErrResumeNotFound().
			WithDetail("resume_id", resumeID)
	}

	// Update personal statement
	statement := resume.PersonalStatement{
		WhyThisCompany: req.WhyThisCompany,
		WhyThisRole:    req.WhyThisRole,
		CareerGoals:    req.CareerGoals,
		UniqueValue:    req.UniqueValue,
		Essay:          req.Essay,
	}
	resumeModel.AddPersonalStatement(statement)

	// Regenerate embeddings (personal statement affects semantic search)
	embeddings, err := s.generateResumeEmbeddings(ctx, resumeModel)
	if err != nil {
		return nil, resume.ErrEmbeddingGenerationFailed().
			WithDetail("resume_id", resumeID).
			WithDetails(map[string]interface{}{
				"error": err.Error(),
			})
	}
	resumeModel.Embeddings = *embeddings

	// Update
	if err := s.repo.Update(ctx, resumeID, resumeModel); err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", resumeID)
	}

	return resume.ToResumeResponse(resumeModel), nil
}

// GetResumeStats gets statistics for a tenant's resumes
func (s *Service) GetResumeStats(ctx context.Context, tenantID kernel.TenantID) (*resume.ResumeStatsResponse, error) {
	resumes, err := s.repo.ListByTenantID(ctx, tenantID)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", tenantID)
	}

	stats := &resume.ResumeStatsResponse{
		TenantID:              tenantID,
		TotalResumes:          len(resumes),
		ActiveResumes:         0,
		InactiveResumes:       0,
		ResumesWithEmbeddings: 0,
		AverageVersion:        0,
	}

	totalVersion := 0
	var lastUpdated *time.Time

	for _, r := range resumes {
		if r.IsActive {
			stats.ActiveResumes++
		} else {
			stats.InactiveResumes++
		}

		if r.HasEmbeddings() {
			stats.ResumesWithEmbeddings++
		}

		totalVersion += r.Version

		if lastUpdated == nil || r.LastUpdatedAt.After(*lastUpdated) {
			lastUpdated = &r.LastUpdatedAt
		}
	}

	if len(resumes) > 0 {
		stats.AverageVersion = float64(totalVersion) / float64(len(resumes))
	}
	stats.LastUpdated = lastUpdated

	return stats, nil
}

// ============================================================================
// Embeddings Management
// ============================================================================

// UpdateEmbeddings updates embeddings for a specific resume
func (s *Service) UpdateEmbeddings(ctx context.Context, resumeID kernel.ResumeID) error {
	resumeModel, err := s.repo.GetByID(ctx, resumeID)
	if err != nil {
		return resume.ErrResumeNotFound().
			WithDetail("resume_id", resumeID)
	}

	embeddings, err := s.generateResumeEmbeddings(ctx, resumeModel)
	if err != nil {
		return resume.ErrEmbeddingGenerationFailed().
			WithDetail("resume_id", resumeID).
			WithDetails(map[string]interface{}{
				"error": err.Error(),
			})
	}

	if err := s.repo.UpdateEmbeddings(ctx, resumeID, *embeddings); err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", resumeID)
	}

	return nil
}

// BulkUpdateEmbeddings updates embeddings for all resumes of a tenant
func (s *Service) BulkUpdateEmbeddings(ctx context.Context, req resume.BulkUpdateEmbeddingsRequest) (*resume.BulkOperationResponse, error) {
	startTime := time.Now()

	resumes, err := s.repo.ListByTenantID(ctx, req.TenantID)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", req.TenantID)
	}

	response := &resume.BulkOperationResponse{
		TotalProcessed: len(resumes),
		SuccessCount:   0,
		FailureCount:   0,
		Errors:         []string{},
	}

	for _, r := range resumes {
		// Skip if already has embeddings and force is false
		if !req.Force && r.HasEmbeddings() {
			continue
		}

		embeddings, err := s.generateResumeEmbeddings(ctx, r)
		if err != nil {
			response.FailureCount++
			response.Errors = append(response.Errors, fmt.Sprintf("Resume %s: %v", r.ID, err))
			continue
		}

		if err := s.repo.UpdateEmbeddings(ctx, r.ID, *embeddings); err != nil {
			response.FailureCount++
			response.Errors = append(response.Errors, fmt.Sprintf("Resume %s: %v", r.ID, err))
			continue
		}

		response.SuccessCount++
	}

	response.ExecutionTime = time.Since(startTime).String()
	return response, nil
}

// ============================================================================
// Private Helper Methods
// ============================================================================

// generateResumeEmbeddings generates all embeddings for a resume
func (s *Service) generateResumeEmbeddings(ctx context.Context, r *resume.Resume) (*resume.ResumeEmbeddings, error) {
	now := time.Now()

	// Prepare texts for embedding with tracking
	type embeddingRequest struct {
		text     string
		field    string
		position int
	}

	requests := []embeddingRequest{
		{text: s.formatExperienceForEmbedding(r), field: "experience", position: 0},
		{text: s.formatEducationForEmbedding(r), field: "education", position: 1},
		{text: s.formatSkillsForEmbedding(r), field: "skills", position: 2},
		{text: s.formatLanguagesForEmbedding(r), field: "languages", position: 3},
	}

	// Add personal statement if exists
	if r.HasPersonalStatement() {
		requests = append(requests, embeddingRequest{
			text:     s.formatPersonalStatementForEmbedding(r),
			field:    "personal_statement",
			position: 4,
		})
	}

	// Filter out empty texts and build texts array
	var texts []string
	var validRequests []embeddingRequest
	for _, req := range requests {
		trimmed := strings.TrimSpace(req.text)
		if trimmed != "" {
			texts = append(texts, trimmed)
			validRequests = append(validRequests, req)
		}
	}

	// Check if we have any text to embed
	if len(texts) == 0 {
		logx.Warn("No text content available for embedding generation")
		return &resume.ResumeEmbeddings{
			ModelUsed:    EmbeddingModel,
			EmbeddingDim: EmbeddingDimension,
			GeneratedAt:  now,
		}, nil
	}

	logx.Debugf("Generating embeddings for %d text chunks", len(texts))

	// Generate embeddings in batch
	embeddings, err := s.embedGen.GenerateBatchEmbeddings(ctx, texts)
	if err != nil {
		return nil, err
	}

	// Verify we got the expected number of embeddings
	if len(embeddings) != len(texts) {
		return nil, fmt.Errorf("embedding count mismatch: expected %d, got %d", len(texts), len(embeddings))
	}

	// Map embeddings back to their fields
	result := &resume.ResumeEmbeddings{
		ModelUsed:    EmbeddingModel,
		EmbeddingDim: EmbeddingDimension,
		GeneratedAt:  now,
	}

	for i, req := range validRequests {
		switch req.field {
		case "experience":
			result.ExperienceEmbedding = embeddings[i]
		case "education":
			result.EducationEmbedding = embeddings[i]
		case "skills":
			result.SkillsEmbedding = embeddings[i]
		case "languages":
			result.LanguagesEmbedding = embeddings[i]
		case "personal_statement":
			result.PersonalStatementEmbedding = embeddings[i]
		}
	}

	logx.Debugf("Successfully generated embeddings for resume")
	return result, nil
}

// formatExperienceForEmbedding formats work experience for embedding
func (s *Service) formatExperienceForEmbedding(r *resume.Resume) string {
	if !r.HasWorkExperience() {
		return ""
	}

	var parts []string
	for _, exp := range r.WorkExperience {
		text := fmt.Sprintf("%s at %s (%s to %s). ", exp.Title, exp.Company, exp.StartDate, exp.EndDate)
		text += exp.DescriptionNormalized + " "
		if len(exp.Achievements) > 0 {
			text += "Achievements: " + strings.Join(exp.Achievements, ". ") + " "
		}
		if len(exp.SkillsUsed) > 0 {
			text += "Skills: " + strings.Join(exp.SkillsUsed, ", ")
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n")
}

// formatEducationForEmbedding formats education for embedding
func (s *Service) formatEducationForEmbedding(r *resume.Resume) string {
	if !r.HasEducation() {
		return ""
	}

	var parts []string
	for _, edu := range r.Education {
		text := fmt.Sprintf("%s in %s from %s (%s). ", edu.Degree, edu.Field, edu.Institution, edu.GraduationDate)
		text += edu.DescriptionNormalized
		if len(edu.Honors) > 0 {
			text += " Honors: " + strings.Join(edu.Honors, ", ")
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n")
}

// formatSkillsForEmbedding formats skills for embedding
func (s *Service) formatSkillsForEmbedding(r *resume.Resume) string {
	var parts []string

	if len(r.Skills.HardSkills) > 0 {
		skillNames := make([]string, len(r.Skills.HardSkills))
		for i, skill := range r.Skills.HardSkills {
			skillNames[i] = skill.Name
			if skill.ProficiencyLevel != "" {
				skillNames[i] += " (" + skill.ProficiencyLevel + ")"
			}
		}
		parts = append(parts, "Technical Skills: "+strings.Join(skillNames, ", "))
	}

	if len(r.Skills.SoftSkills) > 0 {
		skillNames := make([]string, len(r.Skills.SoftSkills))
		for i, skill := range r.Skills.SoftSkills {
			skillNames[i] = skill.Name
		}
		parts = append(parts, "Soft Skills: "+strings.Join(skillNames, ", "))
	}

	return strings.Join(parts, ". ")
}

// formatLanguagesForEmbedding formats languages for embedding
func (s *Service) formatLanguagesForEmbedding(r *resume.Resume) string {
	if len(r.Languages) == 0 {
		return ""
	}

	langStrings := make([]string, len(r.Languages))
	for i, lang := range r.Languages {
		langStrings[i] = fmt.Sprintf("%s (%s)", lang.Language, lang.Proficiency)
	}
	return "Languages: " + strings.Join(langStrings, ", ")
}

// formatPersonalStatementForEmbedding formats personal statement for embedding
func (s *Service) formatPersonalStatementForEmbedding(r *resume.Resume) string {
	var parts []string

	if r.PersonalStatement.WhyThisCompany != "" {
		parts = append(parts, "Why this company: "+r.PersonalStatement.WhyThisCompany)
	}
	if r.PersonalStatement.WhyThisRole != "" {
		parts = append(parts, "Why this role: "+r.PersonalStatement.WhyThisRole)
	}
	if r.PersonalStatement.CareerGoals != "" {
		parts = append(parts, "Career goals: "+r.PersonalStatement.CareerGoals)
	}
	if r.PersonalStatement.UniqueValue != "" {
		parts = append(parts, "Unique value: "+r.PersonalStatement.UniqueValue)
	}
	if r.PersonalStatement.Essay != "" {
		parts = append(parts, r.PersonalStatement.Essay)
	}

	return strings.Join(parts, ". ")
}

// convertParsedDataToDomain converts parser output to domain model
func (s *Service) convertParsedDataToDomain(parsed *resumeparser.ResumeData, req resume.ParseResumeRequest) *resume.Resume {
	logx.Infof("Parsed Resume Data: %+v", parsed)
	now := time.Now()

	// Convert personal info
	personalInfo := resume.PersonalInfo{
		FullName: parsed.PersonalInfo.Name,
		Email:    parsed.PersonalInfo.Email,
		Phone:    parsed.PersonalInfo.Phone,
		LinkedIn: parsed.PersonalInfo.LinkedIn,
	}

	// Parse location
	if parsed.PersonalInfo.Location != "" {
		personalInfo.Location = resume.Location{
			City:    parsed.PersonalInfo.Location,
			Country: "",
		}
	}

	// Convert work experience
	workExp := make([]resume.WorkExperience, len(parsed.Experience))
	for i, exp := range parsed.Experience {
		workExp[i] = resume.WorkExperience{
			Company:               exp.Company,
			Title:                 exp.Title,
			StartDate:             exp.StartDate,
			EndDate:               exp.EndDate,
			DurationMonths:        calculateDurationMonths(exp.StartDate, exp.EndDate),
			DescriptionNormalized: strings.Join(exp.Responsibilities, ". "),
			Achievements:          exp.Responsibilities,
		}
	}

	// Convert education
	education := make([]resume.Education, len(parsed.Education))
	for i, edu := range parsed.Education {
		education[i] = resume.Education{
			Institution:           edu.Institution,
			Degree:                edu.Degree,
			Field:                 edu.Field,
			GraduationDate:        edu.GraduationDate,
			DescriptionNormalized: fmt.Sprintf("%s in %s", edu.Degree, edu.Field),
		}
	}

	// Convert skills
	hardSkills := make([]resume.Skill, len(parsed.HardSkills))
	for i, skill := range parsed.HardSkills {
		hardSkills[i] = resume.Skill{
			Name:             skill.Name,
			ProficiencyLevel: skill.ProficiencyLevel,
		}
	}

	softSkills := make([]resume.Skill, len(parsed.SoftSkills))
	for i, skill := range parsed.SoftSkills {
		softSkills[i] = resume.Skill{
			Name:             skill.Name,
			ProficiencyLevel: skill.ProficiencyLevel,
		}
	}

	skills := resume.Skills{
		HardSkills: hardSkills,
		SoftSkills: softSkills,
	}

	// Convert languages
	languages := make([]resume.Language, len(parsed.Languages))
	for i, lang := range parsed.Languages {
		languages[i] = resume.Language{
			Language:    lang.Language,
			Proficiency: lang.Proficiency,
		}
	}

	// Convert certifications
	certifications := make([]resume.Certification, len(parsed.Certifications))
	for i, cert := range parsed.Certifications {
		certifications[i] = resume.Certification{
			Name:   cert,
			Issuer: "",
		}
	}

	personalStatement := resume.PersonalStatement{
		WhyThisCompany: parsed.PersonalStatement.WhyThisCompany,
		WhyThisRole:    parsed.PersonalStatement.WhyThisRole,
		CareerGoals:    parsed.PersonalStatement.CareerGoals,
		UniqueValue:    parsed.PersonalStatement.UniqueValue,
		Essay:          parsed.PersonalStatement.Essay,
		WrittenAt:      nil,
	}

	return &resume.Resume{
		ID:                  kernel.NewResumeID(uuid.NewString()),
		TenantID:            req.TenantID,
		Title:               req.Title,
		IsActive:            req.IsActive,
		IsDefault:           req.IsDefault,
		Version:             1,
		PersonalInfo:        personalInfo,
		WorkExperience:      workExp,
		Education:           education,
		Skills:              skills,
		Languages:           languages,
		Certifications:      certifications,
		ProfessionalSummary: parsed.Summary,
		PersonalStatement:   personalStatement, // ✅ Now correctly mapped
		FileURL:             req.FilePath,
		FileName:            req.FileName,
		FileType:            req.FileType,
		ParsedAt:            now,
		LastUpdatedAt:       now,
		CreatedAt:           now,
	}
}

// unsetOtherDefaults unsets default flag on other resumes
func (s *Service) unsetOtherDefaults(ctx context.Context, tenantID kernel.TenantID) error {
	existing, err := s.repo.GetDefaultByTenantID(ctx, tenantID)
	if err == nil && existing != nil {
		existing.UnsetAsDefault()
		return s.repo.Update(ctx, existing.ID, existing)
	}
	return nil
}

// calculateDurationMonths calculates months between two dates
func calculateDurationMonths(startDate, endDate string) int {
	// Simple calculation - you might want to use a proper date parser
	if endDate == "Present" {
		endDate = time.Now().Format("2006-01")
	}

	start, _ := time.Parse("2006-01", startDate)
	end, _ := time.Parse("2006-01", endDate)

	years := end.Year() - start.Year()
	months := int(end.Month() - start.Month())

	return years*12 + months
}

// convertSliceToPointers converts slice of values to slice of pointers
func convertSliceToPointers(items []resume.Resume) []*resume.Resume {
	result := make([]*resume.Resume, len(items))
	for i := range items {
		result[i] = &items[i]
	}
	return result
}
