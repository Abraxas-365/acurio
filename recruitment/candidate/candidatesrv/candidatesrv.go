package candidatesrv

import (
	"context"
	"time"

	"github.com/Abraxas-365/relay/pkg/errx"
	"github.com/Abraxas-365/relay/pkg/iam/auth"
	"github.com/Abraxas-365/relay/pkg/iam/user"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/candidate"
	"github.com/google/uuid"
)

// CandidateService provides business operations for candidates
type CandidateService struct {
	candidateRepo candidate.Repository
	userRepo      user.UserRepository
}

// NewCandidateService creates a new instance of the candidate service
func NewCandidateService(
	candidateRepo candidate.Repository,
	userRepo user.UserRepository,
) *CandidateService {
	return &CandidateService{
		candidateRepo: candidateRepo,
		userRepo:      userRepo,
	}
}

// CreateCandidate creates a new candidate
func (s *CandidateService) CreateCandidate(ctx context.Context, req candidate.CreateCandidateRequest, creatorID kernel.UserID, tenantID kernel.TenantID) (*candidate.Candidate, error) {
	// Validate that the creator exists and is active
	creator, err := s.userRepo.FindByID(ctx, creatorID, tenantID)
	if err != nil {
		return nil, user.ErrUserNotFound().WithDetail("user_id", creatorID.String())
	}

	if !creator.IsActive() {
		return nil, user.ErrUserSuspended().WithDetail("user_id", creatorID.String())
	}

	// Verify user has permission to create candidates
	if !creator.HasAnyScope(auth.ScopeCandidatesWrite, auth.ScopeCandidatesAll, auth.ScopeAll) {
		return nil, candidate.ErrInsufficientPermissions().WithDetail("required_scope", "candidates:write")
	}

	// Validate DNI format
	if !req.DNI.IsValid() {
		return nil, candidate.ErrInvalidDNI().
			WithDetail("dni_type", req.DNI.Type).
			WithDetail("dni_number", req.DNI.Number)
	}

	// Check for existing candidate by email
	existingByEmail, err := s.candidateRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingByEmail != nil {
		return nil, candidate.ErrCandidateAlreadyExists().
			WithDetail("email", string(req.Email)).
			WithDetail("existing_id", existingByEmail.ID.String())
	}

	// Check for existing candidate by DNI
	existingByDNI, err := s.candidateRepo.GetByDNI(ctx, req.DNI)
	if err == nil && existingByDNI != nil {
		return nil, candidate.ErrCandidateAlreadyExists().
			WithDetail("dni_type", req.DNI.Type).
			WithDetail("dni_number", req.DNI.Number).
			WithDetail("existing_id", existingByDNI.ID.String())
	}

	// Create new candidate entity
	newCandidate := &candidate.Candidate{
		ID:        kernel.NewCandidateID(uuid.NewString()),
		Email:     req.Email,
		Phone:     req.Phone,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		DNI:       req.DNI,
		Status:    candidate.CandidateStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save candidate
	if err := s.candidateRepo.Create(ctx, newCandidate); err != nil {
		return nil, errx.Wrap(err, "failed to create candidate", errx.TypeInternal)
	}

	return newCandidate, nil
}

// GetCandidateByID retrieves a candidate by ID
func (s *CandidateService) GetCandidateByID(ctx context.Context, candidateID kernel.CandidateID) (*candidate.CandidateResponse, error) {
	candidateEntity, err := s.candidateRepo.GetByID(ctx, candidateID)
	if err != nil {
		return nil, candidate.ErrCandidateNotFound().WithDetail("candidate_id", candidateID.String())
	}

	return s.toCandidateResponse(candidateEntity), nil
}

// GetCandidateByEmail retrieves a candidate by email
func (s *CandidateService) GetCandidateByEmail(ctx context.Context, email kernel.Email) (*candidate.CandidateResponse, error) {
	candidateEntity, err := s.candidateRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, candidate.ErrCandidateNotFound().WithDetail("email", string(email))
	}

	return s.toCandidateResponse(candidateEntity), nil
}

// GetCandidateByDNI retrieves a candidate by DNI
func (s *CandidateService) GetCandidateByDNI(ctx context.Context, dni kernel.DNI) (*candidate.CandidateResponse, error) {
	if !dni.IsValid() {
		return nil, candidate.ErrInvalidDNI().
			WithDetail("dni_type", dni.Type).
			WithDetail("dni_number", dni.Number)
	}

	candidateEntity, err := s.candidateRepo.GetByDNI(ctx, dni)
	if err != nil {
		return nil, candidate.ErrCandidateNotFound().
			WithDetail("dni_type", dni.Type).
			WithDetail("dni_number", dni.Number)
	}

	return s.toCandidateResponse(candidateEntity), nil
}

// ListCandidates retrieves all candidates with pagination
func (s *CandidateService) ListCandidates(ctx context.Context, pagination kernel.PaginationOptions) (*candidate.PaginatedCandidatesResponse, error) {
	candidates, err := s.candidateRepo.List(ctx, pagination)
	if err != nil {
		return nil, errx.Wrap(err, "failed to list candidates", errx.TypeInternal)
	}

	// Convert to response DTOs
	responses := make([]candidate.CandidateResponse, 0, len(candidates.Items))
	for _, c := range candidates.Items {
		responses = append(responses, *s.toCandidateResponse(&c))
	}

	return &kernel.Paginated[candidate.CandidateResponse]{
		Items: responses,
		Page:  candidates.Page,
		Empty: candidates.Empty,
	}, nil
}

// SearchCandidates searches candidates by various criteria
func (s *CandidateService) SearchCandidates(ctx context.Context, req candidate.SearchCandidatesRequest) (*candidate.PaginatedCandidatesResponse, error) {
	candidates, err := s.candidateRepo.Search(ctx, req)
	if err != nil {
		return nil, errx.Wrap(err, "failed to search candidates", errx.TypeInternal)
	}

	// Convert to response DTOs
	responses := make([]candidate.CandidateResponse, 0, len(candidates.Items))
	for _, c := range candidates.Items {
		responses = append(responses, *s.toCandidateResponse(&c))
	}

	return &kernel.Paginated[candidate.CandidateResponse]{
		Items: responses,
		Page:  candidates.Page,
		Empty: candidates.Empty,
	}, nil
}

// UpdateCandidate updates an existing candidate
func (s *CandidateService) UpdateCandidate(ctx context.Context, candidateID kernel.CandidateID, req candidate.UpdateCandidateRequest, updaterID kernel.UserID, tenantID kernel.TenantID) (*candidate.Candidate, error) {
	// Get existing candidate
	candidateEntity, err := s.candidateRepo.GetByID(ctx, candidateID)
	if err != nil {
		return nil, candidate.ErrCandidateNotFound().WithDetail("candidate_id", candidateID.String())
	}

	// Verify updater exists and is active
	updater, err := s.userRepo.FindByID(ctx, updaterID, tenantID)
	if err != nil {
		return nil, user.ErrUserNotFound().WithDetail("user_id", updaterID.String())
	}

	if !updater.IsActive() {
		return nil, user.ErrUserSuspended()
	}

	// Verify user has permission to update candidates
	if !updater.HasAnyScope(auth.ScopeCandidatesWrite, auth.ScopeCandidatesAll, auth.ScopeAll) {
		return nil, candidate.ErrInsufficientPermissions().
			WithDetail("required_scope", "candidates:write").
			WithDetail("user_id", updaterID.String())
	}

	// Business rule: Can't update archived candidates
	if candidateEntity.IsArchived() {
		return nil, candidate.ErrCandidateArchived().WithDetail("candidate_id", candidateID.String())
	}

	// Track if any changes were made
	updated := false

	// Update fields if provided
	if req.Email != nil && *req.Email != candidateEntity.Email {
		// Check for duplicate email
		existingByEmail, err := s.candidateRepo.GetByEmail(ctx, *req.Email)
		if err == nil && existingByEmail != nil && existingByEmail.ID != candidateID {
			return nil, candidate.ErrEmailAlreadyExists().
				WithDetail("email", string(*req.Email)).
				WithDetail("existing_id", existingByEmail.ID.String())
		}
		candidateEntity.Email = *req.Email
		updated = true
	}

	if req.Phone != nil && *req.Phone != candidateEntity.Phone {
		candidateEntity.Phone = *req.Phone
		updated = true
	}

	if req.FirstName != nil && *req.FirstName != candidateEntity.FirstName {
		candidateEntity.FirstName = *req.FirstName
		updated = true
	}

	if req.LastName != nil && *req.LastName != candidateEntity.LastName {
		candidateEntity.LastName = *req.LastName
		updated = true
	}

	if req.DNI != nil && *req.DNI != candidateEntity.DNI {
		// Validate new DNI
		if !req.DNI.IsValid() {
			return nil, candidate.ErrInvalidDNI().
				WithDetail("dni_type", req.DNI.Type).
				WithDetail("dni_number", req.DNI.Number)
		}

		// Check for duplicate DNI
		existingByDNI, err := s.candidateRepo.GetByDNI(ctx, *req.DNI)
		if err == nil && existingByDNI != nil && existingByDNI.ID != candidateID {
			return nil, candidate.ErrDNIAlreadyExists().
				WithDetail("dni_type", req.DNI.Type).
				WithDetail("dni_number", req.DNI.Number).
				WithDetail("existing_id", existingByDNI.ID.String())
		}

		candidateEntity.DNI = *req.DNI
		updated = true
	}

	if updated {
		candidateEntity.UpdatedAt = time.Now()

		// Save changes
		if err := s.candidateRepo.Update(ctx, candidateID, candidateEntity); err != nil {
			return nil, errx.Wrap(err, "failed to update candidate", errx.TypeInternal)
		}
	}

	return candidateEntity, nil
}

// DeleteCandidate deletes a candidate
func (s *CandidateService) DeleteCandidate(ctx context.Context, candidateID kernel.CandidateID, deleterID kernel.UserID, tenantID kernel.TenantID) error {
	// Get candidate
	_, err := s.candidateRepo.GetByID(ctx, candidateID)
	if err != nil {
		return candidate.ErrCandidateNotFound().WithDetail("candidate_id", candidateID.String())
	}

	// Verify deleter has permission
	deleter, err := s.userRepo.FindByID(ctx, deleterID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	if !deleter.HasAnyScope(auth.ScopeCandidatesDelete, auth.ScopeCandidatesAll, auth.ScopeAll) {
		return candidate.ErrInsufficientPermissions().
			WithDetail("required_scope", "candidates:delete").
			WithDetail("user_id", deleterID.String())
	}

	// Business rule: Check for active applications
	applicationCount, err := s.candidateRepo.CountApplications(ctx, candidateID)
	if err != nil {
		// Log error but don't fail
		// logger.Warn("Failed to count applications for candidate", "candidate_id", candidateID, "error", err)
	}

	if applicationCount > 0 {
		return candidate.ErrCandidateHasApplications().
			WithDetail("candidate_id", candidateID.String()).
			WithDetail("application_count", applicationCount)
	}

	// Delete candidate
	if err := s.candidateRepo.Delete(ctx, candidateID); err != nil {
		return errx.Wrap(err, "failed to delete candidate", errx.TypeInternal)
	}

	return nil
}

// ArchiveCandidate archives a candidate (soft delete alternative)
func (s *CandidateService) ArchiveCandidate(ctx context.Context, candidateID kernel.CandidateID, archiverID kernel.UserID, tenantID kernel.TenantID) error {
	// Get candidate
	candidateEntity, err := s.candidateRepo.GetByID(ctx, candidateID)
	if err != nil {
		return candidate.ErrCandidateNotFound().WithDetail("candidate_id", candidateID.String())
	}

	// Verify archiver has permission
	archiver, err := s.userRepo.FindByID(ctx, archiverID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	if !archiver.HasAnyScope(auth.ScopeCandidatesWrite, auth.ScopeCandidatesAll, auth.ScopeAll) {
		return candidate.ErrInsufficientPermissions().
			WithDetail("required_scope", "candidates:write").
			WithDetail("user_id", archiverID.String())
	}

	// Business rule: Can't archive already archived candidates
	if candidateEntity.IsArchived() {
		return candidate.ErrCandidateAlreadyArchived().WithDetail("candidate_id", candidateID.String())
	}

	// Archive candidate
	if err := candidateEntity.Archive(); err != nil {
		return err
	}

	// Save changes
	if err := s.candidateRepo.Update(ctx, candidateID, candidateEntity); err != nil {
		return errx.Wrap(err, "failed to archive candidate", errx.TypeInternal)
	}

	return nil
}

// UnarchiveCandidate unarchives a candidate
func (s *CandidateService) UnarchiveCandidate(ctx context.Context, candidateID kernel.CandidateID, unarchiverID kernel.UserID, tenantID kernel.TenantID) error {
	// Get candidate
	candidateEntity, err := s.candidateRepo.GetByID(ctx, candidateID)
	if err != nil {
		return candidate.ErrCandidateNotFound().WithDetail("candidate_id", candidateID.String())
	}

	// Verify unarchiver has permission
	unarchiver, err := s.userRepo.FindByID(ctx, unarchiverID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	if !unarchiver.HasAnyScope(auth.ScopeCandidatesWrite, auth.ScopeCandidatesAll, auth.ScopeAll) {
		return candidate.ErrInsufficientPermissions().
			WithDetail("required_scope", "candidates:write").
			WithDetail("user_id", unarchiverID.String())
	}

	// Business rule: Can only unarchive archived candidates
	if !candidateEntity.IsArchived() {
		return candidate.ErrCandidateNotArchived().WithDetail("candidate_id", candidateID.String())
	}

	// Unarchive candidate
	if err := candidateEntity.Unarchive(); err != nil {
		return err
	}

	// Save changes
	if err := s.candidateRepo.Update(ctx, candidateID, candidateEntity); err != nil {
		return errx.Wrap(err, "failed to unarchive candidate", errx.TypeInternal)
	}

	return nil
}

// GetCandidateStats retrieves statistics for a candidate
func (s *CandidateService) GetCandidateStats(ctx context.Context, candidateID kernel.CandidateID) (*candidate.CandidateStatsResponse, error) {
	candidateEntity, err := s.candidateRepo.GetByID(ctx, candidateID)
	if err != nil {
		return nil, candidate.ErrCandidateNotFound().WithDetail("candidate_id", candidateID.String())
	}

	// Count applications
	applicationCount, err := s.candidateRepo.CountApplications(ctx, candidateID)
	if err != nil {
		applicationCount = 0 // Default to 0 on error
	}

	stats := &candidate.CandidateStatsResponse{
		CandidateID:       candidateID,
		FullName:          candidateEntity.GetFullName(),
		Email:             candidateEntity.Email,
		Status:            candidateEntity.Status,
		TotalApplications: applicationCount,
		IsArchived:        candidateEntity.IsArchived(),
		CreatedAt:         candidateEntity.CreatedAt,
		UpdatedAt:         candidateEntity.UpdatedAt,
	}

	// Calculate days since registration
	days := int(time.Since(candidateEntity.CreatedAt).Hours() / 24)
	stats.DaysSinceRegistration = days

	// Calculate days since last update
	daysSinceUpdate := int(time.Since(candidateEntity.UpdatedAt).Hours() / 24)
	stats.DaysSinceLastUpdate = daysSinceUpdate

	// Calculate days since archived
	if candidateEntity.ArchivedAt != nil {
		daysSinceArchived := int(time.Since(*candidateEntity.ArchivedAt).Hours() / 24)
		stats.DaysSinceArchived = &daysSinceArchived
	}

	return stats, nil
}

// BulkArchiveCandidates archives multiple candidates
func (s *CandidateService) BulkArchiveCandidates(ctx context.Context, candidateIDs []kernel.CandidateID, archiverID kernel.UserID, tenantID kernel.TenantID) (*candidate.BulkCandidateOperationResponse, error) {
	result := &candidate.BulkCandidateOperationResponse{
		Successful: []kernel.CandidateID{},
		Failed:     make(map[kernel.CandidateID]string),
		Total:      len(candidateIDs),
	}

	for _, candidateID := range candidateIDs {
		if err := s.ArchiveCandidate(ctx, candidateID, archiverID, tenantID); err != nil {
			result.Failed[candidateID] = err.Error()
		} else {
			result.Successful = append(result.Successful, candidateID)
		}
	}

	return result, nil
}

// BulkUnarchiveCandidates unarchives multiple candidates
func (s *CandidateService) BulkUnarchiveCandidates(ctx context.Context, candidateIDs []kernel.CandidateID, unarchiverID kernel.UserID, tenantID kernel.TenantID) (*candidate.BulkCandidateOperationResponse, error) {
	result := &candidate.BulkCandidateOperationResponse{
		Successful: []kernel.CandidateID{},
		Failed:     make(map[kernel.CandidateID]string),
		Total:      len(candidateIDs),
	}

	for _, candidateID := range candidateIDs {
		if err := s.UnarchiveCandidate(ctx, candidateID, unarchiverID, tenantID); err != nil {
			result.Failed[candidateID] = err.Error()
		} else {
			result.Successful = append(result.Successful, candidateID)
		}
	}

	return result, nil
}

// BulkDeleteCandidates deletes multiple candidates
func (s *CandidateService) BulkDeleteCandidates(ctx context.Context, candidateIDs []kernel.CandidateID, deleterID kernel.UserID, tenantID kernel.TenantID) (*candidate.BulkCandidateOperationResponse, error) {
	result := &candidate.BulkCandidateOperationResponse{
		Successful: []kernel.CandidateID{},
		Failed:     make(map[kernel.CandidateID]string),
		Total:      len(candidateIDs),
	}

	for _, candidateID := range candidateIDs {
		if err := s.DeleteCandidate(ctx, candidateID, deleterID, tenantID); err != nil {
			result.Failed[candidateID] = err.Error()
		} else {
			result.Successful = append(result.Successful, candidateID)
		}
	}

	return result, nil
}

// ValidateCandidateExists checks if a candidate exists
func (s *CandidateService) ValidateCandidateExists(ctx context.Context, candidateID kernel.CandidateID) error {
	exists, err := s.candidateRepo.Exists(ctx, candidateID)
	if err != nil {
		return errx.Wrap(err, "failed to check candidate existence", errx.TypeInternal)
	}

	if !exists {
		return candidate.ErrCandidateNotFound().WithDetail("candidate_id", candidateID.String())
	}

	return nil
}

// GetCandidatesByDNIType retrieves candidates by DNI type
func (s *CandidateService) GetCandidatesByDNIType(ctx context.Context, dniType kernel.DNIType, pagination kernel.PaginationOptions) (*candidate.PaginatedCandidatesResponse, error) {
	// Use search with DNI type filter
	searchReq := candidate.SearchCandidatesRequest{
		DNIType:    string(dniType),
		Pagination: pagination,
	}

	return s.SearchCandidates(ctx, searchReq)
}

// ExportCandidates exports candidate data (placeholder for future CSV/Excel export)
func (s *CandidateService) ExportCandidates(ctx context.Context, req candidate.ExportCandidatesRequest, exporterID kernel.UserID, tenantID kernel.TenantID) (*candidate.ExportCandidatesResponse, error) {
	// Verify exporter has permission
	exporter, err := s.userRepo.FindByID(ctx, exporterID, tenantID)
	if err != nil {
		return nil, user.ErrUserNotFound()
	}

	if !exporter.HasAnyScope(auth.ScopeCandidatesExport, auth.ScopeCandidatesAll, auth.ScopeAll) {
		return nil, candidate.ErrInsufficientPermissions().
			WithDetail("required_scope", "candidates:export").
			WithDetail("user_id", exporterID.String())
	}

	// Get candidates based on filters
	var candidates *kernel.Paginated[candidate.Candidate]

	if req.CandidateIDs != nil && len(req.CandidateIDs) > 0 {
		// Export specific candidates
		items := make([]candidate.Candidate, 0, len(req.CandidateIDs))
		for _, id := range req.CandidateIDs {
			c, err := s.candidateRepo.GetByID(ctx, id)
			if err == nil {
				items = append(items, *c)
			}
		}
		candidates = &kernel.Paginated[candidate.Candidate]{
			Items: items,
			Page: kernel.Page{
				Number: 1,
				Size:   len(items),
				Total:  len(items),
				Pages:  1,
			},
			Empty: len(items) == 0,
		}
	} else {
		// Export all (with reasonable limit)
		candidates, err = s.candidateRepo.List(ctx, kernel.PaginationOptions{
			Page:     1,
			PageSize: 1000, // Max 1000 candidates per export
		})
		if err != nil {
			return nil, errx.Wrap(err, "failed to fetch candidates for export", errx.TypeInternal)
		}
	}

	// Convert to export format
	exportData := make([]candidate.CandidateExportData, 0, len(candidates.Items))
	for _, c := range candidates.Items {
		exportData = append(exportData, candidate.CandidateExportData{
			ID:        c.ID.String(),
			Email:     string(c.Email),
			Phone:     string(c.Phone),
			FirstName: string(c.FirstName),
			LastName:  string(c.LastName),
			DNIType:   string(c.DNI.Type),
			DNINumber: c.DNI.Number,
			Status:    string(c.Status),
			CreatedAt: c.CreatedAt,
		})
	}

	return &candidate.ExportCandidatesResponse{
		Data:       exportData,
		TotalCount: len(exportData),
		ExportedAt: time.Now(),
		ExportedBy: exporterID,
	}, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// toCandidateResponse converts a Candidate entity to CandidateResponse DTO
func (s *CandidateService) toCandidateResponse(c *candidate.Candidate) *candidate.CandidateResponse {
	return &candidate.CandidateResponse{
		ID:        c.ID,
		Email:     c.Email,
		Phone:     c.Phone,
		FirstName: c.FirstName,
		LastName:  c.LastName,
		DNI:       c.DNI,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
