package candidateapi

import (
	"github.com/Abraxas-365/relay/pkg/iam/auth"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/candidate"
	"github.com/Abraxas-365/relay/recruitment/candidate/candidatesrv"
	"github.com/gofiber/fiber/v2"
)

// Handlers provides HTTP handlers for candidate operations
type Handlers struct {
	service *candidatesrv.CandidateService
}

// NewHandlers creates a new candidate handlers instance
func NewHandlers(service *candidatesrv.CandidateService) *Handlers {
	return &Handlers{
		service: service,
	}
}

// CreateCandidate creates a new candidate
// POST /api/candidates
func (h *Handlers) CreateCandidate(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return candidate.ErrInsufficientPermissions()
	}

	// Parse request body
	var req candidate.CreateCandidateRequest
	if err := c.BodyParser(&req); err != nil {
		return candidate.ErrInvalidEmail().WithDetail("parse_error", err.Error())
	}

	// Create candidate
	newCandidate, err := h.service.CreateCandidate(
		c.Context(),
		req,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(newCandidate)
}

// GetCandidateByID retrieves a candidate by ID
// GET /api/candidates/:id
func (h *Handlers) GetCandidateByID(c *fiber.Ctx) error {
	// Parse candidate ID from URL
	candidateID := kernel.CandidateID(c.Params("id"))
	if candidateID == "" {
		return candidate.ErrCandidateNotFound().WithDetail("id", "missing or empty")
	}

	// Get candidate
	candidateResp, err := h.service.GetCandidateByID(c.Context(), candidateID)
	if err != nil {
		return err
	}

	return c.JSON(candidateResp)
}

// GetCandidateByEmail retrieves a candidate by email
// GET /api/candidates/by-email/:email
func (h *Handlers) GetCandidateByEmail(c *fiber.Ctx) error {
	// Parse email from URL
	email := kernel.Email(c.Params("email"))
	if email == "" {
		return candidate.ErrInvalidEmail().WithDetail("email", "missing or empty")
	}

	// Get candidate
	candidateResp, err := h.service.GetCandidateByEmail(c.Context(), email)
	if err != nil {
		return err
	}

	return c.JSON(candidateResp)
}

// GetCandidateByDNI retrieves a candidate by DNI
// GET /api/candidates/by-dni/:type/:number
func (h *Handlers) GetCandidateByDNI(c *fiber.Ctx) error {
	// Parse DNI from URL
	dniType := kernel.DNIType(c.Params("type"))
	dniNumber := c.Params("number")

	dni := kernel.DNI{
		Type:   dniType,
		Number: dniNumber,
	}

	// Get candidate
	candidateResp, err := h.service.GetCandidateByDNI(c.Context(), dni)
	if err != nil {
		return err
	}

	return c.JSON(candidateResp)
}

// ListCandidates retrieves all candidates with pagination
// GET /api/candidates
func (h *Handlers) ListCandidates(c *fiber.Ctx) error {
	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// List candidates
	candidates, err := h.service.ListCandidates(c.Context(), pagination)
	if err != nil {
		return err
	}

	return c.JSON(candidates)
}

// SearchCandidates searches candidates by various criteria
// POST /api/candidates/search
func (h *Handlers) SearchCandidates(c *fiber.Ctx) error {
	// Parse request body
	var req candidate.SearchCandidatesRequest
	if err := c.BodyParser(&req); err != nil {
		return candidate.ErrInvalidEmail().WithDetail("parse_error", err.Error())
	}

	// Search candidates
	candidates, err := h.service.SearchCandidates(c.Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(candidates)
}

// UpdateCandidate updates an existing candidate
// PUT /api/candidates/:id
func (h *Handlers) UpdateCandidate(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return candidate.ErrInsufficientPermissions()
	}

	// Parse candidate ID from URL
	candidateID := kernel.CandidateID(c.Params("id"))
	if candidateID == "" {
		return candidate.ErrCandidateNotFound().WithDetail("id", "missing or empty")
	}

	// Parse request body
	var req candidate.UpdateCandidateRequest
	if err := c.BodyParser(&req); err != nil {
		return candidate.ErrInvalidEmail().WithDetail("parse_error", err.Error())
	}

	// Update candidate
	updatedCandidate, err := h.service.UpdateCandidate(
		c.Context(),
		candidateID,
		req,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(updatedCandidate)
}

// DeleteCandidate deletes a candidate
// DELETE /api/candidates/:id
func (h *Handlers) DeleteCandidate(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return candidate.ErrInsufficientPermissions()
	}

	// Parse candidate ID from URL
	candidateID := kernel.CandidateID(c.Params("id"))
	if candidateID == "" {
		return candidate.ErrCandidateNotFound().WithDetail("id", "missing or empty")
	}

	// Delete candidate
	if err := h.service.DeleteCandidate(
		c.Context(),
		candidateID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ArchiveCandidate archives a candidate
// POST /api/candidates/:id/archive
func (h *Handlers) ArchiveCandidate(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return candidate.ErrInsufficientPermissions()
	}

	// Parse candidate ID from URL
	candidateID := kernel.CandidateID(c.Params("id"))
	if candidateID == "" {
		return candidate.ErrCandidateNotFound().WithDetail("id", "missing or empty")
	}

	// Archive candidate
	if err := h.service.ArchiveCandidate(
		c.Context(),
		candidateID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Candidate archived successfully",
	})
}

// UnarchiveCandidate unarchives a candidate
// POST /api/candidates/:id/unarchive
func (h *Handlers) UnarchiveCandidate(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return candidate.ErrInsufficientPermissions()
	}

	// Parse candidate ID from URL
	candidateID := kernel.CandidateID(c.Params("id"))
	if candidateID == "" {
		return candidate.ErrCandidateNotFound().WithDetail("id", "missing or empty")
	}

	// Unarchive candidate
	if err := h.service.UnarchiveCandidate(
		c.Context(),
		candidateID,
		*authContext.UserID,
		authContext.TenantID,
	); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Candidate unarchived successfully",
	})
}

// GetCandidateStats retrieves statistics for a candidate
// GET /api/candidates/:id/stats
func (h *Handlers) GetCandidateStats(c *fiber.Ctx) error {
	// Parse candidate ID from URL
	candidateID := kernel.CandidateID(c.Params("id"))
	if candidateID == "" {
		return candidate.ErrCandidateNotFound().WithDetail("id", "missing or empty")
	}

	// Get candidate stats
	stats, err := h.service.GetCandidateStats(c.Context(), candidateID)
	if err != nil {
		return err
	}

	return c.JSON(stats)
}

// BulkArchiveCandidates archives multiple candidates
// POST /api/candidates/bulk/archive
func (h *Handlers) BulkArchiveCandidates(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return candidate.ErrInsufficientPermissions()
	}

	// Parse request body
	var req candidate.BulkArchiveCandidatesRequest
	if err := c.BodyParser(&req); err != nil {
		return candidate.ErrInvalidEmail().WithDetail("parse_error", err.Error())
	}

	// Bulk archive candidates
	result, err := h.service.BulkArchiveCandidates(
		c.Context(),
		req.CandidateIDs,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

// BulkUnarchiveCandidates unarchives multiple candidates
// POST /api/candidates/bulk/unarchive
func (h *Handlers) BulkUnarchiveCandidates(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return candidate.ErrInsufficientPermissions()
	}

	// Parse request body
	var req candidate.BulkArchiveCandidatesRequest
	if err := c.BodyParser(&req); err != nil {
		return candidate.ErrInvalidEmail().WithDetail("parse_error", err.Error())
	}

	// Bulk unarchive candidates
	result, err := h.service.BulkUnarchiveCandidates(
		c.Context(),
		req.CandidateIDs,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

// BulkDeleteCandidates deletes multiple candidates
// POST /api/candidates/bulk/delete
func (h *Handlers) BulkDeleteCandidates(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return candidate.ErrInsufficientPermissions()
	}

	// Parse request body
	var req candidate.BulkDeleteCandidatesRequest
	if err := c.BodyParser(&req); err != nil {
		return candidate.ErrInvalidEmail().WithDetail("parse_error", err.Error())
	}

	// Bulk delete candidates
	result, err := h.service.BulkDeleteCandidates(
		c.Context(),
		req.CandidateIDs,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

// ExportCandidates exports candidate data
// POST /api/candidates/export
func (h *Handlers) ExportCandidates(c *fiber.Ctx) error {
	// Get auth context
	authContext, ok := auth.GetAuthContext(c)
	if !ok {
		return candidate.ErrInsufficientPermissions()
	}

	// Parse request body
	var req candidate.ExportCandidatesRequest
	if err := c.BodyParser(&req); err != nil {
		return candidate.ErrInvalidEmail().WithDetail("parse_error", err.Error())
	}

	// Export candidates
	exportData, err := h.service.ExportCandidates(
		c.Context(),
		req,
		*authContext.UserID,
		authContext.TenantID,
	)
	if err != nil {
		return err
	}

	return c.JSON(exportData)
}

// GetCandidatesByDNIType retrieves candidates by DNI type
// GET /api/candidates/by-dni-type/:type
func (h *Handlers) GetCandidatesByDNIType(c *fiber.Ctx) error {
	// Parse DNI type from URL
	dniType := kernel.DNIType(c.Params("type"))
	if dniType == "" {
		return candidate.ErrInvalidDNI().WithDetail("type", "missing or empty")
	}

	// Parse pagination options
	pagination := parsePaginationOptions(c)

	// Get candidates by DNI type
	candidates, err := h.service.GetCandidatesByDNIType(c.Context(), dniType, pagination)
	if err != nil {
		return err
	}

	return c.JSON(candidates)
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

// RegisterRoutes registers all candidate routes
func RegisterRoutes(app *fiber.App, handlers *Handlers, authMiddleware *auth.UnifiedAuthMiddleware) {
	api := app.Group("/api/candidates")

	// Public/read routes (require authentication + read scope)
	api.Get("/",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesRead),
		handlers.ListCandidates,
	)

	api.Get("/:id",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesRead),
		handlers.GetCandidateByID,
	)

	api.Get("/by-email/:email",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesRead),
		handlers.GetCandidateByEmail,
	)

	api.Get("/by-dni/:type/:number",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesRead),
		handlers.GetCandidateByDNI,
	)

	api.Get("/by-dni-type/:type",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesRead),
		handlers.GetCandidatesByDNIType,
	)

	api.Get("/:id/stats",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesRead),
		handlers.GetCandidateStats,
	)

	// Search route
	api.Post("/search",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesRead),
		handlers.SearchCandidates,
	)

	// Write routes (require write scope)
	api.Post("/",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesWrite),
		handlers.CreateCandidate,
	)

	api.Put("/:id",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesWrite),
		handlers.UpdateCandidate,
	)

	api.Post("/:id/archive",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesWrite),
		handlers.ArchiveCandidate,
	)

	api.Post("/:id/unarchive",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesWrite),
		handlers.UnarchiveCandidate,
	)

	// Delete routes (require delete scope)
	api.Delete("/:id",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesDelete),
		handlers.DeleteCandidate,
	)

	// Bulk operations
	api.Post("/bulk/archive",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesWrite),
		handlers.BulkArchiveCandidates,
	)

	api.Post("/bulk/unarchive",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesWrite),
		handlers.BulkUnarchiveCandidates,
	)

	api.Post("/bulk/delete",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesDelete),
		handlers.BulkDeleteCandidates,
	)

	// Export route (require export scope)
	api.Post("/export",
		authMiddleware.Authenticate(),
		authMiddleware.RequireScope(auth.ScopeCandidatesExport),
		handlers.ExportCandidates,
	)
}
