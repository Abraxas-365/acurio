package jobsrv

import (
	"context"
	"time"

	"github.com/Abraxas-365/relay/pkg/errx"
	"github.com/Abraxas-365/relay/pkg/iam/auth"
	"github.com/Abraxas-365/relay/pkg/iam/user"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/job"
	"github.com/google/uuid"
)

// JobService provides business operations for jobs
type JobService struct {
	jobRepo  job.Repository
	userRepo user.UserRepository
}

// NewJobService creates a new instance of the job service
func NewJobService(
	jobRepo job.Repository,
	userRepo user.UserRepository,
) *JobService {
	return &JobService{
		jobRepo:  jobRepo,
		userRepo: userRepo,
	}
}

// CreateJob creates a new job posting
func (s *JobService) CreateJob(ctx context.Context, req job.CreateJobRequest, tenantID kernel.TenantID) (*job.Job, error) {
	// Validate that the user posting the job exists
	poster, err := s.userRepo.FindByID(ctx, req.PostedBy, tenantID)
	if err != nil {
		return nil, user.ErrUserNotFound().WithDetail("user_id", req.PostedBy.String())
	}

	// Verify user is active
	if !poster.IsActive() {
		return nil, user.ErrUserSuspended().WithDetail("user_id", req.PostedBy.String())
	}

	// Verify user has permission to post jobs
	if !poster.HasAnyScope(auth.ScopeJobsWrite, auth.ScopeJobsAll, auth.ScopeAll) {
		return nil, job.ErrInsufficientPermissions().WithDetail("required_scope", "jobs:write")
	}

	// Create new job entity
	newJob := &job.Job{
		ID:                  kernel.NewJobID(uuid.NewString()),
		Title:               req.Title,
		Description:         req.Description,
		Position:            req.Position,
		GeneralRequirements: req.GeneralRequirements,
		Benefits:            req.Benefits,
		PostedBy:            req.PostedBy,
		Status:              job.JobStatusDraft, // Start as draft
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Save job
	if err := s.jobRepo.Create(ctx, newJob); err != nil {
		return nil, errx.Wrap(err, "failed to create job", errx.TypeInternal)
	}

	return newJob, nil
}

// GetJobByID retrieves a job by ID
func (s *JobService) GetJobByID(ctx context.Context, jobID kernel.JobID) (*job.JobResponse, error) {
	jobEntity, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, job.ErrJobNotFound().WithDetail("job_id", jobID.String())
	}

	return &job.JobResponse{
		ID:                  jobEntity.ID,
		Title:               jobEntity.Title,
		Description:         jobEntity.Description,
		Position:            jobEntity.Position,
		GeneralRequirements: jobEntity.GeneralRequirements,
		Benefits:            jobEntity.Benefits,
		PostedBy:            jobEntity.PostedBy,
		Status:              jobEntity.Status,
		PublishedAt:         jobEntity.PublishedAt,
		ArchivedAt:          jobEntity.ArchivedAt,
		CreatedAt:           jobEntity.CreatedAt,
		UpdatedAt:           jobEntity.UpdatedAt,
	}, nil
}

// GetJobsByUser retrieves all jobs posted by a specific user
func (s *JobService) GetJobsByUser(ctx context.Context, userID kernel.UserID, pagination kernel.PaginationOptions) (*job.PaginatedJobsResponse, error) {
	jobs, err := s.jobRepo.ListByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to get jobs by user", errx.TypeInternal)
	}

	// Convert to response DTOs
	responses := make([]job.JobResponse, 0, len(jobs.Items))
	for _, j := range jobs.Items {
		responses = append(responses, job.JobResponse{
			ID:                  j.ID,
			Title:               j.Title,
			Description:         j.Description,
			Position:            j.Position,
			GeneralRequirements: j.GeneralRequirements,
			Benefits:            j.Benefits,
			PostedBy:            j.PostedBy,
			Status:              j.Status,
			PublishedAt:         j.PublishedAt,
			ArchivedAt:          j.ArchivedAt,
			CreatedAt:           j.CreatedAt,
			UpdatedAt:           j.UpdatedAt,
		})
	}

	return &kernel.Paginated[job.JobResponse]{
		Items: responses,
		Page:  jobs.Page,
		Empty: jobs.Empty,
	}, nil
}

// ListJobs retrieves all jobs with pagination
func (s *JobService) ListJobs(ctx context.Context, pagination kernel.PaginationOptions) (*job.PaginatedJobsResponse, error) {
	jobs, err := s.jobRepo.List(ctx, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to list jobs", errx.TypeInternal)
	}

	// Convert to response DTOs
	responses := make([]job.JobResponse, 0, len(jobs.Items))
	for _, j := range jobs.Items {
		responses = append(responses, s.toJobResponse(&j))
	}

	return &kernel.Paginated[job.JobResponse]{
		Items: responses,
		Page:  jobs.Page,
		Empty: jobs.Empty,
	}, nil
}

// ListPublishedJobs retrieves only published/active jobs
func (s *JobService) ListPublishedJobs(ctx context.Context, pagination kernel.PaginationOptions) (*job.PaginatedJobsResponse, error) {
	jobs, err := s.jobRepo.ListPublished(ctx, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to list published jobs", errx.TypeInternal)
	}

	responses := make([]job.JobResponse, 0, len(jobs.Items))
	for _, j := range jobs.Items {
		responses = append(responses, s.toJobResponse(&j))
	}

	return &kernel.Paginated[job.JobResponse]{
		Items: responses,
		Page:  jobs.Page,
		Empty: jobs.Empty,
	}, nil
}

// ListArchivedJobs retrieves archived jobs
func (s *JobService) ListArchivedJobs(ctx context.Context, pagination kernel.PaginationOptions) (*job.PaginatedJobsResponse, error) {
	jobs, err := s.jobRepo.ListArchived(ctx, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to list archived jobs", errx.TypeInternal)
	}

	responses := make([]job.JobResponse, 0, len(jobs.Items))
	for _, j := range jobs.Items {
		responses = append(responses, s.toJobResponse(&j))
	}

	return &kernel.Paginated[job.JobResponse]{
		Items: responses,
		Page:  jobs.Page,
		Empty: jobs.Empty,
	}, nil
}

// SearchJobs searches jobs by various criteria
func (s *JobService) SearchJobs(ctx context.Context, req job.SearchJobsRequest) (*job.PaginatedJobsResponse, error) {
	jobs, err := s.jobRepo.Search(ctx, req)
	if err != nil {
		return nil, errx.Wrap(err, "failed to search jobs", errx.TypeInternal)
	}

	responses := make([]job.JobResponse, 0, len(jobs.Items))
	for _, j := range jobs.Items {
		responses = append(responses, s.toJobResponse(&j))
	}

	return &kernel.Paginated[job.JobResponse]{
		Items: responses,
		Page:  jobs.Page,
		Empty: jobs.Empty,
	}, nil
}

// UpdateJob updates an existing job
func (s *JobService) UpdateJob(ctx context.Context, jobID kernel.JobID, req job.UpdateJobRequest, updaterID kernel.UserID, tenantID kernel.TenantID) (*job.Job, error) {
	// Get existing job
	jobEntity, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, job.ErrJobNotFound().WithDetail("job_id", jobID.String())
	}

	// Verify updater exists and is active
	updater, err := s.userRepo.FindByID(ctx, updaterID, tenantID)
	if err != nil {
		return nil, user.ErrUserNotFound().WithDetail("user_id", updaterID.String())
	}

	if !updater.IsActive() {
		return nil, user.ErrUserSuspended()
	}

	// Verify user has permission to update jobs
	// Must be either the poster, have jobs:write scope, or be admin
	canUpdate := updater.HasAnyScope(auth.ScopeJobsWrite, auth.ScopeJobsAll, auth.ScopeAll) ||
		jobEntity.PostedBy == updaterID

	if !canUpdate {
		return nil, job.ErrUnauthorizedUpdate().
			WithDetail("job_id", jobID.String()).
			WithDetail("user_id", updaterID.String())
	}

	// Verify job is not archived
	if jobEntity.IsArchived() {
		return nil, job.ErrJobArchived().WithDetail("job_id", jobID.String())
	}

	// Update fields if provided
	updated := false

	if req.Title != nil && *req.Title != jobEntity.Title {
		jobEntity.Title = *req.Title
		updated = true
	}

	if req.Description != nil && *req.Description != jobEntity.Description {
		jobEntity.Description = *req.Description
		updated = true
	}

	if req.Position != nil && *req.Position != jobEntity.Position {
		jobEntity.Position = *req.Position
		updated = true
	}

	if req.GeneralRequirements != nil {
		jobEntity.GeneralRequirements = *req.GeneralRequirements
		updated = true
	}

	if req.Benefits != nil {
		jobEntity.Benefits = *req.Benefits
		updated = true
	}

	if updated {
		jobEntity.UpdatedAt = time.Now()

		// Save changes
		if err := s.jobRepo.Update(ctx, jobID, jobEntity); err != nil {
			return nil, errx.Wrap(err, "failed to update job", errx.TypeInternal)
		}
	}

	return jobEntity, nil
}

// PublishJob marks a job as published/active
func (s *JobService) PublishJob(ctx context.Context, jobID kernel.JobID, publisherID kernel.UserID, tenantID kernel.TenantID) error {
	// Get job
	jobEntity, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return job.ErrJobNotFound().WithDetail("job_id", jobID.String())
	}

	// Verify publisher has permission
	publisher, err := s.userRepo.FindByID(ctx, publisherID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	canPublish := publisher.HasAnyScope(auth.ScopeJobsPublish, auth.ScopeJobsAll, auth.ScopeAll) ||
		jobEntity.PostedBy == publisherID

	if !canPublish {
		return job.ErrInsufficientPermissions().
			WithDetail("required_scope", "jobs:publish").
			WithDetail("user_id", publisherID.String())
	}

	// Business rule: Can't publish archived jobs
	if jobEntity.IsArchived() {
		return job.ErrJobArchived().
			WithDetail("job_id", jobID.String()).
			WithDetail("message", "Cannot publish an archived job")
	}

	// Business rule: Can't publish already published jobs
	if jobEntity.IsPublished() {
		return job.ErrJobAlreadyPublished().WithDetail("job_id", jobID.String())
	}

	return s.jobRepo.Publish(ctx, jobID)
}

// UnpublishJob marks a job as unpublished/draft
func (s *JobService) UnpublishJob(ctx context.Context, jobID kernel.JobID, unpublisherID kernel.UserID, tenantID kernel.TenantID) error {
	// Get job
	jobEntity, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return job.ErrJobNotFound().WithDetail("job_id", jobID.String())
	}

	// Verify unpublisher has permission
	unpublisher, err := s.userRepo.FindByID(ctx, unpublisherID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	canUnpublish := unpublisher.HasAnyScope(auth.ScopeJobsPublish, auth.ScopeJobsAll, auth.ScopeAll) ||
		jobEntity.PostedBy == unpublisherID

	if !canUnpublish {
		return job.ErrInsufficientPermissions().
			WithDetail("required_scope", "jobs:publish").
			WithDetail("user_id", unpublisherID.String())
	}

	return s.jobRepo.Unpublish(ctx, jobID)
}

// ArchiveJob archives a job
func (s *JobService) ArchiveJob(ctx context.Context, jobID kernel.JobID, archiverID kernel.UserID, tenantID kernel.TenantID) error {
	// Get job
	jobEntity, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return job.ErrJobNotFound().WithDetail("job_id", jobID.String())
	}

	// Verify archiver has permission
	archiver, err := s.userRepo.FindByID(ctx, archiverID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	canArchive := archiver.HasAnyScope(auth.ScopeJobsArchive, auth.ScopeJobsAll, auth.ScopeAll) ||
		jobEntity.PostedBy == archiverID

	if !canArchive {
		return job.ErrInsufficientPermissions().
			WithDetail("required_scope", "jobs:archive").
			WithDetail("user_id", archiverID.String())
	}

	// Business rule: Can't archive already archived jobs
	if jobEntity.IsArchived() {
		return job.ErrJobAlreadyArchived().WithDetail("job_id", jobID.String())
	}

	return s.jobRepo.Archive(ctx, jobID)
}

// UnarchiveJob unarchives a job
func (s *JobService) UnarchiveJob(ctx context.Context, jobID kernel.JobID, unarchiverID kernel.UserID, tenantID kernel.TenantID) error {
	// Get job
	jobEntity, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return job.ErrJobNotFound().WithDetail("job_id", jobID.String())
	}

	// Verify unarchiver has permission
	unarchiver, err := s.userRepo.FindByID(ctx, unarchiverID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	canUnarchive := unarchiver.HasAnyScope(auth.ScopeJobsArchive, auth.ScopeJobsAll, auth.ScopeAll) ||
		jobEntity.PostedBy == unarchiverID

	if !canUnarchive {
		return job.ErrInsufficientPermissions().
			WithDetail("required_scope", "jobs:archive").
			WithDetail("user_id", unarchiverID.String())
	}

	// Business rule: Can only unarchive archived jobs
	if !jobEntity.IsArchived() {
		return job.ErrJobNotArchived().WithDetail("job_id", jobID.String())
	}

	return s.jobRepo.Unarchive(ctx, jobID)
}

// DeleteJob deletes a job
func (s *JobService) DeleteJob(ctx context.Context, jobID kernel.JobID, deleterID kernel.UserID, tenantID kernel.TenantID) error {
	// Get job
	jobEntity, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return job.ErrJobNotFound().WithDetail("job_id", jobID.String())
	}

	// Verify deleter has permission
	deleter, err := s.userRepo.FindByID(ctx, deleterID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	canDelete := deleter.HasAnyScope(auth.ScopeJobsDelete, auth.ScopeJobsAll, auth.ScopeAll) ||
		jobEntity.PostedBy == deleterID

	if !canDelete {
		return job.ErrInsufficientPermissions().
			WithDetail("required_scope", "jobs:delete").
			WithDetail("user_id", deleterID.String())
	}

	// Business rule: Check for active applications
	applicationCount, err := s.jobRepo.CountApplications(ctx, jobID)
	if err != nil {
		// Log error but don't fail
		// logger.Warn("Failed to count applications for job", "job_id", jobID, "error", err)
	}

	if applicationCount > 0 {
		return job.ErrJobHasApplications().
			WithDetail("job_id", jobID.String()).
			WithDetail("application_count", applicationCount)
	}

	// Delete job
	if err := s.jobRepo.Delete(ctx, jobID); err != nil {
		return errx.Wrap(err, "failed to delete job", errx.TypeInternal)
	}

	return nil
}

// GetJobStats retrieves statistics for a job
func (s *JobService) GetJobStats(ctx context.Context, jobID kernel.JobID) (*job.JobStatsResponse, error) {
	jobEntity, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, job.ErrJobNotFound().WithDetail("job_id", jobID.String())
	}

	// Count applications
	applicationCount, err := s.jobRepo.CountApplications(ctx, jobID)
	if err != nil {
		applicationCount = 0 // Default to 0 on error
	}

	stats := &job.JobStatsResponse{
		JobID:             jobID,
		Title:             jobEntity.Title,
		Status:            jobEntity.Status,
		TotalApplications: applicationCount,
		IsPublished:       jobEntity.IsPublished(),
		IsArchived:        jobEntity.IsArchived(),
		CreatedAt:         jobEntity.CreatedAt,
	}

	// Calculate days since published
	if jobEntity.PublishedAt != nil {
		days := int(time.Since(*jobEntity.PublishedAt).Hours() / 24)
		stats.DaysSincePublished = &days
	}

	// Calculate days until archived
	if jobEntity.ArchivedAt != nil {
		days := int(time.Since(*jobEntity.ArchivedAt).Hours() / 24)
		stats.DaysSinceArchived = &days
	}

	return stats, nil
}

// BulkPublishJobs publishes multiple jobs
func (s *JobService) BulkPublishJobs(ctx context.Context, jobIDs []kernel.JobID, publisherID kernel.UserID, tenantID kernel.TenantID) (*job.BulkJobOperationResponse, error) {
	result := &job.BulkJobOperationResponse{
		Successful: []kernel.JobID{},
		Failed:     make(map[kernel.JobID]string),
		Total:      len(jobIDs),
	}

	for _, jobID := range jobIDs {
		if err := s.PublishJob(ctx, jobID, publisherID, tenantID); err != nil {
			result.Failed[jobID] = err.Error()
		} else {
			result.Successful = append(result.Successful, jobID)
		}
	}

	return result, nil
}

// BulkArchiveJobs archives multiple jobs
func (s *JobService) BulkArchiveJobs(ctx context.Context, jobIDs []kernel.JobID, archiverID kernel.UserID, tenantID kernel.TenantID) (*job.BulkJobOperationResponse, error) {
	result := &job.BulkJobOperationResponse{
		Successful: []kernel.JobID{},
		Failed:     make(map[kernel.JobID]string),
		Total:      len(jobIDs),
	}

	for _, jobID := range jobIDs {
		if err := s.ArchiveJob(ctx, jobID, archiverID, tenantID); err != nil {
			result.Failed[jobID] = err.Error()
		} else {
			result.Successful = append(result.Successful, jobID)
		}
	}

	return result, nil
}

// GetJobsByTitle retrieves jobs by title
func (s *JobService) GetJobsByTitle(ctx context.Context, title kernel.JobTitle) ([]*job.JobResponse, error) {
	jobs, err := s.jobRepo.GetByTitle(ctx, title)
	if err != nil {
		return nil, errx.Wrap(err, "failed to get jobs by title", errx.TypeInternal)
	}

	responses := make([]*job.JobResponse, 0, len(jobs))
	for _, j := range jobs {
		resp := s.toJobResponse(j)
		responses = append(responses, &resp)
	}

	return responses, nil
}

// CountUserJobs counts the number of jobs posted by a user
func (s *JobService) CountUserJobs(ctx context.Context, userID kernel.UserID) (int64, error) {
	count, err := s.jobRepo.CountByUserID(ctx, userID)
	if err != nil {
		return 0, errx.Wrap(err, "failed to count user jobs", errx.TypeInternal)
	}

	return count, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// toJobResponse converts a Job entity to JobResponse DTO
func (s *JobService) toJobResponse(j *job.Job) job.JobResponse {
	return job.JobResponse{
		ID:                  j.ID,
		Title:               j.Title,
		Description:         j.Description,
		Position:            j.Position,
		GeneralRequirements: j.GeneralRequirements,
		Benefits:            j.Benefits,
		PostedBy:            j.PostedBy,
		Status:              j.Status,
		PublishedAt:         j.PublishedAt,
		ArchivedAt:          j.ArchivedAt,
		CreatedAt:           j.CreatedAt,
		UpdatedAt:           j.UpdatedAt,
	}
}
