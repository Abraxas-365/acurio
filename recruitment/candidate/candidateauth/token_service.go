package candidateauth

import (
	"time"

	"github.com/Abraxas-365/relay/pkg/errx"
	"github.com/Abraxas-365/relay/pkg/iam/auth"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// CandidateTokenService wraps IAM's TokenService for candidate-specific tokens
type CandidateTokenService struct {
	iamTokenService auth.TokenService
}

// NewCandidateTokenService creates a wrapper around IAM's TokenService
func NewCandidateTokenService(iamTokenService auth.TokenService) *CandidateTokenService {
	return &CandidateTokenService{
		iamTokenService: iamTokenService,
	}
}

// GenerateCandidateToken generates a token for a candidate using IAM's generic method
func (s *CandidateTokenService) GenerateCandidateToken(
	candidateID kernel.CandidateID,
	email kernel.Email,
	duration time.Duration,
) (string, error) {
	// Use a dummy userID and tenantID, but add candidate-specific claims
	dummyUserID := kernel.UserID(candidateID)     // Convert candidate ID
	dummyTenantID := kernel.TenantID("CANDIDATE") // Special tenant marker

	claims := map[string]any{
		"candidate_id": candidateID,
		"email":        email,
		"type":         "candidate", // Mark as candidate token
		"expires_at":   time.Now().Add(duration),
	}

	// Generate token using IAM's service
	token, err := s.iamTokenService.GenerateAccessToken(dummyUserID, dummyTenantID, claims)
	if err != nil {
		return "", errx.Wrap(err, "failed to generate candidate token", errx.TypeInternal)
	}

	return token, nil
}

// ValidateCandidateToken validates a candidate token
func (s *CandidateTokenService) ValidateCandidateToken(tokenString string) (*CandidateClaims, error) {
	// Validate using IAM's service
	tokenClaims, err := s.iamTokenService.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Extract candidate-specific data from the generic token claims
	// Note: This depends on your TokenClaims structure
	candidateID := kernel.CandidateID(tokenClaims.UserID) // Convert back
	email := kernel.Email(tokenClaims.Email)

	return &CandidateClaims{
		CandidateID: candidateID,
		Email:       email,
		ExpiresAt:   tokenClaims.ExpiresAt,
	}, nil
}

// CandidateClaims represents candidate-specific claims
type CandidateClaims struct {
	CandidateID kernel.CandidateID
	Email       kernel.Email
	ExpiresAt   time.Time
}
