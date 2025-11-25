package jobinfra

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/job"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// PostgresJobRepository implements job.Repository using PostgreSQL
type PostgresJobRepository struct {
	db *sqlx.DB
}

// NewPostgresJobRepository creates a new PostgreSQL job repository
func NewPostgresJobRepository(db *sqlx.DB) *PostgresJobRepository {
	return &PostgresJobRepository{
		db: db,
	}
}

// ============================================================================
// Database Model
// ============================================================================

type jobModel struct {
	ID                  string          `db:"id"`
	JobTitle            string          `db:"job_title"`
	JobDescription      string          `db:"job_description"`
	JobPosition         string          `db:"job_position"`
	GeneralRequirements json.RawMessage `db:"general_requirements"`
	Benefits            json.RawMessage `db:"benefits"`
	PostedBy            string          `db:"posted_by"`
	Status              string          `db:"status"`
	PublishedAt         *time.Time      `db:"published_at"`
	ArchivedAt          *time.Time      `db:"archived_at"`
	CreatedAt           time.Time       `db:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"`
}

// toEntity converts database model to domain entity
func (m *jobModel) toEntity() (*job.Job, error) {
	var requirements []kernel.JobRequirement
	if len(m.GeneralRequirements) > 0 {
		if err := json.Unmarshal(m.GeneralRequirements, &requirements); err != nil {
			return nil, fmt.Errorf("failed to unmarshal requirements: %w", err)
		}
	}

	var benefits []kernel.JobBenefit
	if len(m.Benefits) > 0 {
		if err := json.Unmarshal(m.Benefits, &benefits); err != nil {
			return nil, fmt.Errorf("failed to unmarshal benefits: %w", err)
		}
	}

	return &job.Job{
		ID:                  kernel.JobID(m.ID),
		Title:               kernel.JobTitle(m.JobTitle),
		Description:         kernel.JobDescription(m.JobDescription),
		Position:            kernel.JobPosition(m.JobPosition),
		GeneralRequirements: requirements,
		Benefits:            benefits,
		PostedBy:            kernel.UserID(m.PostedBy),
		Status:              job.JobStatus(m.Status),
		PublishedAt:         m.PublishedAt,
		ArchivedAt:          m.ArchivedAt,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}, nil
}

// fromEntity converts domain entity to database model
func fromEntity(j *job.Job) (*jobModel, error) {
	requirements, err := json.Marshal(j.GeneralRequirements)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal requirements: %w", err)
	}

	benefits, err := json.Marshal(j.Benefits)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal benefits: %w", err)
	}

	return &jobModel{
		ID:                  string(j.ID),
		JobTitle:            string(j.Title),
		JobDescription:      string(j.Description),
		JobPosition:         string(j.Position),
		GeneralRequirements: requirements,
		Benefits:            benefits,
		PostedBy:            string(j.PostedBy),
		Status:              string(j.Status),
		PublishedAt:         j.PublishedAt,
		ArchivedAt:          j.ArchivedAt,
		CreatedAt:           j.CreatedAt,
		UpdatedAt:           j.UpdatedAt,
	}, nil
}

// ============================================================================
// Repository Implementation
// ============================================================================

// Create creates a new job
func (r *PostgresJobRepository) Create(ctx context.Context, jobEntity *job.Job) error {
	model, err := fromEntity(jobEntity)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO jobs (
			id, job_title, job_description, job_position,
			general_requirements, benefits, posted_by, status,
			published_at, archived_at, created_at, updated_at
		) VALUES (
			:id, :job_title, :job_description, :job_position,
			:general_requirements, :benefits, :posted_by, :status,
			:published_at, :archived_at, :created_at, :updated_at
		)
	`

	_, err = r.db.NamedExecContext(ctx, query, model)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return job.ErrJobAlreadyExists()
			}
			if pqErr.Code == "23503" { // foreign_key_violation
				return fmt.Errorf("invalid posted_by user_id: %w", err)
			}
		}
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// Update updates an existing job
func (r *PostgresJobRepository) Update(ctx context.Context, id kernel.JobID, jobEntity *job.Job) error {
	model, err := fromEntity(jobEntity)
	if err != nil {
		return err
	}

	query := `
		UPDATE jobs SET
			job_title = :job_title,
			job_description = :job_description,
			job_position = :job_position,
			general_requirements = :general_requirements,
			benefits = :benefits,
			status = :status,
			published_at = :published_at,
			archived_at = :archived_at,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, model)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return job.ErrJobNotFound()
	}

	return nil
}

// GetByID retrieves a job by ID
func (r *PostgresJobRepository) GetByID(ctx context.Context, id kernel.JobID) (*job.Job, error) {
	query := `
		SELECT 
			id, job_title, job_description, job_position,
			general_requirements, benefits, posted_by, status,
			published_at, archived_at, created_at, updated_at
		FROM jobs
		WHERE id = $1
	`

	var model jobModel
	err := r.db.GetContext(ctx, &model, query, string(id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, job.ErrJobNotFound()
		}
		return nil, fmt.Errorf("failed to get job by id: %w", err)
	}

	return model.toEntity()
}

// Delete deletes a job by ID
func (r *PostgresJobRepository) Delete(ctx context.Context, id kernel.JobID) error {
	query := `DELETE FROM jobs WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, string(id))
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return job.ErrJobNotFound()
	}

	return nil
}

// List retrieves all jobs with pagination
func (r *PostgresJobRepository) List(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[job.Job], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM jobs`
	if err := r.db.GetContext(ctx, &total, countQuery); err != nil {
		return nil, fmt.Errorf("failed to count jobs: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	// Get paginated results
	query := `
		SELECT 
			id, job_title, job_description, job_position,
			general_requirements, benefits, posted_by, status,
			published_at, archived_at, created_at, updated_at
		FROM jobs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var models []jobModel
	err := r.db.SelectContext(ctx, &models, query, pagination.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	// Convert to entities
	entities := make([]job.Job, 0, len(models))
	for _, model := range models {
		entity, err := model.toEntity()
		if err != nil {
			return nil, err
		}
		entities = append(entities, *entity)
	}

	return &kernel.Paginated[job.Job]{
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

// ListByUserID retrieves jobs posted by a specific user
func (r *PostgresJobRepository) ListByUserID(ctx context.Context, userID kernel.UserID, pagination kernel.PaginationOptions) (*kernel.Paginated[job.Job], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM jobs WHERE posted_by = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, string(userID)); err != nil {
		return nil, fmt.Errorf("failed to count user jobs: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	// Get paginated results
	query := `
		SELECT 
			id, job_title, job_description, job_position,
			general_requirements, benefits, posted_by, status,
			published_at, archived_at, created_at, updated_at
		FROM jobs
		WHERE posted_by = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var models []jobModel
	err := r.db.SelectContext(ctx, &models, query, string(userID), pagination.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list user jobs: %w", err)
	}

	// Convert to entities
	entities := make([]job.Job, 0, len(models))
	for _, model := range models {
		entity, err := model.toEntity()
		if err != nil {
			return nil, err
		}
		entities = append(entities, *entity)
	}

	return &kernel.Paginated[job.Job]{
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

// Search searches jobs by various criteria
func (r *PostgresJobRepository) Search(ctx context.Context, req job.SearchJobsRequest) (*kernel.Paginated[job.Job], error) {
	// Build dynamic query
	whereConditions := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Query != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("(job_title ILIKE $%d OR job_description ILIKE $%d OR job_position ILIKE $%d)", argCount, argCount, argCount))
		args = append(args, "%"+req.Query+"%")
		argCount++
	}

	if req.Title != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("job_title ILIKE $%d", argCount))
		args = append(args, "%"+req.Title+"%")
		argCount++
	}

	if req.Position != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("job_position ILIKE $%d", argCount))
		args = append(args, "%"+req.Position+"%")
		argCount++
	}

	if req.PostedBy != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("posted_by = $%d", argCount))
		args = append(args, req.PostedBy)
		argCount++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + whereConditions[0]
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + whereConditions[i]
		}
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM jobs %s", whereClause)
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// Calculate pagination
	offset := (req.Pagination.Page - 1) * req.Pagination.PageSize
	totalPages := (total + req.Pagination.PageSize - 1) / req.Pagination.PageSize

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT 
			id, job_title, job_description, job_position,
			general_requirements, benefits, posted_by, status,
			published_at, archived_at, created_at, updated_at
		FROM jobs
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount, argCount+1)

	args = append(args, req.Pagination.PageSize, offset)

	var models []jobModel
	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search jobs: %w", err)
	}

	// Convert to entities
	entities := make([]job.Job, 0, len(models))
	for _, model := range models {
		entity, err := model.toEntity()
		if err != nil {
			return nil, err
		}
		entities = append(entities, *entity)
	}

	return &kernel.Paginated[job.Job]{
		Items: entities,
		Page: kernel.Page{
			Number: req.Pagination.Page,
			Size:   req.Pagination.PageSize,
			Total:  total,
			Pages:  totalPages,
		},
		Empty: len(entities) == 0,
	}, nil
}

// GetByTitle retrieves jobs by title (exact or partial match)
func (r *PostgresJobRepository) GetByTitle(ctx context.Context, title kernel.JobTitle) ([]*job.Job, error) {
	query := `
		SELECT 
			id, job_title, job_description, job_position,
			general_requirements, benefits, posted_by, status,
			published_at, archived_at, created_at, updated_at
		FROM jobs
		WHERE job_title ILIKE $1
		ORDER BY created_at DESC
	`

	var models []jobModel
	err := r.db.SelectContext(ctx, &models, query, "%"+string(title)+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs by title: %w", err)
	}

	// Convert to entities
	entities := make([]*job.Job, 0, len(models))
	for _, model := range models {
		entity, err := model.toEntity()
		if err != nil {
			return nil, err
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

// Exists checks if a job exists by ID
func (r *PostgresJobRepository) Exists(ctx context.Context, id kernel.JobID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM jobs WHERE id = $1)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, string(id))
	if err != nil {
		return false, fmt.Errorf("failed to check job existence: %w", err)
	}

	return exists, nil
}

// CountByUserID counts the number of jobs posted by a user
func (r *PostgresJobRepository) CountByUserID(ctx context.Context, userID kernel.UserID) (int64, error) {
	query := `SELECT COUNT(*) FROM jobs WHERE posted_by = $1`

	var count int64
	err := r.db.GetContext(ctx, &count, query, string(userID))
	if err != nil {
		return 0, fmt.Errorf("failed to count user jobs: %w", err)
	}

	return count, nil
}

// Archive archives a job
func (r *PostgresJobRepository) Archive(ctx context.Context, id kernel.JobID) error {
	query := `
		UPDATE jobs 
		SET status = 'ARCHIVED', 
		    archived_at = $1, 
		    updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, string(id))
	if err != nil {
		return fmt.Errorf("failed to archive job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return job.ErrJobNotFound()
	}

	return nil
}

// Unarchive unarchives a job
func (r *PostgresJobRepository) Unarchive(ctx context.Context, id kernel.JobID) error {
	query := `
		UPDATE jobs 
		SET status = 'DRAFT', 
		    archived_at = NULL, 
		    updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, string(id))
	if err != nil {
		return fmt.Errorf("failed to unarchive job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return job.ErrJobNotFound()
	}

	return nil
}

// ListArchived retrieves archived jobs with pagination
func (r *PostgresJobRepository) ListArchived(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[job.Job], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM jobs WHERE status = 'ARCHIVED'`
	if err := r.db.GetContext(ctx, &total, countQuery); err != nil {
		return nil, fmt.Errorf("failed to count archived jobs: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	// Get paginated results
	query := `
		SELECT 
			id, job_title, job_description, job_position,
			general_requirements, benefits, posted_by, status,
			published_at, archived_at, created_at, updated_at
		FROM jobs
		WHERE status = 'ARCHIVED'
		ORDER BY archived_at DESC
		LIMIT $1 OFFSET $2
	`

	var models []jobModel
	err := r.db.SelectContext(ctx, &models, query, pagination.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list archived jobs: %w", err)
	}

	// Convert to entities
	entities := make([]job.Job, 0, len(models))
	for _, model := range models {
		entity, err := model.toEntity()
		if err != nil {
			return nil, err
		}
		entities = append(entities, *entity)
	}

	return &kernel.Paginated[job.Job]{
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

// Publish marks a job as published/active
func (r *PostgresJobRepository) Publish(ctx context.Context, id kernel.JobID) error {
	query := `
		UPDATE jobs 
		SET status = 'PUBLISHED', 
		    published_at = $1, 
		    updated_at = $1
		WHERE id = $2 AND status = 'DRAFT'
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, string(id))
	if err != nil {
		return fmt.Errorf("failed to publish job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return job.ErrJobNotFound()
	}

	return nil
}

// Unpublish marks a job as unpublished/draft
func (r *PostgresJobRepository) Unpublish(ctx context.Context, id kernel.JobID) error {
	query := `
		UPDATE jobs 
		SET status = 'DRAFT', 
		    updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, string(id))
	if err != nil {
		return fmt.Errorf("failed to unpublish job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return job.ErrJobNotFound()
	}

	return nil
}

// ListPublished retrieves only published jobs
func (r *PostgresJobRepository) ListPublished(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[job.Job], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM jobs WHERE status = 'PUBLISHED'`
	if err := r.db.GetContext(ctx, &total, countQuery); err != nil {
		return nil, fmt.Errorf("failed to count published jobs: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	// Get paginated results
	query := `
		SELECT 
			id, job_title, job_description, job_position,
			general_requirements, benefits, posted_by, status,
			published_at, archived_at, created_at, updated_at
		FROM jobs
		WHERE status = 'PUBLISHED'
		ORDER BY published_at DESC
		LIMIT $1 OFFSET $2
	`

	var models []jobModel
	err := r.db.SelectContext(ctx, &models, query, pagination.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list published jobs: %w", err)
	}

	// Convert to entities
	entities := make([]job.Job, 0, len(models))
	for _, model := range models {
		entity, err := model.toEntity()
		if err != nil {
			return nil, err
		}
		entities = append(entities, *entity)
	}

	return &kernel.Paginated[job.Job]{
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

// CountApplications counts applications for a specific job
func (r *PostgresJobRepository) CountApplications(ctx context.Context, jobID kernel.JobID) (int64, error) {
	query := `SELECT COUNT(*) FROM applications WHERE job_id = $1`

	var count int64
	err := r.db.GetContext(ctx, &count, query, string(jobID))
	if err != nil {
		return 0, fmt.Errorf("failed to count applications: %w", err)
	}

	return count, nil
}
