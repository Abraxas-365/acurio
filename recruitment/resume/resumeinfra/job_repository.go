package resumeinfra

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/pkg/logx"
	"github.com/Abraxas-365/relay/recruitment/resume"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PostgresJobRepository struct {
	db *sqlx.DB
}

func NewPostgresJobRepository(db *sqlx.DB) resume.JobRepository {
	return &PostgresJobRepository{db: db}
}

// dbJob is the database model with proper JSON handling
type dbJob struct {
	ID       string  `db:"id"`
	TenantID string  `db:"tenant_id"`
	ResumeID *string `db:"resume_id"`

	Status   string `db:"status"`
	FilePath string `db:"file_path"`
	FileName string `db:"file_name"`
	FileType string `db:"file_type"`
	Title    string `db:"title"`

	AttemptCount int `db:"attempt_count"`
	MaxAttempts  int `db:"max_attempts"`

	ErrorMessage string         `db:"error_message"`
	ErrorDetails sql.NullString `db:"error_details"`

	CurrentStep        *string `db:"current_step"`
	ProgressPercentage int     `db:"progress_percentage"`

	CreatedAt   time.Time  `db:"created_at"`
	StartedAt   *time.Time `db:"started_at"`
	CompletedAt *time.Time `db:"completed_at"`
	FailedAt    *time.Time `db:"failed_at"`
	NextRetryAt *time.Time `db:"next_retry_at"`

	RequestPayload string `db:"request_payload"`
}

// Create creates a new job record
func (r *PostgresJobRepository) Create(ctx context.Context, job *resume.ResumeProcessingJob) error {
	query := `
		INSERT INTO resume_processing_jobs (
			id, tenant_id, resume_id, status, file_path, file_name, file_type, title,
			attempt_count, max_attempts, error_message, error_details,
			current_step, progress_percentage,
			created_at, started_at, completed_at, failed_at, next_retry_at,
			request_payload
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14,
			$15, $16, $17, $18, $19,
			$20
		)
	`

	dbJob, err := r.toDBJob(job)
	if err != nil {
		return fmt.Errorf("convert to db job: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		dbJob.ID, dbJob.TenantID, dbJob.ResumeID, dbJob.Status,
		dbJob.FilePath, dbJob.FileName, dbJob.FileType, dbJob.Title,
		dbJob.AttemptCount, dbJob.MaxAttempts, dbJob.ErrorMessage, dbJob.ErrorDetails,
		dbJob.CurrentStep, dbJob.ProgressPercentage,
		dbJob.CreatedAt, dbJob.StartedAt, dbJob.CompletedAt, dbJob.FailedAt, dbJob.NextRetryAt,
		dbJob.RequestPayload,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("job already exists: %w", err)
		}
		return fmt.Errorf("create job: %w", err)
	}

	logx.Infof("Created job: %s", job.ID)
	return nil
}

// Update updates an existing job
func (r *PostgresJobRepository) Update(ctx context.Context, job *resume.ResumeProcessingJob) error {
	query := `
		UPDATE resume_processing_jobs SET
			resume_id = $2,
			status = $3,
			attempt_count = $4,
			error_message = $5,
			error_details = $6,
			current_step = $7,
			progress_percentage = $8,
			started_at = $9,
			completed_at = $10,
			failed_at = $11,
			next_retry_at = $12,
			request_payload = $13
		WHERE id = $1
	`

	dbJob, err := r.toDBJob(job)
	if err != nil {
		return fmt.Errorf("convert to db job: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		dbJob.ID,
		dbJob.ResumeID,
		dbJob.Status,
		dbJob.AttemptCount,
		dbJob.ErrorMessage,
		dbJob.ErrorDetails,
		dbJob.CurrentStep,
		dbJob.ProgressPercentage,
		dbJob.StartedAt,
		dbJob.CompletedAt,
		dbJob.FailedAt,
		dbJob.NextRetryAt,
		dbJob.RequestPayload,
	)

	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("job not found: %s", job.ID)
	}

	return nil
}

// GetByID retrieves a job by ID
func (r *PostgresJobRepository) GetByID(ctx context.Context, jobID kernel.JobID) (*resume.ResumeProcessingJob, error) {
	query := `
		SELECT 
			id, tenant_id, resume_id, status, file_path, file_name, file_type, title,
			attempt_count, max_attempts, error_message, error_details,
			current_step, progress_percentage,
			created_at, started_at, completed_at, failed_at, next_retry_at,
			request_payload
		FROM resume_processing_jobs
		WHERE id = $1
	`

	var dbJob dbJob
	err := r.db.GetContext(ctx, &dbJob, query, jobID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("get job: %w", err)
	}

	return r.toDomainJob(&dbJob)
}

// GetByTenantID retrieves all jobs for a tenant with pagination
func (r *PostgresJobRepository) GetByTenantID(
	ctx context.Context,
	tenantID kernel.TenantID,
	pagination kernel.PaginationOptions,
) (*kernel.Paginated[resume.ResumeProcessingJob], error) {
	countQuery := `SELECT COUNT(*) FROM resume_processing_jobs WHERE tenant_id = $1`
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID.String()); err != nil {
		return nil, fmt.Errorf("count jobs: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	query := `
		SELECT 
			id, tenant_id, resume_id, status, file_path, file_name, file_type, title,
			attempt_count, max_attempts, error_message, error_details,
			current_step, progress_percentage,
			created_at, started_at, completed_at, failed_at, next_retry_at,
			request_payload
		FROM resume_processing_jobs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var dbJobs []dbJob
	if err := r.db.SelectContext(ctx, &dbJobs, query, tenantID.String(), pagination.PageSize, offset); err != nil {
		return nil, fmt.Errorf("get jobs: %w", err)
	}

	jobs := make([]resume.ResumeProcessingJob, 0, len(dbJobs))
	for _, dbJob := range dbJobs {
		job, err := r.toDomainJob(&dbJob)
		if err != nil {
			logx.Errorf("Failed to convert job %s: %v", dbJob.ID, err)
			continue
		}
		jobs = append(jobs, *job)
	}

	return &kernel.Paginated[resume.ResumeProcessingJob]{
		Items: jobs,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
		},
	}, nil
}

// GetFailedJobsForRetry retrieves failed jobs that are ready for retry
func (r *PostgresJobRepository) GetFailedJobsForRetry(ctx context.Context, limit int) ([]*resume.ResumeProcessingJob, error) {
	query := `
		SELECT 
			id, tenant_id, resume_id, status, file_path, file_name, file_type, title,
			attempt_count, max_attempts, error_message, error_details,
			current_step, progress_percentage,
			created_at, started_at, completed_at, failed_at, next_retry_at,
			request_payload
		FROM resume_processing_jobs
		WHERE status = $1 
			AND next_retry_at IS NOT NULL 
			AND next_retry_at <= $2
			AND attempt_count < max_attempts
		ORDER BY next_retry_at ASC
		LIMIT $3
	`

	var dbJobs []dbJob
	err := r.db.SelectContext(ctx, &dbJobs, query, string(resume.JobStatusFailed), time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("get failed jobs: %w", err)
	}

	jobs := make([]*resume.ResumeProcessingJob, 0, len(dbJobs))
	for _, dbJob := range dbJobs {
		job, err := r.toDomainJob(&dbJob)
		if err != nil {
			logx.Errorf("Failed to convert job %s: %v", dbJob.ID, err)
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// MarkAsProcessing marks a job as processing
func (r *PostgresJobRepository) MarkAsProcessing(ctx context.Context, jobID kernel.JobID) error {
	query := `
		UPDATE resume_processing_jobs 
		SET status = $2, started_at = $3
		WHERE id = $1 AND status = $4
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		jobID.String(),
		string(resume.JobStatusProcessing),
		now,
		string(resume.JobStatusPending),
	)

	if err != nil {
		return fmt.Errorf("mark as processing: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("job not found or not in pending status: %s", jobID)
	}

	logx.Infof("Marked job as processing: %s", jobID)
	return nil
}

// MarkAsCompleted marks a job as completed
func (r *PostgresJobRepository) MarkAsCompleted(ctx context.Context, jobID kernel.JobID, resumeID kernel.ResumeID) error {
	query := `
		UPDATE resume_processing_jobs 
		SET 
			status = $2, 
			resume_id = $3, 
			completed_at = $4,
			progress_percentage = 100,
			error_message = '',
			error_details = NULL,
			next_retry_at = NULL
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		jobID.String(),
		string(resume.JobStatusCompleted),
		resumeID.String(),
		now,
	)

	if err != nil {
		return fmt.Errorf("mark as completed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	logx.Infof("Marked job as completed: %s, ResumeID: %s", jobID, resumeID)
	return nil
}

// MarkAsFailed marks a job as failed
func (r *PostgresJobRepository) MarkAsFailed(
	ctx context.Context,
	jobID kernel.JobID,
	errorMsg string,
	errorDetails map[string]interface{},
) error {
	var errorDetailsJSON sql.NullString
	if errorDetails != nil && len(errorDetails) > 0 {
		jsonBytes, err := json.Marshal(errorDetails)
		if err != nil {
			logx.Warnf("Failed to marshal error details: %v", err)
		} else {
			errorDetailsJSON = sql.NullString{
				String: string(jsonBytes),
				Valid:  true,
			}
		}
	}

	query := `
		UPDATE resume_processing_jobs 
		SET 
			status = $2, 
			failed_at = $3,
			error_message = $4,
			error_details = $5
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		jobID.String(),
		string(resume.JobStatusFailed),
		now,
		errorMsg,
		errorDetailsJSON,
	)

	if err != nil {
		return fmt.Errorf("mark as failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	logx.Warnf("Marked job as failed: %s, Error: %s", jobID, errorMsg)
	return nil
}

// UpdateProgress updates the progress of a job
func (r *PostgresJobRepository) UpdateProgress(
	ctx context.Context,
	jobID kernel.JobID,
	step resume.ProcessingStep,
	percentage int,
) error {
	query := `
		UPDATE resume_processing_jobs 
		SET 
			current_step = $2,
			progress_percentage = $3
		WHERE id = $1
	`

	stepStr := string(step)
	result, err := r.db.ExecContext(ctx, query,
		jobID.String(),
		stepStr,
		percentage,
	)

	if err != nil {
		return fmt.Errorf("update progress: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// toDBJob converts domain model to database model
func (r *PostgresJobRepository) toDBJob(job *resume.ResumeProcessingJob) (*dbJob, error) {
	requestPayloadJSON, err := json.Marshal(job.RequestPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal request payload: %w", err)
	}

	var errorDetails sql.NullString
	if job.ErrorDetails != nil && len(job.ErrorDetails) > 0 {
		errorDetailsJSON, err := json.Marshal(job.ErrorDetails)
		if err != nil {
			logx.Warnf("Failed to marshal error details: %v", err)
		} else {
			errorDetails = sql.NullString{
				String: string(errorDetailsJSON),
				Valid:  true,
			}
		}
	}

	var currentStep *string
	if job.CurrentStep != nil {
		stepStr := string(*job.CurrentStep)
		currentStep = &stepStr
	}

	var resumeID *string
	if job.ResumeID != nil {
		idStr := job.ResumeID.String()
		resumeID = &idStr
	}

	return &dbJob{
		ID:                 job.ID.String(),
		TenantID:           job.TenantID.String(),
		ResumeID:           resumeID,
		Status:             string(job.Status),
		FilePath:           job.FilePath,
		FileName:           job.FileName,
		FileType:           job.FileType,
		Title:              job.Title,
		AttemptCount:       job.AttemptCount,
		MaxAttempts:        job.MaxAttempts,
		ErrorMessage:       job.ErrorMessage,
		ErrorDetails:       errorDetails,
		CurrentStep:        currentStep,
		ProgressPercentage: job.ProgressPercentage,
		CreatedAt:          job.CreatedAt,
		StartedAt:          job.StartedAt,
		CompletedAt:        job.CompletedAt,
		FailedAt:           job.FailedAt,
		NextRetryAt:        job.NextRetryAt,
		RequestPayload:     string(requestPayloadJSON),
	}, nil
}

// toDomainJob converts database model to domain model
func (r *PostgresJobRepository) toDomainJob(dbJob *dbJob) (*resume.ResumeProcessingJob, error) {
	var requestPayload resume.ParseResumeRequest
	if err := json.Unmarshal([]byte(dbJob.RequestPayload), &requestPayload); err != nil {
		return nil, fmt.Errorf("unmarshal request payload: %w", err)
	}

	var errorDetails map[string]interface{}
	if dbJob.ErrorDetails.Valid && dbJob.ErrorDetails.String != "" {
		if err := json.Unmarshal([]byte(dbJob.ErrorDetails.String), &errorDetails); err != nil {
			logx.Warnf("Failed to unmarshal error details for job %s: %v", dbJob.ID, err)
			errorDetails = nil
		}
	}

	var currentStep *resume.ProcessingStep
	if dbJob.CurrentStep != nil {
		step := resume.ProcessingStep(*dbJob.CurrentStep)
		currentStep = &step
	}

	var resumeID *kernel.ResumeID
	if dbJob.ResumeID != nil {
		id := kernel.ResumeID(*dbJob.ResumeID)
		resumeID = &id
	}

	return &resume.ResumeProcessingJob{
		ID:                 kernel.JobID(dbJob.ID),
		TenantID:           kernel.TenantID(dbJob.TenantID),
		ResumeID:           resumeID,
		Status:             resume.JobStatus(dbJob.Status),
		FilePath:           dbJob.FilePath,
		FileName:           dbJob.FileName,
		FileType:           dbJob.FileType,
		Title:              dbJob.Title,
		AttemptCount:       dbJob.AttemptCount,
		MaxAttempts:        dbJob.MaxAttempts,
		ErrorMessage:       dbJob.ErrorMessage,
		ErrorDetails:       errorDetails,
		CurrentStep:        currentStep,
		ProgressPercentage: dbJob.ProgressPercentage,
		CreatedAt:          dbJob.CreatedAt,
		StartedAt:          dbJob.StartedAt,
		CompletedAt:        dbJob.CompletedAt,
		FailedAt:           dbJob.FailedAt,
		NextRetryAt:        dbJob.NextRetryAt,
		RequestPayload:     requestPayload,
	}, nil
}
