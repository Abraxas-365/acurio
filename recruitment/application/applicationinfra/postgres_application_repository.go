package applicationinfra

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/application"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

// PostgresApplicationRepository implements application.Repository using PostgreSQL
type PostgresApplicationRepository struct {
	db *sqlx.DB
}

// NewPostgresApplicationRepository creates a new PostgreSQL application repository
func NewPostgresApplicationRepository(db *sqlx.DB) *PostgresApplicationRepository {
	return &PostgresApplicationRepository{
		db: db,
	}
}

// ============================================================================
// Database Models
// ============================================================================

type applicationModel struct {
	ID              string          `db:"id"`
	JobID           string          `db:"job_id"`
	CandidateID     string          `db:"candidate_id"`
	ResumeSummary   string          `db:"resume_summary"`
	ResumeEmbedding pgvector.Vector `db:"resume_embedding"`
	ResumeBucketUrl string          `db:"resume_bucket_url"`
	Status          string          `db:"status"`
	ReviewerID      *string         `db:"reviewer_id"`
	SubmittedBy     *string         `db:"submitted_by"`
	StatusChangedAt *time.Time      `db:"status_changed_at"`
	ArchivedAt      *time.Time      `db:"archived_at"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

// applicationWithDetailsModel for joined queries
type applicationWithDetailsModel struct {
	applicationModel
	JobTitle       string `db:"job_title"`
	CandidateEmail string `db:"candidate_email"`
	FirstName      string `db:"first_name"`
	LastName       string `db:"last_name"`
}

// toEntity converts database model to domain entity
func (m *applicationModel) toEntity() *application.Application {
	var reviewerID *kernel.UserID
	if m.ReviewerID != nil {
		uid := kernel.UserID(*m.ReviewerID)
		reviewerID = &uid
	}

	var submittedBy *kernel.UserID
	if m.SubmittedBy != nil {
		uid := kernel.UserID(*m.SubmittedBy)
		submittedBy = &uid
	}

	return &application.Application{
		ID:              kernel.ApplicationID(m.ID),
		JobID:           kernel.JobID(m.JobID),
		CandidateID:     kernel.CandidateID(m.CandidateID),
		ResumeSummary:   kernel.ResumeSummary(m.ResumeSummary),
		ResumeEmbedding: kernel.ResumeEmbedding(m.ResumeEmbedding.Slice()),
		ResumeBucketUrl: kernel.BucketURL(m.ResumeBucketUrl),
		Status:          application.ApplicationStatus(m.Status),
		ReviewerID:      reviewerID,
		SubmittedBy:     submittedBy,
		StatusChangedAt: m.StatusChangedAt,
		ArchivedAt:      m.ArchivedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

// toDetailsResponse converts joined model to details response
func (m *applicationWithDetailsModel) toDetailsResponse() *application.ApplicationWithDetailsResponse {
	candidateName := fmt.Sprintf("%s %s", m.FirstName, m.LastName)
	if m.FirstName == "" && m.LastName == "" {
		candidateName = "Unknown"
	}

	return &application.ApplicationWithDetailsResponse{
		ID:              kernel.ApplicationID(m.ID),
		JobID:           kernel.JobID(m.JobID),
		JobTitle:        kernel.JobTitle(m.JobTitle),
		CandidateID:     kernel.CandidateID(m.CandidateID),
		CandidateName:   candidateName,
		CandidateEmail:  kernel.Email(m.CandidateEmail),
		ResumeSummary:   kernel.ResumeSummary(m.ResumeSummary),
		ResumeBucketUrl: kernel.BucketURL(m.ResumeBucketUrl),
	}
}

// fromEntity converts domain entity to database model
func fromEntity(app *application.Application) *applicationModel {
	var reviewerID *string
	if app.ReviewerID != nil {
		rid := string(*app.ReviewerID)
		reviewerID = &rid
	}

	var submittedBy *string
	if app.SubmittedBy != nil {
		sid := string(*app.SubmittedBy)
		submittedBy = &sid
	}

	return &applicationModel{
		ID:              string(app.ID),
		JobID:           string(app.JobID),
		CandidateID:     string(app.CandidateID),
		ResumeSummary:   string(app.ResumeSummary),
		ResumeEmbedding: pgvector.NewVector(app.ResumeEmbedding),
		ResumeBucketUrl: string(app.ResumeBucketUrl),
		Status:          string(app.Status),
		ReviewerID:      reviewerID,
		SubmittedBy:     submittedBy,
		StatusChangedAt: app.StatusChangedAt,
		ArchivedAt:      app.ArchivedAt,
		CreatedAt:       app.CreatedAt,
		UpdatedAt:       app.UpdatedAt,
	}
}

// ============================================================================
// Repository Implementation
// ============================================================================

// Create creates a new application
func (r *PostgresApplicationRepository) Create(ctx context.Context, app *application.Application) error {
	model := fromEntity(app)

	query := `
		INSERT INTO applications (
			id, job_id, candidate_id, resume_summary, resume_embedding,
			resume_bucket_url, status, reviewer_id, submitted_by,
			status_changed_at, archived_at, created_at, updated_at
		) VALUES (
			:id, :job_id, :candidate_id, :resume_summary, :resume_embedding,
			:resume_bucket_url, :status, :reviewer_id, :submitted_by,
			:status_changed_at, :archived_at, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, model)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return application.ErrApplicationAlreadyExists()
			}
			if pqErr.Code == "23503" { // foreign_key_violation
				return fmt.Errorf("invalid foreign key reference: %w", err)
			}
		}
		return fmt.Errorf("failed to create application: %w", err)
	}

	return nil
}

// Update updates an existing application
func (r *PostgresApplicationRepository) Update(ctx context.Context, id kernel.ApplicationID, app *application.Application) error {
	model := fromEntity(app)

	query := `
		UPDATE applications SET
			resume_summary = :resume_summary,
			resume_embedding = :resume_embedding,
			resume_bucket_url = :resume_bucket_url,
			status = :status,
			reviewer_id = :reviewer_id,
			status_changed_at = :status_changed_at,
			archived_at = :archived_at,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, model)
	if err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return application.ErrApplicationNotFound()
	}

	return nil
}

// GetByID retrieves an application by ID
func (r *PostgresApplicationRepository) GetByID(ctx context.Context, id kernel.ApplicationID) (*application.Application, error) {
	query := `
		SELECT 
			id, job_id, candidate_id, resume_summary, resume_embedding,
			resume_bucket_url, status, reviewer_id, submitted_by,
			status_changed_at, archived_at, created_at, updated_at
		FROM applications
		WHERE id = $1
	`

	var model applicationModel
	err := r.db.GetContext(ctx, &model, query, string(id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, application.ErrApplicationNotFound()
		}
		return nil, fmt.Errorf("failed to get application by id: %w", err)
	}

	return model.toEntity(), nil
}

// GetWithDetails retrieves an application with candidate and job details
func (r *PostgresApplicationRepository) GetWithDetails(ctx context.Context, id kernel.ApplicationID) (*application.ApplicationWithDetailsResponse, error) {
	query := `
		SELECT 
			a.id, a.job_id, a.candidate_id, a.resume_summary,
			a.resume_bucket_url, a.created_at, a.updated_at,
			j.job_title,
			c.email as candidate_email,
			c.first_name,
			c.last_name
		FROM applications a
		INNER JOIN jobs j ON a.job_id = j.id
		INNER JOIN candidates c ON a.candidate_id = c.id
		WHERE a.id = $1
	`

	var model applicationWithDetailsModel
	err := r.db.GetContext(ctx, &model, query, string(id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, application.ErrApplicationNotFound()
		}
		return nil, fmt.Errorf("failed to get application with details: %w", err)
	}

	return model.toDetailsResponse(), nil
}

// Delete deletes an application by ID
func (r *PostgresApplicationRepository) Delete(ctx context.Context, id kernel.ApplicationID) error {
	query := `DELETE FROM applications WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, string(id))
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return application.ErrApplicationNotFound()
	}

	return nil
}

// List retrieves all applications with pagination
func (r *PostgresApplicationRepository) List(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[application.Application], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM applications`
	if err := r.db.GetContext(ctx, &total, countQuery); err != nil {
		return nil, fmt.Errorf("failed to count applications: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	// Get paginated results
	query := `
		SELECT 
			id, job_id, candidate_id, resume_summary, resume_embedding,
			resume_bucket_url, status, reviewer_id, submitted_by,
			status_changed_at, archived_at, created_at, updated_at
		FROM applications
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var models []applicationModel
	err := r.db.SelectContext(ctx, &models, query, pagination.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	// Convert to entities
	entities := make([]application.Application, 0, len(models))
	for _, model := range models {
		entities = append(entities, *model.toEntity())
	}

	return &kernel.Paginated[application.Application]{
		Items: entities,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
			Pages:  totalPages,
		},
		Empty: len(entities) == 0,
	}, nil
}

// ListByJobID retrieves applications for a specific job
func (r *PostgresApplicationRepository) ListByJobID(ctx context.Context, jobID kernel.JobID, pagination kernel.PaginationOptions) (*kernel.Paginated[application.Application], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM applications WHERE job_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, string(jobID)); err != nil {
		return nil, fmt.Errorf("failed to count job applications: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	// Get paginated results
	query := `
		SELECT 
			id, job_id, candidate_id, resume_summary, resume_embedding,
			resume_bucket_url, status, reviewer_id, submitted_by,
			status_changed_at, archived_at, created_at, updated_at
		FROM applications
		WHERE job_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var models []applicationModel
	err := r.db.SelectContext(ctx, &models, query, string(jobID), pagination.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications by job: %w", err)
	}

	// Convert to entities
	entities := make([]application.Application, 0, len(models))
	for _, model := range models {
		entities = append(entities, *model.toEntity())
	}

	return &kernel.Paginated[application.Application]{
		Items: entities,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
			Pages:  totalPages,
		},
		Empty: len(entities) == 0,
	}, nil
}

// ListByCandidateID retrieves applications for a specific candidate
func (r *PostgresApplicationRepository) ListByCandidateID(ctx context.Context, candidateID kernel.CandidateID, pagination kernel.PaginationOptions) (*kernel.Paginated[application.Application], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM applications WHERE candidate_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, string(candidateID)); err != nil {
		return nil, fmt.Errorf("failed to count candidate applications: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	// Get paginated results
	query := `
		SELECT 
			id, job_id, candidate_id, resume_summary, resume_embedding,
			resume_bucket_url, status, reviewer_id, submitted_by,
			status_changed_at, archived_at, created_at, updated_at
		FROM applications
		WHERE candidate_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var models []applicationModel
	err := r.db.SelectContext(ctx, &models, query, string(candidateID), pagination.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications by candidate: %w", err)
	}

	// Convert to entities
	entities := make([]application.Application, 0, len(models))
	for _, model := range models {
		entities = append(entities, *model.toEntity())
	}

	return &kernel.Paginated[application.Application]{
		Items: entities,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
			Pages:  totalPages,
		},
		Empty: len(entities) == 0,
	}, nil
}

// ListWithDetailsByJobID retrieves applications with details for a specific job
func (r *PostgresApplicationRepository) ListWithDetailsByJobID(ctx context.Context, jobID kernel.JobID, pagination kernel.PaginationOptions) (*kernel.Paginated[application.ApplicationWithDetailsResponse], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM applications WHERE job_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, string(jobID)); err != nil {
		return nil, fmt.Errorf("failed to count job applications: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	// Get paginated results with details
	query := `
		SELECT 
			a.id, a.job_id, a.candidate_id, a.resume_summary,
			a.resume_bucket_url, a.created_at, a.updated_at,
			j.job_title,
			c.email as candidate_email,
			c.first_name,
			c.last_name
		FROM applications a
		INNER JOIN jobs j ON a.job_id = j.id
		INNER JOIN candidates c ON a.candidate_id = c.id
		WHERE a.job_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`

	var models []applicationWithDetailsModel
	err := r.db.SelectContext(ctx, &models, query, string(jobID), pagination.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications with details by job: %w", err)
	}

	// Convert to response DTOs
	responses := make([]application.ApplicationWithDetailsResponse, 0, len(models))
	for _, model := range models {
		responses = append(responses, *model.toDetailsResponse())
	}

	return &kernel.Paginated[application.ApplicationWithDetailsResponse]{
		Items: responses,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
			Pages:  totalPages,
		},
		Empty: len(responses) == 0,
	}, nil
}

// ListWithDetailsByCandidateID retrieves applications with details for a specific candidate
func (r *PostgresApplicationRepository) ListWithDetailsByCandidateID(ctx context.Context, candidateID kernel.CandidateID, pagination kernel.PaginationOptions) (*kernel.Paginated[application.ApplicationWithDetailsResponse], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM applications WHERE candidate_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, string(candidateID)); err != nil {
		return nil, fmt.Errorf("failed to count candidate applications: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	// Get paginated results with details
	query := `
		SELECT 
			a.id, a.job_id, a.candidate_id, a.resume_summary,
			a.resume_bucket_url, a.created_at, a.updated_at,
			j.job_title,
			c.email as candidate_email,
			c.first_name,
			c.last_name
		FROM applications a
		INNER JOIN jobs j ON a.job_id = j.id
		INNER JOIN candidates c ON a.candidate_id = c.id
		WHERE a.candidate_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`

	var models []applicationWithDetailsModel
	err := r.db.SelectContext(ctx, &models, query, string(candidateID), pagination.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications with details by candidate: %w", err)
	}

	// Convert to response DTOs
	responses := make([]application.ApplicationWithDetailsResponse, 0, len(models))
	for _, model := range models {
		responses = append(responses, *model.toDetailsResponse())
	}

	return &kernel.Paginated[application.ApplicationWithDetailsResponse]{
		Items: responses,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
			Pages:  totalPages,
		},
		Empty: len(responses) == 0,
	}, nil
}

// Exists checks if an application exists by ID
func (r *PostgresApplicationRepository) Exists(ctx context.Context, id kernel.ApplicationID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM applications WHERE id = $1)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, string(id))
	if err != nil {
		return false, fmt.Errorf("failed to check application existence: %w", err)
	}

	return exists, nil
}

// ExistsByJobAndCandidate checks if an application exists for a job and candidate
func (r *PostgresApplicationRepository) ExistsByJobAndCandidate(ctx context.Context, jobID kernel.JobID, candidateID kernel.CandidateID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM applications WHERE job_id = $1 AND candidate_id = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, string(jobID), string(candidateID))
	if err != nil {
		return false, fmt.Errorf("failed to check application existence: %w", err)
	}

	return exists, nil
}

// UpdateResumeBucketUrl updates the resume bucket URL
func (r *PostgresApplicationRepository) UpdateResumeBucketUrl(ctx context.Context, id kernel.ApplicationID, url kernel.BucketURL) error {
	query := `
		UPDATE applications 
		SET resume_bucket_url = $1, 
		    updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, string(url), time.Now(), string(id))
	if err != nil {
		return fmt.Errorf("failed to update resume bucket url: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return application.ErrApplicationNotFound()
	}

	return nil
}

// CountByJobID counts applications for a specific job
func (r *PostgresApplicationRepository) CountByJobID(ctx context.Context, jobID kernel.JobID) (int64, error) {
	query := `SELECT COUNT(*) FROM applications WHERE job_id = $1`

	var count int64
	err := r.db.GetContext(ctx, &count, query, string(jobID))
	if err != nil {
		return 0, fmt.Errorf("failed to count job applications: %w", err)
	}

	return count, nil
}

// CountByCandidateID counts applications for a specific candidate
func (r *PostgresApplicationRepository) CountByCandidateID(ctx context.Context, candidateID kernel.CandidateID) (int64, error) {
	query := `SELECT COUNT(*) FROM applications WHERE candidate_id = $1`

	var count int64
	err := r.db.GetContext(ctx, &count, query, string(candidateID))
	if err != nil {
		return 0, fmt.Errorf("failed to count candidate applications: %w", err)
	}

	return count, nil
}

// AssignReviewer assigns a reviewer to an application
func (r *PostgresApplicationRepository) AssignReviewer(ctx context.Context, id kernel.ApplicationID, reviewerID kernel.UserID) error {
	query := `
		UPDATE applications 
		SET reviewer_id = $1, 
		    updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, string(reviewerID), time.Now(), string(id))
	if err != nil {
		return fmt.Errorf("failed to assign reviewer: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return application.ErrApplicationNotFound()
	}

	return nil
}

// GetApplicationsByReviewer retrieves applications assigned to a reviewer
func (r *PostgresApplicationRepository) GetApplicationsByReviewer(ctx context.Context, reviewerID kernel.UserID, pagination kernel.PaginationOptions) (*kernel.Paginated[application.Application], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM applications WHERE reviewer_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, string(reviewerID)); err != nil {
		return nil, fmt.Errorf("failed to count reviewer applications: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	// Get paginated results
	query := `
		SELECT 
			id, job_id, candidate_id, resume_summary, resume_embedding,
			resume_bucket_url, status, reviewer_id, submitted_by,
			status_changed_at, archived_at, created_at, updated_at
		FROM applications
		WHERE reviewer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var models []applicationModel
	err := r.db.SelectContext(ctx, &models, query, string(reviewerID), pagination.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get applications by reviewer: %w", err)
	}

	// Convert to entities
	entities := make([]application.Application, 0, len(models))
	for _, model := range models {
		entities = append(entities, *model.toEntity())
	}

	return &kernel.Paginated[application.Application]{
		Items: entities,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
			Pages:  totalPages,
		},
		Empty: len(entities) == 0,
	}, nil
}

// Archive archives an application
func (r *PostgresApplicationRepository) Archive(ctx context.Context, id kernel.ApplicationID) error {
	query := `
		UPDATE applications 
		SET status = 'ARCHIVED', 
		    archived_at = $1, 
		    updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, string(id))
	if err != nil {
		return fmt.Errorf("failed to archive application: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return application.ErrApplicationNotFound()
	}

	return nil
}

// Unarchive unarchives an application
func (r *PostgresApplicationRepository) Unarchive(ctx context.Context, id kernel.ApplicationID) error {
	query := `
		UPDATE applications 
		SET status = 'SUBMITTED', 
		    archived_at = NULL, 
		    updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, string(id))
	if err != nil {
		return fmt.Errorf("failed to unarchive application: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return application.ErrApplicationNotFound()
	}

	return nil
}
