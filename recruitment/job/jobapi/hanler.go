package jobapi

import (
	"github.com/Abraxas-365/relay/pkg/iam/auth"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/job"
	"github.com/Abraxas-365/relay/recruitment/job/jobsrv"
	"github.com/gofiber/fiber/v2"
)

// Handlers provides HTTP handlers for job operations
type Handlers struct {
	service *jobsrv.JobService
}

// NewHandlers creates a new job handlers instance
func NewHandlers(service *jobsrv.JobService) *Handlers {
	return &Handlers{
		service: service,
	}
}

// CreateJob creates a new job posting
// POST /api/jobs
func (h *Handlers) CreateJob(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return job.ErrInsufficientPermissions()
	}

	// Parse request body
	var req job.CreateJobRequest
	if err := c.BodyParser(&req); err != nil {
		return job.ErrInsufficientPermissions().WithDetail("parse_error", err.Error())
	}

	// Set the poster to the authenticated user
	req.PostedBy = *authContext.UserID

	// Create job
	newJob, err := h.service.CreateJob(
		c.Context(),
		req,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(newJob)
}

// GetJobByID retrieves a job by ID
// GET /api/jobs/:id
func (h *Handlers) GetJobByID(c *fiber.Ctx) error {
	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("id"))
	if jobID == "" {
		return job.ErrJobNotFound().WithDetail("id", "missing or empty")
	}

	// Get job
	jobResp, err := h.service.GetJobByID(c.Context(), jobID)
	if err != nil {
		return err
	}

	return c.JSON(jobResp)
}

// ListJobs retrieves all jobs with pagination
// GET /api/jobs
func (h *Handlers) ListJobs(c *fiber.Ctx) error {
	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// List jobs
	jobs, err := h.service.ListJobs(c.Context(), pagination)
	if err != nil {
		return err
	}

	return c.JSON(jobs)
}

// ListPublishedJobs retrieves only published/active jobs
// GET /api/jobs/published
func (h *Handlers) ListPublishedJobs(c *fiber.Ctx) error {
	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// List published jobs
	jobs, err := h.service.ListPublishedJobs(c.Context(), pagination)
	if err != nil {
		return err
	}

	return c.JSON(jobs)
}

// ListArchivedJobs retrieves archived jobs
// GET /api/jobs/archived
func (h *Handlers) ListArchivedJobs(c *fiber.Ctx) error {
	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// List archived jobs
	jobs, err := h.service.ListArchivedJobs(c.Context(), pagination)
	if err != nil {
		return err
	}

	return c.JSON(jobs)
}

// ListJobsByUser retrieves jobs posted by a specific user
// GET /api/jobs/by-user/:userId
func (h *Handlers) ListJobsByUser(c *fiber.Ctx) error {
	// Parse user ID from URL
	userID := kernel.UserID(c.Params("userId"))
	if userID == "" {
		return job.ErrInsufficientPermissions().WithDetail("user_id", "missing or empty")
	}

	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// Get jobs by user
	jobs, err := h.service.GetJobsByUser(c.Context(), userID, pagination)
	if err != nil {
		return err
	}

	return c.JSON(jobs)
}

// SearchJobs searches jobs by various criteria
// POST /api/jobs/search
func (h *Handlers) SearchJobs(c *fiber.Ctx) error {
	// Parse request body
	var req job.SearchJobsRequest
	if err := c.BodyParser(&req); err != nil {
		return job.ErrInsufficientPermissions().WithDetail("parse_error", err.Error())
	}

	// Search jobs
	jobs, err := h.service.SearchJobs(c.Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(jobs)
}

// UpdateJob updates an existing job
// PUT /api/jobs/:id
func (h *Handlers) UpdateJob(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return job.ErrInsufficientPermissions()
	}

	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("id"))
	if jobID == "" {
		return job.ErrJobNotFound().WithDetail("id", "missing or empty")
	}

	// Parse request body
	var req job.UpdateJobRequest
	if err := c.BodyParser(&req); err != nil {
		return job.ErrInsufficientPermissions().WithDetail("parse_error", err.Error())
	}

	// Update job
	updatedJob, err := h.service.UpdateJob(
		c.Context(),
		jobID,
		req,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(updatedJob)
}

// DeleteJob deletes a job
// DELETE /api/jobs/:id
func (h *Handlers) DeleteJob(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return job.ErrInsufficientPermissions()
	}

	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("id"))
	if jobID == "" {
		return job.ErrJobNotFound().WithDetail("id", "missing or empty")
	}

	// Delete job
	if err := h.service.DeleteJob(
		c.Context(),
		jobID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// PublishJob marks a job as published/active
// POST /api/jobs/:id/publish
func (h *Handlers) PublishJob(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return job.ErrInsufficientPermissions()
	}

	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("id"))
	if jobID == "" {
		return job.ErrJobNotFound().WithDetail("id", "missing or empty")
	}

	// Publish job
	if err := h.service.PublishJob(
		c.Context(),
		jobID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Job published successfully",
	})
}

// UnpublishJob marks a job as unpublished/draft
// POST /api/jobs/:id/unpublish
func (h *Handlers) UnpublishJob(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return job.ErrInsufficientPermissions()
	}

	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("id"))
	if jobID == "" {
		return job.ErrJobNotFound().WithDetail("id", "missing or empty")
	}

	// Unpublish job
	if err := h.service.UnpublishJob(
		c.Context(),
		jobID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Job unpublished successfully",
	})
}

// ArchiveJob archives a job
// POST /api/jobs/:id/archive
func (h *Handlers) ArchiveJob(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return job.ErrInsufficientPermissions()
	}

	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("id"))
	if jobID == "" {
		return job.ErrJobNotFound().WithDetail("id", "missing or empty")
	}

	// Archive job
	if err := h.service.ArchiveJob(
		c.Context(),
		jobID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Job archived successfully",
	})
}

// UnarchiveJob unarchives a job
// POST /api/jobs/:id/unarchive
func (h *Handlers) UnarchiveJob(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return job.ErrInsufficientPermissions()
	}

	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("id"))
	if jobID == "" {
		return job.ErrJobNotFound().WithDetail("id", "missing or empty")
	}

	// Unarchive job
	if err := h.service.UnarchiveJob(
		c.Context(),
		jobID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Job unarchived successfully",
	})
}

// GetJobStats retrieves statistics for a job
// GET /api/jobs/:id/stats
func (h *Handlers) GetJobStats(c *fiber.Ctx) error {
	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("id"))
	if jobID == "" {
		return job.ErrJobNotFound().WithDetail("id", "missing or empty")
	}

	// Get job stats
	stats, err := h.service.GetJobStats(c.Context(), jobID)
	if err != nil {
		return err
	}

	return c.JSON(stats)
}

// BulkPublishJobs publishes multiple jobs
// POST /api/jobs/bulk/publish
func (h *Handlers) BulkPublishJobs(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return job.ErrInsufficientPermissions()
	}

	// Parse request body
	var req struct {
		JobIDs []kernel.JobID `json:"job_ids" validate:"required,min=1"`
	}
	if err := c.BodyParser(&req); err != nil {
		return job.ErrInsufficientPermissions().WithDetail("parse_error", err.Error())
	}

	// Bulk publish jobs
	result, err := h.service.BulkPublishJobs(
		c.Context(),
		req.JobIDs,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

// BulkArchiveJobs archives multiple jobs
// POST /api/jobs/bulk/archive
func (h *Handlers) BulkArchiveJobs(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return job.ErrInsufficientPermissions()
	}

	// Parse request body
	var req struct {
		JobIDs []kernel.JobID `json:"job_ids" validate:"required,min=1"`
	}
	if err := c.BodyParser(&req); err != nil {
		return job.ErrInsufficientPermissions().WithDetail("parse_error", err.Error())
	}

	// Bulk archive jobs
	result, err := h.service.BulkArchiveJobs(
		c.Context(),
		req.JobIDs,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

// GetJobsByTitle retrieves jobs by title
// GET /api/jobs/by-title/:title
func (h *Handlers) GetJobsByTitle(c *fiber.Ctx) error {
	// Parse title from URL
	title := kernel.JobTitle(c.Params("title"))
	if title == "" {
		return job.ErrJobNotFound().WithDetail("title", "missing or empty")
	}

	// Get jobs by title
	jobs, err := h.service.GetJobsByTitle(c.Context(), title)
	if err != nil {
		return err
	}

	return c.JSON(jobs)
}

// CountUserJobs counts the number of jobs posted by a user
// GET /api/jobs/count/by-user/:userId
func (h *Handlers) CountUserJobs(c *fiber.Ctx) error {
	// Parse user ID from URL
	userID := kernel.UserID(c.Params("userId"))
	if userID == "" {
		return job.ErrInsufficientPermissions().WithDetail("user_id", "missing or empty")
	}

	// Count jobs
	count, err := h.service.CountUserJobs(c.Context(), userID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"user_id": userID,
		"count":   count,
	})
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

// RegisterRoutes registers all job routes
func RegisterRoutes(app *fiber.App, handlers *Handlers, authMiddleware *auth.UnifiedAuthMiddleware) {
	api := app.Group("/api/jobs")

	// Public/read routes (require authentication + read scope)
	api.Get("/",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsRead),
		handlers.ListJobs,
	)

	api.Get("/published",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsRead),
		handlers.ListPublishedJobs,
	)

	api.Get("/archived",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsRead),
		handlers.ListArchivedJobs,
	)

	api.Get("/:id",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsRead),
		handlers.GetJobByID,
	)

	api.Get("/by-user/:userId",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsRead),
		handlers.ListJobsByUser,
	)

	api.Get("/by-title/:title",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsRead),
		handlers.GetJobsByTitle,
	)

	api.Get("/:id/stats",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsRead),
		handlers.GetJobStats,
	)

	api.Get("/count/by-user/:userId",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsRead),
		handlers.CountUserJobs,
	)

	// Search route
	api.Post("/search",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsRead),
		handlers.SearchJobs,
	)

	// Write routes (require write scope)
	api.Post("/",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsWrite),
		handlers.CreateJob,
	)

	api.Put("/:id",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsWrite),
		handlers.UpdateJob,
	)

	// Publish/Unpublish routes (require publish scope)
	api.Post("/:id/publish",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsPublish),
		handlers.PublishJob,
	)

	api.Post("/:id/unpublish",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsPublish),
		handlers.UnpublishJob,
	)

	// Archive routes (require archive scope)
	api.Post("/:id/archive",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsArchive),
		handlers.ArchiveJob,
	)

	api.Post("/:id/unarchive",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsArchive),
		handlers.UnarchiveJob,
	)

	// Delete routes (require delete scope)
	api.Delete("/:id",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsDelete),
		handlers.DeleteJob,
	)

	// Bulk operations
	api.Post("/bulk/publish",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsPublish),
		handlers.BulkPublishJobs,
	)

	api.Post("/bulk/archive",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeJobsArchive),
		handlers.BulkArchiveJobs,
	)
}
