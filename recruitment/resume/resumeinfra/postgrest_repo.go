package resumeinfra

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/pkg/logx"
	"github.com/Abraxas-365/relay/recruitment/resume"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

type PostgresResumeRepository struct {
	db *sqlx.DB
}

func NewPostgresResumeRepository(db *sqlx.DB) *PostgresResumeRepository {
	return &PostgresResumeRepository{db: db}
}

// ============================================================================
// CRUD Operations
// ============================================================================

// Create creates a new resume
func (r *PostgresResumeRepository) Create(ctx context.Context, resumeModel *resume.Resume) error {
	query := `
		INSERT INTO resumes (
			id, tenant_id, title, is_active, is_default, version,
			personal_info, work_experience, education, skills, languages,
			certifications, projects, achievements, volunteer_work,
			professional_summary, personal_statement,
			file_url, file_name, file_type,
			parsed_at, last_updated_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17,
			$18, $19, $20,
			$21, $22, $23
		)`

	// Marshal JSONB fields
	personalInfo, err := json.Marshal(resumeModel.PersonalInfo)
	if err != nil {
		return resume.ErrInvalidResumeData().
			WithDetail("field", "personal_info").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	workExperience, err := json.Marshal(resumeModel.WorkExperience)
	if err != nil {
		return resume.ErrInvalidResumeData().
			WithDetail("field", "work_experience").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	education, err := json.Marshal(resumeModel.Education)
	if err != nil {
		return resume.ErrInvalidResumeData().
			WithDetail("field", "education").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	skills, err := json.Marshal(resumeModel.Skills)
	if err != nil {
		return resume.ErrInvalidResumeData().
			WithDetail("field", "skills").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	languages, err := json.Marshal(resumeModel.Languages)
	if err != nil {
		return resume.ErrInvalidResumeData().
			WithDetail("field", "languages").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	certifications, err := json.Marshal(resumeModel.Certifications)
	if err != nil {
		return resume.ErrInvalidResumeData().
			WithDetail("field", "certifications").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	projects, err := json.Marshal(resumeModel.Projects)
	if err != nil {
		return resume.ErrInvalidResumeData().
			WithDetail("field", "projects").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	achievements, err := json.Marshal(resumeModel.Achievements)
	if err != nil {
		return resume.ErrInvalidResumeData().
			WithDetail("field", "achievements").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	volunteerWork, err := json.Marshal(resumeModel.VolunteerWork)
	if err != nil {
		return resume.ErrInvalidResumeData().
			WithDetail("field", "volunteer_work").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	personalStatement, err := json.Marshal(resumeModel.PersonalStatement)
	if err != nil {
		return resume.ErrInvalidResumeData().
			WithDetail("field", "personal_statement").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	_, err = r.db.ExecContext(ctx, query,
		resumeModel.ID, resumeModel.TenantID, resumeModel.Title, resumeModel.IsActive, resumeModel.IsDefault, resumeModel.Version,
		personalInfo, workExperience, education, skills, languages,
		certifications, projects, achievements, volunteerWork,
		resumeModel.ProfessionalSummary, personalStatement,
		resumeModel.FileURL, resumeModel.FileName, resumeModel.FileType,
		resumeModel.ParsedAt, resumeModel.LastUpdatedAt, resumeModel.CreatedAt,
	)
	if err != nil {
		// Check for duplicate key error
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return resume.ErrResumeAlreadyExists().
				WithDetail("resume_id", resumeModel.ID).
				WithDetail("tenant_id", resumeModel.TenantID)
		}
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", resumeModel.ID).
			WithDetail("operation", "insert")
	}

	// Create embeddings if they exist
	logx.Infof("Checking for embeddings to insert for resume ID: %v", resumeModel.HasEmbeddings())
	if resumeModel.HasEmbeddings() {
		if err := r.insertEmbeddings(ctx, resumeModel); err != nil {
			return err
		}
	}

	return nil
}

// Update updates an existing resume
func (r *PostgresResumeRepository) Update(ctx context.Context, id kernel.ResumeID, resumeModel *resume.Resume) error {
	query := `
		UPDATE resumes SET
			title = $1,
			is_active = $2,
			is_default = $3,
			version = $4,
			personal_info = $5,
			work_experience = $6,
			education = $7,
			skills = $8,
			languages = $9,
			certifications = $10,
			projects = $11,
			achievements = $12,
			volunteer_work = $13,
			professional_summary = $14,
			personal_statement = $15,
			last_updated_at = $16
		WHERE id = $17`

	// Marshal JSONB fields
	personalInfo, _ := json.Marshal(resumeModel.PersonalInfo)
	workExperience, _ := json.Marshal(resumeModel.WorkExperience)
	education, _ := json.Marshal(resumeModel.Education)
	skills, _ := json.Marshal(resumeModel.Skills)
	languages, _ := json.Marshal(resumeModel.Languages)
	certifications, _ := json.Marshal(resumeModel.Certifications)
	projects, _ := json.Marshal(resumeModel.Projects)
	achievements, _ := json.Marshal(resumeModel.Achievements)
	volunteerWork, _ := json.Marshal(resumeModel.VolunteerWork)
	personalStatement, _ := json.Marshal(resumeModel.PersonalStatement)

	result, err := r.db.ExecContext(ctx, query,
		resumeModel.Title, resumeModel.IsActive, resumeModel.IsDefault, resumeModel.Version,
		personalInfo, workExperience, education, skills, languages,
		certifications, projects, achievements, volunteerWork,
		resumeModel.ProfessionalSummary, personalStatement,
		resumeModel.LastUpdatedAt, id,
	)
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id).
			WithDetail("operation", "update")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id)
	}
	if rows == 0 {
		return resume.ErrResumeNotFound().
			WithDetail("resume_id", id)
	}

	// Update embeddings if they exist
	if resumeModel.HasEmbeddings() {
		if err := r.updateEmbeddingsInternal(ctx, id, resumeModel.Embeddings); err != nil {
			return err
		}
	}

	return nil
}

// GetByID retrieves a resume by ID
func (r *PostgresResumeRepository) GetByID(ctx context.Context, id kernel.ResumeID) (*resume.Resume, error) {
	query := `
		SELECT 
			id, tenant_id, title, is_active, is_default, version,
			personal_info, work_experience, education, skills, languages,
			certifications, projects, achievements, volunteer_work,
			professional_summary, personal_statement,
			file_url, file_name, file_type,
			parsed_at, last_updated_at, created_at
		FROM resumes
		WHERE id = $1`

	row := &resumeRow{}
	err := r.db.GetContext(ctx, row, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, resume.ErrResumeNotFound().
				WithDetail("resume_id", id)
		}
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id).
			WithDetail("operation", "get")
	}

	resumeModel, err := row.ToDomain()
	if err != nil {
		return nil, resume.ErrInvalidResumeData().
			WithDetail("resume_id", id).
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	// Load embeddings
	embeddings, err := r.getEmbeddings(ctx, id)
	if err == nil {
		resumeModel.Embeddings = *embeddings
	}

	return resumeModel, nil
}

// GetByTenantID retrieves the default resume for a tenant
func (r *PostgresResumeRepository) GetByTenantID(ctx context.Context, tenantID kernel.TenantID) (*resume.Resume, error) {
	return r.GetDefaultByTenantID(ctx, tenantID)
}

// ListByTenantID retrieves all resumes for a tenant
func (r *PostgresResumeRepository) ListByTenantID(ctx context.Context, tenantID kernel.TenantID) ([]*resume.Resume, error) {
	query := `
		SELECT 
			id, tenant_id, title, is_active, is_default, version,
			personal_info, work_experience, education, skills, languages,
			certifications, projects, achievements, volunteer_work,
			professional_summary, personal_statement,
			file_url, file_name, file_type,
			parsed_at, last_updated_at, created_at
		FROM resumes
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows := []resumeRow{}
	err := r.db.SelectContext(ctx, &rows, query, tenantID)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", tenantID).
			WithDetail("operation", "list")
	}

	resumes := make([]*resume.Resume, len(rows))
	for i, row := range rows {
		resumeModel, err := row.ToDomain()
		if err != nil {
			return nil, resume.ErrInvalidResumeData().
				WithDetail("tenant_id", tenantID).
				WithDetail("row_index", i).
				WithDetails(map[string]any{
					"error": err.Error(),
				})
		}

		// Load embeddings
		embeddings, err := r.getEmbeddings(ctx, resumeModel.ID)
		if err == nil {
			resumeModel.Embeddings = *embeddings
		}

		resumes[i] = resumeModel
	}

	return resumes, nil
}

// GetActiveByTenantID retrieves all active resumes for a tenant
func (r *PostgresResumeRepository) GetActiveByTenantID(ctx context.Context, tenantID kernel.TenantID) ([]*resume.Resume, error) {
	query := `
		SELECT 
			id, tenant_id, title, is_active, is_default, version,
			personal_info, work_experience, education, skills, languages,
			certifications, projects, achievements, volunteer_work,
			professional_summary, personal_statement,
			file_url, file_name, file_type,
			parsed_at, last_updated_at, created_at
		FROM resumes
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at DESC`

	rows := []resumeRow{}
	err := r.db.SelectContext(ctx, &rows, query, tenantID)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", tenantID).
			WithDetail("operation", "list_active")
	}

	resumes := make([]*resume.Resume, len(rows))
	for i, row := range rows {
		resumeModel, err := row.ToDomain()
		if err != nil {
			return nil, resume.ErrInvalidResumeData().
				WithDetail("tenant_id", tenantID).
				WithDetail("row_index", i).
				WithDetails(map[string]any{
					"error": err.Error(),
				})
		}

		// Load embeddings
		embeddings, err := r.getEmbeddings(ctx, resumeModel.ID)
		if err == nil {
			resumeModel.Embeddings = *embeddings
		}

		resumes[i] = resumeModel
	}

	return resumes, nil
}

// GetDefaultByTenantID retrieves the default resume for a tenant
func (r *PostgresResumeRepository) GetDefaultByTenantID(ctx context.Context, tenantID kernel.TenantID) (*resume.Resume, error) {
	query := `
		SELECT 
			id, tenant_id, title, is_active, is_default, version,
			personal_info, work_experience, education, skills, languages,
			certifications, projects, achievements, volunteer_work,
			professional_summary, personal_statement,
			file_url, file_name, file_type,
			parsed_at, last_updated_at, created_at
		FROM resumes
		WHERE tenant_id = $1 AND is_default = true
		LIMIT 1`

	row := &resumeRow{}
	err := r.db.GetContext(ctx, row, query, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, resume.ErrResumeNotFound().
				WithDetail("tenant_id", tenantID).
				WithDetail("filter", "default")
		}
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", tenantID).
			WithDetail("operation", "get_default")
	}

	resumeModel, err := row.ToDomain()
	if err != nil {
		return nil, resume.ErrInvalidResumeData().
			WithDetail("tenant_id", tenantID).
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	// Load embeddings
	embeddings, err := r.getEmbeddings(ctx, resumeModel.ID)
	if err == nil {
		resumeModel.Embeddings = *embeddings
	}

	return resumeModel, nil
}

// SetDefault sets a resume as the default for a tenant
func (r *PostgresResumeRepository) SetDefault(ctx context.Context, id kernel.ResumeID, tenantID kernel.TenantID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id).
			WithDetail("tenant_id", tenantID).
			WithDetail("operation", "begin_transaction")
	}
	defer tx.Rollback()

	// Unset all defaults for this tenant
	_, err = tx.ExecContext(ctx, `
		UPDATE resumes 
		SET is_default = false 
		WHERE tenant_id = $1 AND is_default = true
	`, tenantID)
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", tenantID).
			WithDetail("operation", "unset_defaults")
	}

	// Set this one as default
	result, err := tx.ExecContext(ctx, `
		UPDATE resumes 
		SET is_default = true 
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID)
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id).
			WithDetail("tenant_id", tenantID).
			WithDetail("operation", "set_default")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id).
			WithDetail("tenant_id", tenantID)
	}
	if rows == 0 {
		return resume.ErrTenantMismatch().
			WithDetail("resume_id", id).
			WithDetail("tenant_id", tenantID)
	}

	return tx.Commit()
}

// ToggleActive activates or deactivates a resume
func (r *PostgresResumeRepository) ToggleActive(ctx context.Context, id kernel.ResumeID, isActive bool) error {
	query := `UPDATE resumes SET is_active = $1, last_updated_at = NOW() WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, isActive, id)
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id).
			WithDetail("is_active", isActive)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id)
	}
	if rows == 0 {
		return resume.ErrResumeNotFound().
			WithDetail("resume_id", id)
	}

	return nil
}

// Delete deletes a resume
func (r *PostgresResumeRepository) Delete(ctx context.Context, id kernel.ResumeID) error {
	query := `DELETE FROM resumes WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id).
			WithDetail("operation", "delete")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("resume_id", id)
	}
	if rows == 0 {
		return resume.ErrResumeNotFound().
			WithDetail("resume_id", id)
	}

	return nil
}

// Exists checks if any resume exists for a tenant
func (r *PostgresResumeRepository) Exists(ctx context.Context, tenantID kernel.TenantID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM resumes WHERE tenant_id = $1)`

	err := r.db.GetContext(ctx, &exists, query, tenantID)
	if err != nil {
		return false, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", tenantID).
			WithDetail("operation", "exists")
	}

	return exists, nil
}

// CountByTenantID counts resumes for a tenant
func (r *PostgresResumeRepository) CountByTenantID(ctx context.Context, tenantID kernel.TenantID) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM resumes WHERE tenant_id = $1`

	err := r.db.GetContext(ctx, &count, query, tenantID)
	if err != nil {
		return 0, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", tenantID).
			WithDetail("operation", "count")
	}

	return count, nil
}

// ============================================================================
// Pagination
// ============================================================================

// List retrieves all resumes with pagination
func (r *PostgresResumeRepository) List(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[resume.Resume], error) {
	// Count total
	var total int
	err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM resumes`)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("operation", "count_all")
	}

	// Calculate offset
	offset := (pagination.Page - 1) * pagination.PageSize

	// Get paginated results
	query := `
		SELECT 
			id, tenant_id, title, is_active, is_default, version,
			personal_info, work_experience, education, skills, languages,
			certifications, projects, achievements, volunteer_work,
			professional_summary, personal_statement,
			file_url, file_name, file_type,
			parsed_at, last_updated_at, created_at
		FROM resumes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows := []resumeRow{}
	err = r.db.SelectContext(ctx, &rows, query, pagination.PageSize, offset)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("operation", "list_paginated").
			WithDetails(map[string]any{
				"page":      pagination.Page,
				"page_size": pagination.PageSize,
			})
	}

	resumes := make([]resume.Resume, len(rows))
	for i, row := range rows {
		resumeModel, err := row.ToDomain()
		if err != nil {
			return nil, resume.ErrInvalidResumeData().
				WithDetail("row_index", i).
				WithDetails(map[string]any{
					"error": err.Error(),
				})
		}

		// Load embeddings
		embeddings, err := r.getEmbeddings(ctx, resumeModel.ID)
		if err == nil {
			resumeModel.Embeddings = *embeddings
		}

		resumes[i] = *resumeModel
	}

	return &kernel.Paginated[resume.Resume]{
		Items: resumes,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
			Pages:  (total + pagination.PageSize - 1) / pagination.PageSize,
		},
		Empty: len(resumes) == 0,
	}, nil
}

// ListByTenantIDWithPagination retrieves resumes for a tenant with pagination
func (r *PostgresResumeRepository) ListByTenantIDWithPagination(ctx context.Context, tenantID kernel.TenantID, pagination kernel.PaginationOptions) (*kernel.Paginated[resume.Resume], error) {
	// Count total
	var total int
	err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM resumes WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", tenantID).
			WithDetail("operation", "count")
	}

	// Calculate offset
	offset := (pagination.Page - 1) * pagination.PageSize

	// Get paginated results
	query := `
		SELECT 
			id, tenant_id, title, is_active, is_default, version,
			personal_info, work_experience, education, skills, languages,
			certifications, projects, achievements, volunteer_work,
			professional_summary, personal_statement,
			file_url, file_name, file_type,
			parsed_at, last_updated_at, created_at
		FROM resumes
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows := []resumeRow{}
	err = r.db.SelectContext(ctx, &rows, query, tenantID, pagination.PageSize, offset)
	if err != nil {
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeResumeNotFound, err).
			WithDetail("tenant_id", tenantID).
			WithDetail("operation", "list_paginated").
			WithDetails(map[string]any{
				"page":      pagination.Page,
				"page_size": pagination.PageSize,
			})
	}

	resumes := make([]resume.Resume, len(rows))
	for i, row := range rows {
		resumeModel, err := row.ToDomain()
		if err != nil {
			return nil, resume.ErrInvalidResumeData().
				WithDetail("tenant_id", tenantID).
				WithDetail("row_index", i).
				WithDetails(map[string]any{
					"error": err.Error(),
				})
		}

		// Load embeddings
		embeddings, err := r.getEmbeddings(ctx, resumeModel.ID)
		if err == nil {
			resumeModel.Embeddings = *embeddings
		}

		resumes[i] = *resumeModel
	}

	return &kernel.Paginated[resume.Resume]{
		Items: resumes,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
			Pages:  (total + pagination.PageSize - 1) / pagination.PageSize,
		},
		Empty: len(resumes) == 0,
	}, nil
}

// ============================================================================
// Semantic Search with pgvector
// ============================================================================

// SemanticSearch performs vector similarity search using pgvector
func (r *PostgresResumeRepository) SemanticSearch(ctx context.Context, req resume.SearchResumesRequest) ([]resume.ResumeMatchResult, error) {
	// First, we need to generate the query embedding
	// This should be done in the service layer and passed here
	// For now, this is a placeholder that returns an error

	// Build the base query with vector similarity
	baseQuery := `
		SELECT 
			r.id, r.tenant_id, r.title, r.is_active, r.is_default, r.version,
			r.personal_info, r.work_experience, r.education, r.skills, r.languages,
			r.certifications, r.projects, r.achievements, r.volunteer_work,
			r.professional_summary, r.personal_statement,
			r.file_url, r.file_name, r.file_type,
			r.parsed_at, r.last_updated_at, r.created_at,
			-- Weighted similarity scores (cosine distance)
			(1 - (e.experience_embedding <=> $1)) * 0.4 AS experience_score,
			(1 - (e.education_embedding <=> $1)) * 0.2 AS education_score,
			(1 - (e.skills_embedding <=> $1)) * 0.3 AS skills_score,
			(1 - (e.languages_embedding <=> $1)) * 0.1 AS languages_score,
			-- Combined similarity score
			(
				(1 - (e.experience_embedding <=> $1)) * 0.4 +
				(1 - (e.education_embedding <=> $1)) * 0.2 +
				(1 - (e.skills_embedding <=> $1)) * 0.3 +
				(1 - (e.languages_embedding <=> $1)) * 0.1
			) AS similarity_score
		FROM resumes r
		INNER JOIN resume_embeddings e ON r.id = e.resume_id
		WHERE 1=1`

	// Build WHERE clauses and args
	conditions := []string{}
	args := []any{} // Will start with query embedding
	argPos := 2     // Start at $2 since $1 is the query embedding

	// Add tenant filter if specified
	if req.TenantID != nil {
		conditions = append(conditions, fmt.Sprintf("r.tenant_id = $%d", argPos))
		args = append(args, *req.TenantID)
		argPos++
	}

	// Add active filter
	if req.OnlyActive {
		conditions = append(conditions, "r.is_active = true")
	}

	// Add years of experience filter
	if req.MinYearsExperience != nil {
		conditions = append(conditions, fmt.Sprintf("calculate_total_experience_months(r.work_experience) >= $%d", argPos))
		args = append(args, int(*req.MinYearsExperience*12))
		argPos++
	}

	if req.MaxYearsExperience != nil {
		conditions = append(conditions, fmt.Sprintf("calculate_total_experience_months(r.work_experience) <= $%d", argPos))
		args = append(args, int(*req.MaxYearsExperience*12))
		argPos++
	}

	// Add skills filter (required skills)
	if len(req.RequiredSkills) > 0 {
		conditions = append(conditions, fmt.Sprintf("extract_all_skills(r.skills) @> $%d::text[]", argPos))
		args = append(args, pq.Array(req.RequiredSkills))
		argPos++
	}

	// Add location filter
	if len(req.Locations) > 0 {
		locationConditions := []string{}
		for _, loc := range req.Locations {
			locationConditions = append(locationConditions, fmt.Sprintf("r.personal_info->>'location' ILIKE $%d", argPos))
			args = append(args, "%"+loc+"%")
			argPos++
		}
		if len(locationConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(locationConditions, " OR ")+")")
		}
	}

	// Append all conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY and LIMIT
	baseQuery += fmt.Sprintf(`
		ORDER BY similarity_score DESC
		LIMIT $%d`, argPos)
	args = append(args, req.TopK)

	// NOTE: This is a placeholder since we don't have the query embedding yet
	// The embedding should be generated by the service layer
	return []resume.ResumeMatchResult{}, resume.ErrSearchFailed().
		WithDetail("reason", "query embedding must be generated by service layer").
		WithDetail("query", req.Query).
		WithDetails(map[string]any{
			"note": "implement embedding generation in service layer first",
		})

	// When implemented with embedding:
	// queryEmbedding := pgvector.NewVector(embeddingFloats)
	// args = append([]any{queryEmbedding}, args...)
	// ... execute query and return results
}

// SearchByTenant performs semantic search within a specific tenant
func (r *PostgresResumeRepository) SearchByTenant(ctx context.Context, tenantID kernel.TenantID, req resume.SearchResumesRequest) ([]resume.ResumeMatchResult, error) {
	req.TenantID = &tenantID
	return r.SemanticSearch(ctx, req)
}

// ============================================================================
// Embeddings with pgvector
// ============================================================================

// UpdateEmbeddings updates only the embeddings for a resume
func (r *PostgresResumeRepository) UpdateEmbeddings(ctx context.Context, id kernel.ResumeID, embeddings resume.ResumeEmbeddings) error {
	return r.updateEmbeddingsInternal(ctx, id, embeddings)
}

// ============================================================================
// Private Helper Methods
// ============================================================================

func (r *PostgresResumeRepository) insertEmbeddings(ctx context.Context, resumeModel *resume.Resume) error {
	query := `
		INSERT INTO resume_embeddings (
			resume_id, experience_embedding, education_embedding,
			skills_embedding, languages_embedding, personal_statement_embedding,
			model_used, embedding_dim, generated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (resume_id) DO UPDATE SET
			experience_embedding = COALESCE(EXCLUDED.experience_embedding, resume_embeddings.experience_embedding),
			education_embedding = COALESCE(EXCLUDED.education_embedding, resume_embeddings.education_embedding),
			skills_embedding = COALESCE(EXCLUDED.skills_embedding, resume_embeddings.skills_embedding),
			languages_embedding = COALESCE(EXCLUDED.languages_embedding, resume_embeddings.languages_embedding),
			personal_statement_embedding = COALESCE(EXCLUDED.personal_statement_embedding, resume_embeddings.personal_statement_embedding),
			model_used = EXCLUDED.model_used,
			embedding_dim = EXCLUDED.embedding_dim,
			generated_at = EXCLUDED.generated_at`

	_, err := r.db.ExecContext(ctx, query,
		resumeModel.ID,
		float32SliceToVectorOrNil(resumeModel.Embeddings.ExperienceEmbedding),
		float32SliceToVectorOrNil(resumeModel.Embeddings.EducationEmbedding),
		float32SliceToVectorOrNil(resumeModel.Embeddings.SkillsEmbedding),
		float32SliceToVectorOrNil(resumeModel.Embeddings.LanguagesEmbedding),
		float32SliceToVectorOrNil(resumeModel.Embeddings.PersonalStatementEmbedding),
		resumeModel.Embeddings.ModelUsed,
		resumeModel.Embeddings.EmbeddingDim,
		resumeModel.Embeddings.GeneratedAt,
	)

	if err != nil {
		logx.Errorf("Failed to insert embeddings for resume %s: %v", resumeModel.ID, err)
		return resume.ErrEmbeddingGenerationFailed().
			WithDetail("resume_id", resumeModel.ID).
			WithDetail("operation", "insert_embeddings").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	logx.Infof("Successfully inserted embeddings for resume %s", resumeModel.ID)
	return nil
}

// Helper function
func float32SliceToVectorOrNil(slice []float32) interface{} {
	if len(slice) == 0 {
		return nil
	}
	return pgvector.NewVector(slice)
}

func (r *PostgresResumeRepository) updateEmbeddingsInternal(ctx context.Context, id kernel.ResumeID, embeddings resume.ResumeEmbeddings) error {
	query := `
		INSERT INTO resume_embeddings (
			resume_id, experience_embedding, education_embedding,
			skills_embedding, languages_embedding, personal_statement_embedding,
			model_used, embedding_dim, generated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (resume_id) DO UPDATE SET
			experience_embedding = EXCLUDED.experience_embedding,
			education_embedding = EXCLUDED.education_embedding,
			skills_embedding = EXCLUDED.skills_embedding,
			languages_embedding = EXCLUDED.languages_embedding,
			personal_statement_embedding = EXCLUDED.personal_statement_embedding,
			model_used = EXCLUDED.model_used,
			embedding_dim = EXCLUDED.embedding_dim,
			generated_at = EXCLUDED.generated_at`

	_, err := r.db.ExecContext(ctx, query,
		id,
		float32SliceToVector(embeddings.ExperienceEmbedding),
		float32SliceToVector(embeddings.EducationEmbedding),
		float32SliceToVector(embeddings.SkillsEmbedding),
		float32SliceToVector(embeddings.LanguagesEmbedding),
		float32SliceToVector(embeddings.PersonalStatementEmbedding),
		embeddings.ModelUsed,
		embeddings.EmbeddingDim,
		embeddings.GeneratedAt,
	)

	if err != nil {
		return resume.ErrEmbeddingGenerationFailed().
			WithDetail("resume_id", id).
			WithDetail("operation", "update").
			WithDetails(map[string]any{
				"error": err.Error(),
			})
	}

	return nil
}

func (r *PostgresResumeRepository) getEmbeddings(ctx context.Context, resumeID kernel.ResumeID) (*resume.ResumeEmbeddings, error) {
	query := `
		SELECT 
			experience_embedding::text, education_embedding::text,
			skills_embedding::text, languages_embedding::text, 
			personal_statement_embedding::text,
			model_used, embedding_dim, generated_at
		FROM resume_embeddings
		WHERE resume_id = $1`

	row := &embeddingsRow{}
	err := r.db.GetContext(ctx, row, query, resumeID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No embeddings found - not an error
			return nil, nil
		}
		return nil, resume.ErrRegistry.NewWithCause(resume.CodeEmbeddingGenerationFailed, err).
			WithDetail("resume_id", resumeID).
			WithDetail("operation", "get_embeddings")
	}

	return row.ToDomain(), nil
}

// ============================================================================
// pgvector Conversion Helpers
// ============================================================================

// float32SliceToVector converts []float32 to pgvector.Vector
func float32SliceToVector(slice []float32) pgvector.Vector {
	if len(slice) == 0 {
		return pgvector.NewVector([]float32{})
	}
	return pgvector.NewVector(slice)
}

// vectorToFloat32Slice converts pgvector string to []float32
func vectorToFloat32Slice(vectorStr string) []float32 {
	if vectorStr == "" || vectorStr == "[]" {
		return []float32{}
	}

	// Parse pgvector format: [1.0,2.0,3.0]
	vec := pgvector.Vector{}
	if err := vec.Scan(vectorStr); err != nil {
		return []float32{}
	}

	return vec.Slice()
}
