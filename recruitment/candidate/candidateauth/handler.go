package candidateauth

import (
	"io"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/application"
	"github.com/Abraxas-365/relay/recruitment/application/applicationsrv"
	"github.com/gofiber/fiber/v2"
)

type Handlers struct {
	authService        *CandidateAuthService
	applicationService *applicationsrv.ApplicationService
}

func NewHandlers(
	authService *CandidateAuthService,
	applicationService *applicationsrv.ApplicationService,
) *Handlers {
	return &Handlers{
		authService:        authService,
		applicationService: applicationService,
	}
}

// Request/Response DTOs
type RequestCodeRequest struct {
	Email kernel.Email `json:"email" validate:"required,email"`
	Phone kernel.Phone `json:"phone" validate:"required"`
}

type VerifyCodeRequest struct {
	Email kernel.Email `json:"email" validate:"required,email"`
	Phone kernel.Phone `json:"phone" validate:"required"`
	Code  string       `json:"code" validate:"required,len=6"`
}

type UpdateProfileRequest struct {
	FirstName kernel.FirstName `json:"first_name" validate:"required"`
	LastName  kernel.LastName  `json:"last_name" validate:"required"`
	DNI       kernel.DNI       `json:"dni" validate:"required"`
}

// RequestCode sends OTP to candidate
// POST /api/candidates/auth/request-code
func (h *Handlers) RequestCode(c *fiber.Ctx) error {
	var req RequestCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request")
	}

	if err := h.authService.RequestVerificationCode(c.Context(), req.Email, req.Phone); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Verification code sent to your email",
		"email":   req.Email,
	})
}

// VerifyCode verifies OTP and returns session token
// POST /api/candidates/auth/verify-code
func (h *Handlers) VerifyCode(c *fiber.Ctx) error {
	var req VerifyCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request")
	}

	session, err := h.authService.VerifyAndCreateSession(
		c.Context(),
		req.Email,
		req.Phone,
		req.Code,
	)
	if err != nil {
		return err
	}

	return c.JSON(session)
}

// GetProfile gets the candidate's profile
// GET /api/candidates/auth/profile
func (h *Handlers) GetProfile(c *fiber.Ctx) error {
	candidateID, ok := GetCandidateID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid session")
	}

	profile, err := h.authService.GetCandidateProfile(c.Context(), candidateID)
	if err != nil {
		return err
	}

	return c.JSON(profile)
}

// UpdateProfile updates candidate profile
// PUT /api/candidates/auth/profile
func (h *Handlers) UpdateProfile(c *fiber.Ctx) error {
	candidateID, ok := GetCandidateID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid session")
	}

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request")
	}

	if err := h.authService.UpdateCandidateProfile(
		c.Context(),
		candidateID,
		req.FirstName,
		req.LastName,
		req.DNI,
	); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Profile updated successfully",
	})
}

// ApplyToJob allows authenticated candidate to apply to a job
// POST /api/candidates/auth/apply/:jobId
// UPDATED: Now uses AI to parse resume and generate embeddings
func (h *Handlers) ApplyToJob(c *fiber.Ctx) error {
	candidateID, ok := GetCandidateID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid session")
	}

	// Parse job ID from URL
	jobID := kernel.JobID(c.Params("jobId"))
	if jobID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Job ID is required")
	}

	// Get resume file
	file, err := c.FormFile("resume")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Resume file is required")
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
		return fiber.NewError(fiber.StatusBadRequest, "Invalid file type. Supported: PDF, JPG, PNG, WebP")
	}

	// Validate file size (10MB max)
	if file.Size > 10*1024*1024 {
		return fiber.NewError(fiber.StatusBadRequest, "File size exceeds 10MB limit")
	}

	// Open and read file
	fileContent, err := file.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Failed to read resume file")
	}
	defer fileContent.Close()

	fileData, err := io.ReadAll(fileContent)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Failed to read resume file")
	}

	// Create application without resume data first
	appReq := application.CreateApplicationRequest{
		JobID:       jobID,
		CandidateID: candidateID,
	}

	app, err := h.applicationService.CreateApplication(c.Context(), appReq)
	if err != nil {
		return err
	}

	// UPDATED: Process and upload resume with AI parsing
	err = h.applicationService.ProcessAndUploadResume(
		c.Context(),
		app.ID,
		fileData,
		file.Filename,
		contentType,
	)
	if err != nil {
		// Rollback: delete application if resume processing fails
		h.applicationService.DeleteApplication(
			c.Context(),
			app.ID,
			kernel.UserID(candidateID),
			kernel.TenantID("PUBLIC"),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to process resume: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":        "Application submitted and resume processed successfully",
		"application_id": app.ID,
		"candidate_id":   candidateID,
		"job_id":         jobID,
	})
}

// GetMyApplications gets all applications for the authenticated candidate
// GET /api/candidates/auth/applications
func (h *Handlers) GetMyApplications(c *fiber.Ctx) error {
	candidateID, ok := GetCandidateID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid session")
	}

	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 20)

	pagination := kernel.PaginationOptions{
		Page:     page,
		PageSize: pageSize,
	}

	applications, err := h.applicationService.ListApplicationsByCandidate(
		c.Context(),
		candidateID,
		pagination,
	)
	if err != nil {
		return err
	}

	return c.JSON(applications)
}

// RegisterRoutes registers candidate auth routes
func RegisterRoutes(
	app *fiber.App,
	handlers *Handlers,
	candidateAuthMiddleware fiber.Handler,
) {
	api := app.Group("/api/candidates/auth")

	// Public routes - no auth required
	api.Post("/request-code", handlers.RequestCode)
	api.Post("/verify-code", handlers.VerifyCode)

	// Protected routes - require candidate session token
	api.Get("/profile", candidateAuthMiddleware, handlers.GetProfile)
	api.Put("/profile", candidateAuthMiddleware, handlers.UpdateProfile)
	api.Post("/apply/:jobId", candidateAuthMiddleware, handlers.ApplyToJob)
	api.Get("/applications", candidateAuthMiddleware, handlers.GetMyApplications)
}

