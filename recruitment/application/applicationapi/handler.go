package applicationapi

import (
	"io"

	"github.com/Abraxas-365/relay/pkg/iam/auth"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/application"
	"github.com/Abraxas-365/relay/recruitment/application/applicationsrv"
	"github.com/gofiber/fiber/v2"
)

// Handlers provides HTTP handlers for application operations
type Handlers struct {
	service *applicationsrv.ApplicationService
}

// NewHandlers creates a new application handlers instance
func NewHandlers(service *applicationsrv.ApplicationService) *Handlers {
	return &Handlers{
		service: service,
	}
}

// CreateApplication creates a new application
// POST /api/applications
func (h *Handlers) CreateApplication(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse request body
	var req application.CreateApplicationRequest
	if err := c.BodyParser(&req); err != nil {
		return application.ErrInvalidRequest().WithDetail("parse_error", err.Error())
	}

	req.CreatedBy = authContext.UserID

	// Create application
	newApplication, err := h.service.CreateApplication(
		c.Context(),
		req,
	)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(newApplication)
}

// GetApplicationByID retrieves an application by ID
// GET /api/applications/:id
func (h *Handlers) GetApplicationByID(c *fiber.Ctx) error {
	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Get application
	app, err := h.service.GetApplicationByID(c.Context(), applicationID)
	if err != nil {
		return err
	}

	return c.JSON(app)
}

// GetApplicationWithDetails retrieves an application with candidate and job details
// GET /api/applications/:id/details
func (h *Handlers) GetApplicationWithDetails(c *fiber.Ctx) error {
	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Get application with details
	appWithDetails, err := h.service.GetApplicationWithDetails(c.Context(), applicationID)
	if err != nil {
		return err
	}

	return c.JSON(appWithDetails)
}

// ListApplications retrieves all applications with pagination
// GET /api/applications
func (h *Handlers) ListApplications(c *fiber.Ctx) error {
	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// List applications
	applications, err := h.service.ListApplications(c.Context(), pagination)
	if err != nil {
		return err
	}

	return c.JSON(applications)
}

// ListApplicationsByJob retrieves applications for a specific job
// GET /api/applications/by-job/:jobId
func (h *Handlers) ListApplicationsByJob(c *fiber.Ctx) error {
	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("jobId"))
	if jobID == "" {
		return application.ErrInvalidRequest().WithDetail("job_id", "missing or empty")
	}

	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// List applications by job
	applications, err := h.service.ListApplicationsByJob(c.Context(), jobID, pagination)
	if err != nil {
		return err
	}

	return c.JSON(applications)
}

// ListApplicationsByJobWithDetails retrieves applications with details for a specific job
// GET /api/applications/by-job/:jobId/details
func (h *Handlers) ListApplicationsByJobWithDetails(c *fiber.Ctx) error {
	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("jobId"))
	if jobID == "" {
		return application.ErrInvalidRequest().WithDetail("job_id", "missing or empty")
	}

	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// List applications with details by job
	applications, err := h.service.ListApplicationsByJobWithDetails(c.Context(), jobID, pagination)
	if err != nil {
		return err
	}

	return c.JSON(applications)
}

// ListApplicationsByCandidate retrieves applications for a specific candidate
// GET /api/applications/by-candidate/:candidateId
func (h *Handlers) ListApplicationsByCandidate(c *fiber.Ctx) error {
	// Parse candidate ID from URL
	candidateID := kernel.CandidateID(c.Params("candidateId"))
	if candidateID == "" {
		return application.ErrInvalidRequest().WithDetail("candidate_id", "missing or empty")
	}

	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// List applications by candidate
	applications, err := h.service.ListApplicationsByCandidate(c.Context(), candidateID, pagination)
	if err != nil {
		return err
	}

	return c.JSON(applications)
}

// ListApplicationsByCandidateWithDetails retrieves applications with details for a specific candidate
// GET /api/applications/by-candidate/:candidateId/details
func (h *Handlers) ListApplicationsByCandidateWithDetails(c *fiber.Ctx) error {
	// Parse candidate ID from URL
	candidateID := kernel.CandidateID(c.Params("candidateId"))
	if candidateID == "" {
		return application.ErrInvalidRequest().WithDetail("candidate_id", "missing or empty")
	}

	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// List applications with details by candidate
	applications, err := h.service.ListApplicationsByCandidateWithDetails(c.Context(), candidateID, pagination)
	if err != nil {
		return err
	}

	return c.JSON(applications)
}

// UpdateApplication updates an existing application
// PUT /api/applications/:id
func (h *Handlers) UpdateApplication(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Parse request body
	var req application.UpdateApplicationRequest
	if err := c.BodyParser(&req); err != nil {
		return application.ErrInvalidRequest().WithDetail("parse_error", err.Error())
	}

	// Update application
	updatedApplication, err := h.service.UpdateApplication(
		c.Context(),
		applicationID,
		req,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(updatedApplication)
}

// UploadResume uploads a resume file for an application with AI processing
// POST /api/applications/:id/resume
// UPDATED: Now uses AI to parse resume and generate embeddings
func (h *Handlers) UploadResume(c *fiber.Ctx) error {
	// Get auth context
	_, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return application.ErrInvalidRequest().WithDetail("file_error", err.Error())
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	validTypes := map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/jpg":       true,
		"image/png":       true,
		"image/webp":      true,
	}
	if !validTypes[contentType] {
		return application.ErrInvalidFileType().
			WithDetail("content_type", contentType).
			WithDetail("allowed_types", "pdf, jpg, png, webp")
	}

	// Open file
	fileContent, err := file.Open()
	if err != nil {
		return application.ErrInvalidRequest().WithDetail("file_open_error", err.Error())
	}
	defer fileContent.Close()

	// Read file data
	fileData, err := io.ReadAll(fileContent)
	if err != nil {
		return application.ErrInvalidRequest().WithDetail("file_read_error", err.Error())
	}

	// UPDATED: Use new AI-powered resume processing
	err = h.service.ProcessAndUploadResume(
		c.Context(),
		applicationID,
		fileData,
		file.Filename,
		contentType,
	)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":        "Resume uploaded and processed successfully",
		"application_id": applicationID,
	})
}

// DownloadResume downloads a resume file for an application
// GET /api/applications/:id/resume
func (h *Handlers) DownloadResume(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Download resume
	stream, filename, err := h.service.DownloadResume(
		c.Context(),
		applicationID,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}
	defer stream.Close()

	// Set headers for file download
	c.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Set("Content-Type", "application/octet-stream")

	// Stream file to response
	return c.SendStream(stream)
}

// AssignReviewer assigns a reviewer to an application
// POST /api/applications/:id/assign-reviewer
func (h *Handlers) AssignReviewer(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Parse request body
	var req application.AssignReviewerRequest
	if err := c.BodyParser(&req); err != nil {
		return application.ErrInvalidRequest().WithDetail("parse_error", err.Error())
	}

	// Assign reviewer
	if err := h.service.AssignReviewer(
		c.Context(),
		applicationID,
		req.ReviewerID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Reviewer assigned successfully",
	})
}

// GetApplicationsByReviewer retrieves applications assigned to a reviewer
// GET /api/applications/by-reviewer/:reviewerId
func (h *Handlers) GetApplicationsByReviewer(c *fiber.Ctx) error {
	// Parse reviewer ID from URL
	reviewerID := kernel.UserID(c.Params("reviewerId"))
	if reviewerID == "" {
		return application.ErrInvalidRequest().WithDetail("reviewer_id", "missing or empty")
	}

	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// Get applications by reviewer
	applications, err := h.service.GetApplicationsByReviewer(c.Context(), reviewerID, pagination)
	if err != nil {
		return err
	}

	return c.JSON(applications)
}

// UpdateApplicationStatus updates the status of an application
// PATCH /api/applications/:id/status
func (h *Handlers) UpdateApplicationStatus(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Parse request body
	var req application.UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return application.ErrInvalidRequest().WithDetail("parse_error", err.Error())
	}

	// Update status
	if err := h.service.UpdateApplicationStatus(
		c.Context(),
		applicationID,
		req.Status,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Application status updated successfully",
	})
}

// WithdrawApplication withdraws an application
// POST /api/applications/:id/withdraw
func (h *Handlers) WithdrawApplication(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Withdraw (update status to withdrawn)
	if err := h.service.UpdateApplicationStatus(
		c.Context(),
		applicationID,
		application.ApplicationStatusWithdrawn,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Application withdrawn successfully",
	})
}

// ApproveApplication approves an application
// POST /api/applications/:id/approve
func (h *Handlers) ApproveApplication(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Approve
	if err := h.service.UpdateApplicationStatus(
		c.Context(),
		applicationID,
		application.ApplicationStatusApproved,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Application approved successfully",
	})
}

// RejectApplication rejects an application
// POST /api/applications/:id/reject
func (h *Handlers) RejectApplication(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Reject
	if err := h.service.UpdateApplicationStatus(
		c.Context(),
		applicationID,
		application.ApplicationStatusRejected,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Application rejected successfully",
	})
}

// ArchiveApplication archives an application
// POST /api/applications/:id/archive
func (h *Handlers) ArchiveApplication(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Archive application
	if err := h.service.ArchiveApplication(
		c.Context(),
		applicationID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Application archived successfully",
	})
}

// UnarchiveApplication unarchives an application
// POST /api/applications/:id/unarchive
func (h *Handlers) UnarchiveApplication(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Unarchive application
	if err := h.service.UnarchiveApplication(
		c.Context(),
		applicationID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Application unarchived successfully",
	})
}

// DeleteApplication deletes an application
// DELETE /api/applications/:id
func (h *Handlers) DeleteApplication(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Delete application
	if err := h.service.DeleteApplication(
		c.Context(),
		applicationID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetApplicationStats retrieves statistics for an application
// GET /api/applications/:id/stats
func (h *Handlers) GetApplicationStats(c *fiber.Ctx) error {
	// Parse application ID from URL
	applicationID := kernel.ApplicationID(c.Params("id"))
	if applicationID == "" {
		return application.ErrApplicationNotFound().WithDetail("id", "missing or empty")
	}

	// Get application stats
	stats, err := h.service.GetApplicationStats(c.Context(), applicationID)
	if err != nil {
		return err
	}

	return c.JSON(stats)
}

// BulkArchiveApplications archives multiple applications
// POST /api/applications/bulk/archive
func (h *Handlers) BulkArchiveApplications(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse request body
	var req application.BulkArchiveApplicationsRequest
	if err := c.BodyParser(&req); err != nil {
		return application.ErrInvalidRequest().WithDetail("parse_error", err.Error())
	}

	// Bulk archive applications
	result, err := h.service.BulkArchiveApplications(
		c.Context(),
		req.ApplicationIDs,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

// BulkUpdateStatus updates status for multiple applications
// POST /api/applications/bulk/update-status
func (h *Handlers) BulkUpdateStatus(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return application.ErrInsufficientPermissions()
	}

	// Parse request body
	var req application.BulkUpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return application.ErrInvalidRequest().WithDetail("parse_error", err.Error())
	}

	// Bulk update status
	result, err := h.service.BulkUpdateStatus(
		c.Context(),
		req.ApplicationIDs,
		req.Status,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

// ============================================================================
// Helper Functions
// ============================================================================

// parsePaginationOptions extracts pagination options from query parameters
func parsePaginationOptions(c *fiber.Ctx) kernel.PaginationOptions {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 20)

	// Ensure valid values
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return kernel.PaginationOptions{
		Page:     page,
		PageSize: pageSize,
	}
}

// RegisterRoutes registers all application routes
func RegisterRoutes(app *fiber.App, handlers *Handlers, authMiddleware *auth.UnifiedAuthMiddleware) {
	api := app.Group("/api/applications")

	// Public/read routes (require authentication + read scope)
	api.Get("/",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsRead),
		handlers.ListApplications,
	)

	api.Get("/:id",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsRead),
		handlers.GetApplicationByID,
	)

	api.Get("/:id/details",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsRead),
		handlers.GetApplicationWithDetails,
	)

	api.Get("/by-job/:jobId",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsRead),
		handlers.ListApplicationsByJob,
	)

	api.Get("/by-job/:jobId/details",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsRead),
		handlers.ListApplicationsByJobWithDetails,
	)

	api.Get("/by-candidate/:candidateId",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsRead),
		handlers.ListApplicationsByCandidate,
	)

	api.Get("/by-candidate/:candidateId/details",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsRead),
		handlers.ListApplicationsByCandidateWithDetails,
	)

	api.Get("/by-reviewer/:reviewerId",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsRead),
		handlers.GetApplicationsByReviewer,
	)

	api.Get("/:id/stats",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsRead),
		handlers.GetApplicationStats,
	)

	// Write routes (require write scope)
	api.Post("/",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsWrite),
		handlers.CreateApplication,
	)

	api.Put("/:id",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsWrite),
		handlers.UpdateApplication,
	)

	api.Post("/:id/archive",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsWrite),
		handlers.ArchiveApplication,
	)

	api.Post("/:id/unarchive",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsWrite),
		handlers.UnarchiveApplication,
	)

	// Resume operations - UPDATED: Now uses AI processing
	api.Post("/:id/resume",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsWrite),
		handlers.UploadResume,
	)

	api.Get("/:id/resume",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsRead),
		handlers.DownloadResume,
	)

	// Reviewer assignment (require assign scope)
	api.Post("/:id/assign-reviewer",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsAssign),
		handlers.AssignReviewer,
	)

	// Status updates (require appropriate scopes)
	api.Patch("/:id/status",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsWrite),
		handlers.UpdateApplicationStatus,
	)

	api.Post("/:id/withdraw",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsWrite),
		handlers.WithdrawApplication,
	)

	// Approval/Rejection (require approve scope)
	api.Post("/:id/approve",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsApprove),
		handlers.ApproveApplication,
	)

	api.Post("/:id/reject",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsApprove),
		handlers.RejectApplication,
	)

	// Delete routes (require delete scope)
	api.Delete("/:id",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsDelete),
		handlers.DeleteApplication,
	)

	// Bulk operations
	api.Post("/bulk/archive",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsWrite),
		handlers.BulkArchiveApplications,
	)

	api.Post("/bulk/update-status",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeApplicationsWrite),
		handlers.BulkUpdateStatus,
	)
}
