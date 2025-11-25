package applicationsrv

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/Abraxas-365/relay/pkg/errx"
	"github.com/Abraxas-365/relay/pkg/fsx"
	"github.com/Abraxas-365/relay/pkg/iam/auth"
	"github.com/Abraxas-365/relay/pkg/iam/user"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/application"
	"github.com/Abraxas-365/relay/recruitment/candidate"
	"github.com/Abraxas-365/relay/recruitment/job"
	"github.com/google/uuid"
)

// ApplicationService provides business operations for applications
type ApplicationService struct {
	applicationRepo application.Repository
	candidateRepo   candidate.Repository
	jobRepo         job.Repository
	userRepo        user.UserRepository
	fileSystem      fsx.FileSystem
}

// NewApplicationService creates a new instance of the application service
func NewApplicationService(
	applicationRepo application.Repository,
	candidateRepo candidate.Repository,
	jobRepo job.Repository,
	userRepo user.UserRepository,
	fileSystem fsx.FileSystem,
) *ApplicationService {
	return &ApplicationService{
		applicationRepo: applicationRepo,
		candidateRepo:   candidateRepo,
		jobRepo:         jobRepo,
		userRepo:        userRepo,
		fileSystem:      fileSystem,
	}
}

// CreateApplication creates a new application
func (s *ApplicationService) CreateApplication(ctx context.Context, req application.CreateApplicationRequest) (*application.Application, error) {
	// Validate candidate exists and is active
	candidateEntity, err := s.candidateRepo.GetByID(ctx, req.CandidateID)
	if err != nil {
		return nil, candidate.ErrCandidateNotFound().WithDetail("candidate_id", req.CandidateID.String())
	}

	if !candidateEntity.CanApplyToJob() {
		return nil, application.ErrCandidateCannotApply().
			WithDetail("candidate_id", req.CandidateID.String()).
			WithDetail("status", candidateEntity.Status)
	}

	// Validate job exists and is published
	jobEntity, err := s.jobRepo.GetByID(ctx, req.JobID)
	if err != nil {
		return nil, job.ErrJobNotFound().WithDetail("job_id", req.JobID.String())
	}

	if !jobEntity.IsPublished() {
		return nil, application.ErrJobNotPublished().
			WithDetail("job_id", req.JobID.String()).
			WithDetail("status", jobEntity.Status)
	}

	if jobEntity.IsArchived() {
		return nil, application.ErrJobArchived().WithDetail("job_id", req.JobID.String())
	}

	// Business rule: Check for duplicate application
	exists, err := s.applicationRepo.ExistsByJobAndCandidate(ctx, req.JobID, req.CandidateID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to check duplicate application", errx.TypeInternal)
	}

	if exists {
		return nil, application.ErrApplicationAlreadyExists().
			WithDetail("job_id", req.JobID.String()).
			WithDetail("candidate_id", req.CandidateID.String())
	}

	// Create new application entity
	newApplication := &application.Application{
		ID:              kernel.NewApplicationID(uuid.NewString()),
		JobID:           req.JobID,
		CandidateID:     req.CandidateID,
		ResumeSummary:   req.ResumeSummary,
		ResumeEmbedding: req.ResumeEmbedding,
		ResumeBucketUrl: req.ResumeBucketUrl,
		Status:          application.ApplicationStatusSubmitted,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Save application
	if err := s.applicationRepo.Create(ctx, newApplication); err != nil {
		return nil, errx.Wrap(err, "failed to create application", errx.TypeInternal)
	}

	return newApplication, nil
}

// ProcessAndUploadResume processes resume with AI and uploads to storage
// NEW METHOD: Handles AI parsing, embedding generation, and file upload
func (s *ApplicationService) ProcessAndUploadResume(
	ctx context.Context,
	applicationID kernel.ApplicationID,
	fileData []byte,
	fileName string,
	contentType string,
) error {
	// Get application
	app, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return application.ErrApplicationNotFound().WithDetail("application_id", applicationID.String())
	}

	// Validate file size (10MB max)
	if len(fileData) > 10*1024*1024 {
		return application.ErrFileSizeTooLarge().
			WithDetail("file_size", len(fileData)).
			WithDetail("max_size", 10*1024*1024)
	}

	// Process resume with AI (Vision API + Embeddings)
	processor := NewResumeProcessor(os.Getenv("OPENAI_API_KEY"))
	summary, embedding, err := processor.ProcessResume(ctx, fileData, contentType)
	if err != nil {
		return errx.Wrap(err, "failed to process resume with AI", errx.TypeInternal)
	}

	// Upload to storage
	storagePath := s.fileSystem.Join("resumes", app.ID.String(), fileName)
	if err := s.fileSystem.WriteFile(ctx, storagePath, fileData); err != nil {
		return errx.Wrap(err, "failed to upload resume", errx.TypeExternal)
	}

	// Update application with processed data
	app.ResumeSummary = summary
	app.ResumeEmbedding = embedding
	app.ResumeBucketUrl = kernel.BucketURL(storagePath)
	app.UpdatedAt = time.Now()

	if err := s.applicationRepo.Update(ctx, applicationID, app); err != nil {
		// Cleanup uploaded file on failure
		s.fileSystem.DeleteFile(context.Background(), storagePath)
		return errx.Wrap(err, "failed to update application", errx.TypeInternal)
	}

	return nil
}

// GetApplicationByID retrieves an application by ID
func (s *ApplicationService) GetApplicationByID(ctx context.Context, applicationID kernel.ApplicationID) (*application.ApplicationResponse, error) {
	app, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, application.ErrApplicationNotFound().WithDetail("application_id", applicationID.String())
	}

	return s.toApplicationResponse(app), nil
}

// GetApplicationWithDetails retrieves an application with candidate and job details
func (s *ApplicationService) GetApplicationWithDetails(ctx context.Context, applicationID kernel.ApplicationID) (*application.ApplicationWithDetailsResponse, error) {
	appWithDetails, err := s.applicationRepo.GetWithDetails(ctx, applicationID)
	if err != nil {
		return nil, application.ErrApplicationNotFound().WithDetail("application_id", applicationID.String())
	}

	return appWithDetails, nil
}

// ListApplications retrieves all applications with pagination
func (s *ApplicationService) ListApplications(ctx context.Context, pagination kernel.PaginationOptions) (*application.PaginatedApplicationsResponse, error) {
	applications, err := s.applicationRepo.List(ctx, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to list applications", errx.TypeInternal)
	}

	responses := make([]application.ApplicationResponse, 0, len(applications.Items))
	for _, app := range applications.Items {
		responses = append(responses, *s.toApplicationResponse(&app))
	}

	return &kernel.Paginated[application.ApplicationResponse]{
		Items: responses,
		Page:  applications.Page,
		Empty: applications.Empty,
	}, nil
}

// ListApplicationsByJob retrieves applications for a specific job
func (s *ApplicationService) ListApplicationsByJob(ctx context.Context, jobID kernel.JobID, pagination kernel.PaginationOptions) (*application.PaginatedApplicationsResponse, error) {
	// Verify job exists
	if err := s.validateJobExists(ctx, jobID); err != nil {
		return nil, err
	}

	applications, err := s.applicationRepo.ListByJobID(ctx, jobID, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to list applications by job", errx.TypeInternal)
	}

	responses := make([]application.ApplicationResponse, 0, len(applications.Items))
	for _, app := range applications.Items {
		responses = append(responses, *s.toApplicationResponse(&app))
	}

	return &kernel.Paginated[application.ApplicationResponse]{
		Items: responses,
		Page:  applications.Page,
		Empty: applications.Empty,
	}, nil
}

// ListApplicationsByJobWithDetails retrieves applications with details for a specific job
func (s *ApplicationService) ListApplicationsByJobWithDetails(ctx context.Context, jobID kernel.JobID, pagination kernel.PaginationOptions) (*application.PaginatedApplicationsWithDetailsResponse, error) {
	// Verify job exists
	if err := s.validateJobExists(ctx, jobID); err != nil {
		return nil, err
	}

	applications, err := s.applicationRepo.ListWithDetailsByJobID(ctx, jobID, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to list applications with details by job", errx.TypeInternal)
	}

	return applications, nil
}

// ListApplicationsByCandidate retrieves applications for a specific candidate
func (s *ApplicationService) ListApplicationsByCandidate(ctx context.Context, candidateID kernel.CandidateID, pagination kernel.PaginationOptions) (*application.PaginatedApplicationsResponse, error) {
	// Verify candidate exists
	if err := s.validateCandidateExists(ctx, candidateID); err != nil {
		return nil, err
	}

	applications, err := s.applicationRepo.ListByCandidateID(ctx, candidateID, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to list applications by candidate", errx.TypeInternal)
	}

	responses := make([]application.ApplicationResponse, 0, len(applications.Items))
	for _, app := range applications.Items {
		responses = append(responses, *s.toApplicationResponse(&app))
	}

	return &kernel.Paginated[application.ApplicationResponse]{
		Items: responses,
		Page:  applications.Page,
		Empty: applications.Empty,
	}, nil
}

// ListApplicationsByCandidateWithDetails retrieves applications with details for a specific candidate
func (s *ApplicationService) ListApplicationsByCandidateWithDetails(ctx context.Context, candidateID kernel.CandidateID, pagination kernel.PaginationOptions) (*application.PaginatedApplicationsWithDetailsResponse, error) {
	// Verify candidate exists
	if err := s.validateCandidateExists(ctx, candidateID); err != nil {
		return nil, err
	}

	applications, err := s.applicationRepo.ListWithDetailsByCandidateID(ctx, candidateID, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to list applications with details by candidate", errx.TypeInternal)
	}

	return applications, nil
}

// UpdateApplication updates an existing application
func (s *ApplicationService) UpdateApplication(ctx context.Context, applicationID kernel.ApplicationID, req application.UpdateApplicationRequest, updaterID kernel.UserID, tenantID kernel.TenantID) (*application.Application, error) {
	// Get existing application
	app, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, application.ErrApplicationNotFound().WithDetail("application_id", applicationID.String())
	}

	// Verify updater has permission
	updater, err := s.userRepo.FindByID(ctx, updaterID, tenantID)
	if err != nil {
		return nil, user.ErrUserNotFound()
	}

	if !updater.HasAnyScope(auth.ScopeApplicationsWrite, auth.ScopeApplicationsAll, auth.ScopeAll) {
		return nil, application.ErrInsufficientPermissions().
			WithDetail("required_scope", "applications:write").
			WithDetail("user_id", updaterID.String())
	}

	// Business rule: Can't update archived applications
	if app.IsArchived() {
		return nil, application.ErrApplicationArchived().WithDetail("application_id", applicationID.String())
	}

	// Track if any changes were made
	updated := false

	// Update fields if provided
	if req.ResumeSummary != nil && *req.ResumeSummary != app.ResumeSummary {
		app.ResumeSummary = *req.ResumeSummary
		updated = true
	}

	if req.ResumeEmbedding != nil {
		app.ResumeEmbedding = *req.ResumeEmbedding
		updated = true
	}

	if req.ResumeBucketUrl != nil && *req.ResumeBucketUrl != app.ResumeBucketUrl {
		app.ResumeBucketUrl = *req.ResumeBucketUrl
		updated = true
	}

	if updated {
		app.UpdatedAt = time.Now()

		// Save changes
		if err := s.applicationRepo.Update(ctx, applicationID, app); err != nil {
			return nil, errx.Wrap(err, "failed to update application", errx.TypeInternal)
		}
	}

	return app, nil
}

// UploadResume uploads a resume file for an application
func (s *ApplicationService) UploadResume(ctx context.Context, req application.UploadResumeRequest, uploaderID kernel.UserID, tenantID kernel.TenantID) (*application.UploadResumeResponse, error) {
	// Verify uploader has permission
	uploader, err := s.userRepo.FindByID(ctx, uploaderID, tenantID)
	if err != nil {
		return nil, user.ErrUserNotFound()
	}

	if !uploader.HasAnyScope(auth.ScopeApplicationsWrite, auth.ScopeApplicationsAll, auth.ScopeAll) {
		return nil, application.ErrInsufficientPermissions().
			WithDetail("required_scope", "applications:write")
	}

	// Validate application exists
	app, err := s.applicationRepo.GetByID(ctx, req.ApplicationID)
	if err != nil {
		return nil, application.ErrApplicationNotFound().WithDetail("application_id", req.ApplicationID.String())
	}

	// Validate file size (10MB max)
	if req.FileSize > 10*1024*1024 {
		return nil, application.ErrFileSizeTooLarge().
			WithDetail("file_size", req.FileSize).
			WithDetail("max_size", 10*1024*1024)
	}

	// Validate content type (PDF only for now)
	allowedTypes := map[string]bool{
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	}

	if !allowedTypes[req.ContentType] {
		return nil, application.ErrInvalidFileType().
			WithDetail("content_type", req.ContentType).
			WithDetail("allowed_types", "pdf, doc, docx")
	}

	// Generate storage path: resumes/{application_id}/{filename}
	storagePath := s.fileSystem.Join("resumes", app.ID.String(), req.FileName)

	// Upload file to storage
	if err := s.fileSystem.WriteFile(ctx, storagePath, req.FileData); err != nil {
		return nil, errx.Wrap(err, "failed to upload resume", errx.TypeExternal).
			WithDetail("path", storagePath)
	}

	// Update application with bucket URL
	bucketURL := kernel.BucketURL(storagePath)
	if err := s.applicationRepo.UpdateResumeBucketUrl(ctx, app.ID, bucketURL); err != nil {
		// Attempt to clean up uploaded file
		s.fileSystem.DeleteFile(context.Background(), storagePath)
		return nil, errx.Wrap(err, "failed to update application with resume URL", errx.TypeInternal)
	}

	return &application.UploadResumeResponse{
		ApplicationID: app.ID,
		BucketURL:     bucketURL,
		FileName:      req.FileName,
		FileSize:      req.FileSize,
		UploadedAt:    time.Now(),
		UploadedBy:    uploaderID,
	}, nil
}

// DownloadResume downloads a resume file for an application
func (s *ApplicationService) DownloadResume(ctx context.Context, applicationID kernel.ApplicationID, downloaderID kernel.UserID, tenantID kernel.TenantID) (io.ReadCloser, string, error) {
	// Verify downloader has permission
	downloader, err := s.userRepo.FindByID(ctx, downloaderID, tenantID)
	if err != nil {
		return nil, "", user.ErrUserNotFound()
	}

	if !downloader.HasAnyScope(auth.ScopeApplicationsRead, auth.ScopeApplicationsAll, auth.ScopeAll) {
		return nil, "", application.ErrInsufficientPermissions().
			WithDetail("required_scope", "applications:read")
	}

	// Get application
	app, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, "", application.ErrApplicationNotFound().WithDetail("application_id", applicationID.String())
	}

	if app.ResumeBucketUrl == "" {
		return nil, "", application.ErrResumeNotFound().WithDetail("application_id", applicationID.String())
	}

	// Get file stream from storage
	stream, err := s.fileSystem.ReadFileStream(ctx, string(app.ResumeBucketUrl))
	if err != nil {
		return nil, "", errx.Wrap(err, "failed to download resume", errx.TypeExternal).
			WithDetail("bucket_url", app.ResumeBucketUrl)
	}

	// Extract filename from bucket URL
	filename := extractFilename(string(app.ResumeBucketUrl))

	return stream, filename, nil
}

// AssignReviewer assigns a reviewer to an application
func (s *ApplicationService) AssignReviewer(ctx context.Context, applicationID kernel.ApplicationID, reviewerID kernel.UserID, assignerID kernel.UserID, tenantID kernel.TenantID) error {
	// Verify assigner has permission
	assigner, err := s.userRepo.FindByID(ctx, assignerID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	if !assigner.HasAnyScope(auth.ScopeApplicationsAssign, auth.ScopeApplicationsAll, auth.ScopeAll) {
		return application.ErrInsufficientPermissions().
			WithDetail("required_scope", "applications:assign")
	}

	// Verify application exists
	app, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return application.ErrApplicationNotFound().WithDetail("application_id", applicationID.String())
	}

	// Verify reviewer exists and has review permission
	reviewer, err := s.userRepo.FindByID(ctx, reviewerID, tenantID)
	if err != nil {
		return user.ErrUserNotFound().WithDetail("user_id", reviewerID.String())
	}

	if !reviewer.IsActive() {
		return user.ErrUserSuspended().WithDetail("user_id", reviewerID.String())
	}

	if !reviewer.HasAnyScope(auth.ScopeApplicationsReview, auth.ScopeApplicationsAll, auth.ScopeAll) {
		return application.ErrReviewerInvalidPermissions().
			WithDetail("reviewer_id", reviewerID.String()).
			WithDetail("required_scope", "applications:review")
	}

	// Business rule: Can't assign to archived applications
	if app.IsArchived() {
		return application.ErrApplicationArchived().WithDetail("application_id", applicationID.String())
	}

	// Assign reviewer
	if err := s.applicationRepo.AssignReviewer(ctx, applicationID, reviewerID); err != nil {
		return errx.Wrap(err, "failed to assign reviewer", errx.TypeInternal)
	}

	return nil
}

// GetApplicationsByReviewer retrieves applications assigned to a reviewer
func (s *ApplicationService) GetApplicationsByReviewer(ctx context.Context, reviewerID kernel.UserID, pagination kernel.PaginationOptions) (*application.PaginatedApplicationsResponse, error) {
	applications, err := s.applicationRepo.GetApplicationsByReviewer(ctx, reviewerID, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to get applications by reviewer", errx.TypeInternal)
	}

	responses := make([]application.ApplicationResponse, 0, len(applications.Items))
	for _, app := range applications.Items {
		responses = append(responses, *s.toApplicationResponse(&app))
	}

	return &kernel.Paginated[application.ApplicationResponse]{
		Items: responses,
		Page:  applications.Page,
		Empty: applications.Empty,
	}, nil
}

// UpdateApplicationStatus updates the status of an application
func (s *ApplicationService) UpdateApplicationStatus(ctx context.Context, applicationID kernel.ApplicationID, newStatus application.ApplicationStatus, updaterID kernel.UserID, tenantID kernel.TenantID) error {
	// Verify updater has permission
	updater, err := s.userRepo.FindByID(ctx, updaterID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	// Status changes require different permissions
	requiredScope := auth.ScopeApplicationsWrite
	if newStatus == application.ApplicationStatusApproved || newStatus == application.ApplicationStatusRejected {
		requiredScope = auth.ScopeApplicationsApprove
	}

	if !updater.HasAnyScope(requiredScope, auth.ScopeApplicationsAll, auth.ScopeAll) {
		return application.ErrInsufficientPermissions().
			WithDetail("required_scope", requiredScope)
	}

	// Get application
	app, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return application.ErrApplicationNotFound().WithDetail("application_id", applicationID.String())
	}

	// Update status
	if err := app.UpdateStatus(newStatus); err != nil {
		return err
	}

	// Save changes
	if err := s.applicationRepo.Update(ctx, applicationID, app); err != nil {
		return errx.Wrap(err, "failed to update application status", errx.TypeInternal)
	}

	return nil
}

// ArchiveApplication archives an application
func (s *ApplicationService) ArchiveApplication(ctx context.Context, applicationID kernel.ApplicationID, archiverID kernel.UserID, tenantID kernel.TenantID) error {
	// Verify archiver has permission
	archiver, err := s.userRepo.FindByID(ctx, archiverID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	if !archiver.HasAnyScope(auth.ScopeApplicationsWrite, auth.ScopeApplicationsAll, auth.ScopeAll) {
		return application.ErrInsufficientPermissions().
			WithDetail("required_scope", "applications:write")
	}

	// Archive application
	if err := s.applicationRepo.Archive(ctx, applicationID); err != nil {
		return errx.Wrap(err, "failed to archive application", errx.TypeInternal)
	}

	return nil
}

// UnarchiveApplication unarchives an application
func (s *ApplicationService) UnarchiveApplication(ctx context.Context, applicationID kernel.ApplicationID, unarchiverID kernel.UserID, tenantID kernel.TenantID) error {
	// Verify unarchiver has permission
	unarchiver, err := s.userRepo.FindByID(ctx, unarchiverID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	if !unarchiver.HasAnyScope(auth.ScopeApplicationsWrite, auth.ScopeApplicationsAll, auth.ScopeAll) {
		return application.ErrInsufficientPermissions().
			WithDetail("required_scope", "applications:write")
	}

	// Unarchive application
	if err := s.applicationRepo.Unarchive(ctx, applicationID); err != nil {
		return errx.Wrap(err, "failed to unarchive application", errx.TypeInternal)
	}

	return nil
}

// DeleteApplication deletes an application
func (s *ApplicationService) DeleteApplication(ctx context.Context, applicationID kernel.ApplicationID, deleterID kernel.UserID, tenantID kernel.TenantID) error {
	// Verify deleter has permission
	deleter, err := s.userRepo.FindByID(ctx, deleterID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	if !deleter.HasAnyScope(auth.ScopeApplicationsDelete, auth.ScopeApplicationsAll, auth.ScopeAll) {
		return application.ErrInsufficientPermissions().
			WithDetail("required_scope", "applications:delete")
	}

	// Get application
	app, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return application.ErrApplicationNotFound().WithDetail("application_id", applicationID.String())
	}

	// Delete resume file if exists
	if app.ResumeBucketUrl != "" {
		if err := s.fileSystem.DeleteFile(ctx, string(app.ResumeBucketUrl)); err != nil {
			// Log error but don't fail
			// logger.Warn("Failed to delete resume file", "bucket_url", app.ResumeBucketUrl, "error", err)
		}
	}

	// Delete application
	if err := s.applicationRepo.Delete(ctx, applicationID); err != nil {
		return errx.Wrap(err, "failed to delete application", errx.TypeInternal)
	}

	return nil
}

// GetApplicationStats retrieves statistics for an application
func (s *ApplicationService) GetApplicationStats(ctx context.Context, applicationID kernel.ApplicationID) (*application.ApplicationStatsResponse, error) {
	app, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, application.ErrApplicationNotFound().WithDetail("application_id", applicationID.String())
	}

	stats := &application.ApplicationStatsResponse{
		ApplicationID: applicationID,
		Status:        app.Status,
		IsArchived:    app.IsArchived(),
		HasResume:     app.ResumeBucketUrl != "",
		HasReviewer:   app.ReviewerID != nil,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
	}

	// Calculate days since submission
	days := int(time.Since(app.CreatedAt).Hours() / 24)
	stats.DaysSinceSubmission = days

	// Calculate days since last update
	daysSinceUpdate := int(time.Since(app.UpdatedAt).Hours() / 24)
	stats.DaysSinceLastUpdate = daysSinceUpdate

	// Days since status change
	if app.StatusChangedAt != nil {
		daysSinceStatusChange := int(time.Since(*app.StatusChangedAt).Hours() / 24)
		stats.DaysSinceStatusChange = &daysSinceStatusChange
	}

	return stats, nil
}

// BulkArchiveApplications archives multiple applications
func (s *ApplicationService) BulkArchiveApplications(ctx context.Context, applicationIDs []kernel.ApplicationID, archiverID kernel.UserID, tenantID kernel.TenantID) (*application.BulkApplicationOperationResponse, error) {
	result := &application.BulkApplicationOperationResponse{
		Successful: []kernel.ApplicationID{},
		Failed:     make(map[kernel.ApplicationID]string),
		Total:      len(applicationIDs),
	}

	for _, appID := range applicationIDs {
		if err := s.ArchiveApplication(ctx, appID, archiverID, tenantID); err != nil {
			result.Failed[appID] = err.Error()
		} else {
			result.Successful = append(result.Successful, appID)
		}
	}

	return result, nil
}

// BulkUpdateStatus updates status for multiple applications
func (s *ApplicationService) BulkUpdateStatus(ctx context.Context, applicationIDs []kernel.ApplicationID, newStatus application.ApplicationStatus, updaterID kernel.UserID, tenantID kernel.TenantID) (*application.BulkApplicationOperationResponse, error) {
	result := &application.BulkApplicationOperationResponse{
		Successful: []kernel.ApplicationID{},
		Failed:     make(map[kernel.ApplicationID]string),
		Total:      len(applicationIDs),
	}

	for _, appID := range applicationIDs {
		if err := s.UpdateApplicationStatus(ctx, appID, newStatus, updaterID, tenantID); err != nil {
			result.Failed[appID] = err.Error()
		} else {
			result.Successful = append(result.Successful, appID)
		}
	}

	return result, nil
}

// ============================================================================
// Validation Helper Methods
// ============================================================================

// validateJobExists checks if a job exists
func (s *ApplicationService) validateJobExists(ctx context.Context, jobID kernel.JobID) error {
	exists, err := s.jobRepo.Exists(ctx, jobID)
	if err != nil {
		return errx.Wrap(err, "failed to validate job existence", errx.TypeInternal)
	}

	if !exists {
		return job.ErrJobNotFound().WithDetail("job_id", jobID.String())
	}

	return nil
}

// validateCandidateExists checks if a candidate exists
func (s *ApplicationService) validateCandidateExists(ctx context.Context, candidateID kernel.CandidateID) error {
	exists, err := s.candidateRepo.Exists(ctx, candidateID)
	if err != nil {
		return errx.Wrap(err, "failed to validate candidate existence", errx.TypeInternal)
	}

	if !exists {
		return candidate.ErrCandidateNotFound().WithDetail("candidate_id", candidateID.String())
	}

	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// toApplicationResponse converts an Application entity to ApplicationResponse DTO
func (s *ApplicationService) toApplicationResponse(app *application.Application) *application.ApplicationResponse {
	return &application.ApplicationResponse{
		ID:              app.ID,
		JobID:           app.JobID,
		CandidateID:     app.CandidateID,
		ResumeSummary:   app.ResumeSummary,
		ResumeEmbedding: app.ResumeEmbedding,
		ResumeBucketUrl: app.ResumeBucketUrl,
		Status:          app.Status,
		ReviewerID:      app.ReviewerID,
		SubmittedBy:     app.SubmittedBy,
		CreatedAt:       app.CreatedAt,
		UpdatedAt:       app.UpdatedAt,
	}
}

// extractFilename extracts filename from bucket URL path
func extractFilename(path string) string {
	parts := []rune(path)
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '/' {
			return string(parts[i+1:])
		}
	}
	return path
}
