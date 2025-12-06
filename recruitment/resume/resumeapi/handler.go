package resumeapi

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/Abraxas-365/relay/pkg/fsx"
	"github.com/Abraxas-365/relay/pkg/iam/auth"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/resume"
	"github.com/Abraxas-365/relay/recruitment/resume/resumesrv"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ResumeHandlers struct {
	service    *resumesrv.Service
	fileSystem fsx.FileSystem // Add file system for uploads
}

func NewResumeHandlers(service *resumesrv.Service, fileSystem fsx.FileSystem) *ResumeHandlers {
	return &ResumeHandlers{
		service:    service,
		fileSystem: fileSystem,
	}
}

func (h *ResumeHandlers) RegisterRoutes(app *fiber.App, authMiddleware *auth.UnifiedAuthMiddleware) {
	resumes := app.Group("/api/v1/resumes", authMiddleware.Authenticate())

	// Resume CRUD
	resumes.Post("/parse", h.ParseResume)  // Parse and create from file (ASYNC)
	resumes.Post("/", h.CreateResume)      // Create manually
	resumes.Get("/:id", h.GetResume)       // Get by ID
	resumes.Put("/:id", h.UpdateResume)    // Update
	resumes.Delete("/:id", h.DeleteResume) // Delete
	resumes.Get("/", h.ListResumes)        // List all for tenant

	// Job Management (NEW)
	resumes.Get("/jobs/stats", h.GetJobStats)         // Get job statistics
	resumes.Get("/jobs/:job_id", h.GetJobStatus)      // Get job status
	resumes.Get("/jobs", h.ListJobs)                  // List all jobs
	resumes.Post("/jobs/:job_id/cancel", h.CancelJob) // Cancel job
	resumes.Post("/jobs/:job_id/retry", h.RetryJob)   // Retry failed job

	// Search & Stats
	resumes.Post("/search", h.SearchResumes) // Semantic search
	resumes.Get("/stats", h.GetStats)        // Get statistics

	// Resume Management
	resumes.Put("/:id/default", h.SetDefaultResume)       // Set as default
	resumes.Put("/:id/activate", h.ToggleActive)          // Toggle active status
	resumes.Put("/:id/statement", h.AddPersonalStatement) // Add personal statement

	// Embeddings Management
	resumes.Put("/:id/embeddings", h.UpdateEmbeddings)       // Update embeddings for one
	resumes.Post("/embeddings/bulk", h.BulkUpdateEmbeddings) // Bulk update embeddings
}

// ============================================================================
// Resume CRUD Handlers
// ============================================================================

// ParseResume parses a resume from an uploaded file (async processing)
// POST /api/v1/resumes/parse
func (h *ResumeHandlers) ParseResume(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "file is required",
		})
	}

	// Validate file size (e.g., max 10MB)
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if file.Size > maxSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":    "file too large",
			"max_size": "10MB",
			"size":     file.Size,
		})
	}

	// Get form fields
	title := c.FormValue("title")
	if title == "" {
		title = file.Filename
	}

	isActive := c.FormValue("is_active", "true") == "true"
	isDefault := c.FormValue("is_default", "false") == "true"

	// Determine file type
	fileType := determineFileType(file.Filename, file.Header.Get("Content-Type"))
	if fileType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":           "unsupported file type",
			"supported_types": []string{"pdf", "jpg", "jpeg", "png"},
			"detected_type":   file.Header.Get("Content-Type"),
			"file_extension":  filepath.Ext(file.Filename),
		})
	}

	// Open uploaded file
	uploadedFile, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to open uploaded file",
		})
	}
	defer uploadedFile.Close()

	// Generate unique file path
	// Format: resumes/{tenant_id}/{year}/{month}/{uuid}.{ext}
	now := time.Now()
	uniqueID := uuid.New().String()
	extension := filepath.Ext(file.Filename)
	if extension == "" {
		extension = "." + fileType
	}

	filePath := h.fileSystem.Join(
		"resumes",
		authCtx.TenantID.String(),
		fmt.Sprintf("%d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		uniqueID+extension,
	)

	// Upload file to storage (S3, GCS, etc.)
	if err := h.fileSystem.WriteFileStream(c.Context(), filePath, uploadedFile); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to upload file to storage",
			"details": err.Error(),
		})
	}

	// Create parse request
	req := resume.ParseResumeRequest{
		TenantID:  authCtx.TenantID,
		FilePath:  filePath,
		FileName:  file.Filename,
		FileType:  fileType,
		Title:     title,
		IsActive:  isActive,
		IsDefault: isDefault,
	}

	// Queue for async processing
	jobResponse, err := h.service.ParseResumeAsync(c.Context(), req)
	if err != nil {
		// If queueing fails, clean up the uploaded file
		_ = h.fileSystem.DeleteFile(c.Context(), filePath)
		return err
	}

	// Return 202 Accepted with job tracking information
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":    "Resume upload successful, processing started",
		"job":        jobResponse,
		"status_url": fmt.Sprintf("/api/v1/resumes/jobs/%s", jobResponse.JobID),
	})
}

// CreateResume creates a resume manually
// POST /api/v1/resumes
func (h *ResumeHandlers) CreateResume(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	var req resume.CreateResumeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	req.TenantID = authCtx.TenantID

	response, err := h.service.CreateResume(c.Context(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// GetResume retrieves a resume by ID
// GET /api/v1/resumes/:id
func (h *ResumeHandlers) GetResume(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	resumeID := kernel.ResumeID(c.Params("id"))
	if resumeID.IsEmpty() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid resume ID",
		})
	}

	response, err := h.service.GetResume(c.Context(), resumeID)
	if err != nil {
		return err
	}

	// Verify tenant ownership
	if response.TenantID != authCtx.TenantID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied",
		})
	}

	return c.JSON(response)
}

// UpdateResume updates a resume
// PUT /api/v1/resumes/:id
func (h *ResumeHandlers) UpdateResume(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	resumeID := kernel.ResumeID(c.Params("id"))
	if resumeID.IsEmpty() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid resume ID",
		})
	}

	var req resume.UpdateResumeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Verify tenant ownership first
	existing, err := h.service.GetResume(c.Context(), resumeID)
	if err != nil {
		return err
	}

	if existing.TenantID != authCtx.TenantID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied",
		})
	}

	response, err := h.service.UpdateResume(c.Context(), resumeID, req)
	if err != nil {
		return err
	}

	return c.JSON(response)
}

// DeleteResume deletes a resume
// DELETE /api/v1/resumes/:id
func (h *ResumeHandlers) DeleteResume(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	resumeID := kernel.ResumeID(c.Params("id"))
	if resumeID.IsEmpty() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid resume ID",
		})
	}

	// Verify tenant ownership first
	existing, err := h.service.GetResume(c.Context(), resumeID)
	if err != nil {
		return err
	}

	if existing.TenantID != authCtx.TenantID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied",
		})
	}

	// Delete from service
	if err := h.service.DeleteResume(c.Context(), resumeID); err != nil {
		return err
	}

	// Delete file from storage if exists
	if existing.FileURL != "" {
		_ = h.fileSystem.DeleteFile(c.Context(), existing.FileURL)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ListResumes lists all resumes for the tenant
// GET /api/v1/resumes?page=1&page_size=20&only_active=false
func (h *ResumeHandlers) ListResumes(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	req := resume.ListResumesRequest{
		TenantID:   authCtx.TenantID,
		OnlyActive: c.QueryBool("only_active", false),
		Pagination: kernel.PaginationOptions{
			Page:     c.QueryInt("page", 1),
			PageSize: c.QueryInt("page_size", 20),
		},
	}

	response, err := h.service.ListResumes(c.Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(response)
}

// ============================================================================
// Job Management Handlers (NEW)
// ============================================================================

// GetJobStatus retrieves the status of a resume processing job
// GET /api/v1/resumes/jobs/:job_id
func (h *ResumeHandlers) GetJobStatus(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	jobID := kernel.JobID(c.Params("job_id"))
	if jobID.IsEmpty() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid job ID",
		})
	}

	jobStatus, err := h.service.GetJobStatus(c.Context(), jobID)
	if err != nil {
		return err
	}

	// Verify tenant ownership
	if jobStatus.TenantID != authCtx.TenantID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied",
		})
	}

	return c.JSON(jobStatus)
}

// ListJobs lists all processing jobs for the authenticated tenant
// GET /api/v1/resumes/jobs?page=1&page_size=20
func (h *ResumeHandlers) ListJobs(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	pagination := kernel.PaginationOptions{
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("page_size", 20),
	}

	jobs, err := h.service.ListJobsByTenant(c.Context(), authCtx.TenantID, pagination)
	if err != nil {
		return err
	}

	return c.JSON(jobs)
}

// GetJobStats retrieves job statistics for the tenant
// GET /api/v1/resumes/jobs/stats
func (h *ResumeHandlers) GetJobStats(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	stats, err := h.service.GetJobStats(c.Context(), authCtx.TenantID)
	if err != nil {
		return err
	}

	return c.JSON(stats)
}

// CancelJob cancels a pending or processing job
// POST /api/v1/resumes/jobs/:job_id/cancel
func (h *ResumeHandlers) CancelJob(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	jobID := kernel.JobID(c.Params("job_id"))
	if jobID.IsEmpty() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid job ID",
		})
	}

	if err := h.service.CancelJob(c.Context(), jobID, authCtx.TenantID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "job cancelled successfully",
		"job_id":  jobID,
	})
}

// RetryJob retries a failed job
// POST /api/v1/resumes/jobs/:job_id/retry
func (h *ResumeHandlers) RetryJob(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	jobID := kernel.JobID(c.Params("job_id"))
	if jobID.IsEmpty() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid job ID",
		})
	}

	jobStatus, err := h.service.RetryFailedJob(c.Context(), jobID, authCtx.TenantID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "job retried successfully",
		"job":     jobStatus,
	})
}

// ============================================================================
// Search & Stats Handlers
// ============================================================================

// SearchResumes performs semantic search on resumes
// POST /api/v1/resumes/search
func (h *ResumeHandlers) SearchResumes(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	var req resume.SearchResumesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Set tenant ID to search only within tenant's resumes
	req.TenantID = &authCtx.TenantID

	// Set defaults
	if req.TopK == 0 {
		req.TopK = 10
	}
	if req.Pagination.Page == 0 {
		req.Pagination.Page = 1
	}
	if req.Pagination.PageSize == 0 {
		req.Pagination.PageSize = 20
	}

	response, err := h.service.SearchResumes(c.Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(response)
}

// GetStats gets resume statistics for the tenant
// GET /api/v1/resumes/stats
func (h *ResumeHandlers) GetStats(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	response, err := h.service.GetResumeStats(c.Context(), authCtx.TenantID)
	if err != nil {
		return err
	}

	return c.JSON(response)
}

// ============================================================================
// Resume Management Handlers
// ============================================================================

// SetDefaultResume sets a resume as the default for the tenant
// PUT /api/v1/resumes/:id/default
func (h *ResumeHandlers) SetDefaultResume(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	resumeID := kernel.ResumeID(c.Params("id"))
	if resumeID.IsEmpty() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid resume ID",
		})
	}

	if err := h.service.SetDefaultResume(c.Context(), authCtx.TenantID, resumeID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message":   "resume set as default",
		"resume_id": resumeID,
	})
}

// ToggleActive activates or deactivates a resume
// PUT /api/v1/resumes/:id/activate
// Body: {"is_active": true}
func (h *ResumeHandlers) ToggleActive(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	resumeID := kernel.ResumeID(c.Params("id"))
	if resumeID.IsEmpty() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid resume ID",
		})
	}

	var req resume.ToggleActiveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Verify tenant ownership first
	existing, err := h.service.GetResume(c.Context(), resumeID)
	if err != nil {
		return err
	}

	if existing.TenantID != authCtx.TenantID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied",
		})
	}

	if err := h.service.ToggleActive(c.Context(), resumeID, req.IsActive); err != nil {
		return err
	}

	status := "deactivated"
	if req.IsActive {
		status = "activated"
	}

	return c.JSON(fiber.Map{
		"message":   "resume " + status,
		"resume_id": resumeID,
		"is_active": req.IsActive,
	})
}

// AddPersonalStatement adds or updates a personal statement
// PUT /api/v1/resumes/:id/statement
func (h *ResumeHandlers) AddPersonalStatement(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	resumeID := kernel.ResumeID(c.Params("id"))
	if resumeID.IsEmpty() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid resume ID",
		})
	}

	var req resume.AddPersonalStatementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Verify tenant ownership first
	existing, err := h.service.GetResume(c.Context(), resumeID)
	if err != nil {
		return err
	}

	if existing.TenantID != authCtx.TenantID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied",
		})
	}

	response, err := h.service.AddPersonalStatement(c.Context(), resumeID, req)
	if err != nil {
		return err
	}

	return c.JSON(response)
}

// ============================================================================
// Embeddings Management Handlers
// ============================================================================

// UpdateEmbeddings updates embeddings for a specific resume
// PUT /api/v1/resumes/:id/embeddings
func (h *ResumeHandlers) UpdateEmbeddings(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	resumeID := kernel.ResumeID(c.Params("id"))
	if resumeID.IsEmpty() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid resume ID",
		})
	}

	// Verify tenant ownership first
	existing, err := h.service.GetResume(c.Context(), resumeID)
	if err != nil {
		return err
	}

	if existing.TenantID != authCtx.TenantID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied",
		})
	}

	if err := h.service.UpdateEmbeddings(c.Context(), resumeID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message":   "embeddings updated successfully",
		"resume_id": resumeID,
	})
}

// BulkUpdateEmbeddings updates embeddings for all resumes of the tenant
// POST /api/v1/resumes/embeddings/bulk
// Body: {"force": false}
func (h *ResumeHandlers) BulkUpdateEmbeddings(c *fiber.Ctx) error {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	var req resume.BulkUpdateEmbeddingsRequest
	if err := c.BodyParser(&req); err != nil {
		// If body is empty, use defaults
		req = resume.BulkUpdateEmbeddingsRequest{
			TenantID: authCtx.TenantID,
			Force:    false,
		}
	} else {
		req.TenantID = authCtx.TenantID
	}

	response, err := h.service.BulkUpdateEmbeddings(c.Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(response)
}

// ============================================================================
// Helper Functions
// ============================================================================

// determineFileType determines the file type from filename and content type
func determineFileType(filename, contentType string) string {
	// First try content type
	switch contentType {
	case "application/pdf":
		return "pdf"
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/png":
		return "png"
	}

	// Fallback to file extension
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}

	ext = ext[1:] // Remove leading dot
	switch ext {
	case "pdf":
		return "pdf"
	case "jpg", "jpeg":
		return "jpg"
	case "png":
		return "png"
	default:
		return ""
	}
}
