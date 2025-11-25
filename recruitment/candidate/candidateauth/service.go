package candidateauth

import (
	"context"
	"time"

	"github.com/Abraxas-365/relay/pkg/errx"
	"github.com/Abraxas-365/relay/pkg/iam/otp"
	"github.com/Abraxas-365/relay/pkg/iam/otp/otpsrv"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/candidate"
	"github.com/google/uuid"
)

type CandidateAuthService struct {
	candidateRepo candidate.Repository
	otpService    *otpsrv.OTPService
	tokenService  *CandidateTokenService // Use our wrapper instead
}

func NewCandidateAuthService(
	candidateRepo candidate.Repository,
	otpService *otpsrv.OTPService,
	tokenService *CandidateTokenService, // Wrapper service
) *CandidateAuthService {
	return &CandidateAuthService{
		candidateRepo: candidateRepo,
		otpService:    otpService,
		tokenService:  tokenService,
	}
}

// RequestVerificationCode sends an OTP code to the candidate's email
func (s *CandidateAuthService) RequestVerificationCode(ctx context.Context, email kernel.Email, phone kernel.Phone) error {
	_, err := s.otpService.GenerateOTP(ctx, string(email), otp.OTPPurposeJobApplication)
	return err
}

// VerifyAndCreateSession verifies the OTP and creates a session for the candidate
func (s *CandidateAuthService) VerifyAndCreateSession(
	ctx context.Context,
	email kernel.Email,
	phone kernel.Phone,
	code string,
) (*CandidateSession, error) {
	// Verify OTP
	_, err := s.otpService.VerifyOTP(ctx, string(email), code)
	if err != nil {
		return nil, err
	}

	// Find or create candidate
	candidateEntity, err := s.findOrCreateCandidate(ctx, email, phone)
	if err != nil {
		return nil, err
	}

	// Generate session token using our wrapper (30 minutes validity)
	token, err := s.tokenService.GenerateCandidateToken(
		candidateEntity.ID,
		email,
		30*time.Minute,
	)
	if err != nil {
		return nil, errx.Wrap(err, "failed to generate session token", errx.TypeInternal)
	}

	return &CandidateSession{
		CandidateID: candidateEntity.ID,
		Email:       email,
		Phone:       phone,
		Token:       token,
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	}, nil
}

// findOrCreateCandidate finds an existing candidate or creates a new one
func (s *CandidateAuthService) findOrCreateCandidate(
	ctx context.Context,
	email kernel.Email,
	phone kernel.Phone,
) (*candidate.Candidate, error) {
	// Try to find existing candidate
	candidateEntity, err := s.candidateRepo.GetByEmail(ctx, email)
	if err == nil && candidateEntity != nil {
		// Update phone if provided and different
		if phone != "" && phone != candidateEntity.Phone {
			candidateEntity.Phone = phone
			candidateEntity.UpdatedAt = time.Now()
			if err := s.candidateRepo.Update(ctx, candidateEntity.ID, candidateEntity); err != nil {
				return nil, errx.Wrap(err, "failed to update candidate phone", errx.TypeInternal)
			}
		}
		return candidateEntity, nil
	}

	// Create new candidate
	newCandidate := &candidate.Candidate{
		ID:        kernel.NewCandidateID(uuid.NewString()),
		Email:     email,
		Phone:     phone,
		Status:    candidate.CandidateStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.candidateRepo.Create(ctx, newCandidate); err != nil {
		return nil, errx.Wrap(err, "failed to create candidate", errx.TypeInternal)
	}

	return newCandidate, nil
}

// UpdateCandidateProfile updates candidate profile information
func (s *CandidateAuthService) UpdateCandidateProfile(
	ctx context.Context,
	candidateID kernel.CandidateID,
	firstName kernel.FirstName,
	lastName kernel.LastName,
	dni kernel.DNI,
) error {
	candidateEntity, err := s.candidateRepo.GetByID(ctx, candidateID)
	if err != nil {
		return candidate.ErrCandidateNotFound().WithDetail("candidate_id", candidateID.String())
	}

	// Update profile fields
	if firstName != "" {
		candidateEntity.FirstName = firstName
	}
	if lastName != "" {
		candidateEntity.LastName = lastName
	}
	if dni.IsValid() {
		// Check for duplicate DNI
		existing, _ := s.candidateRepo.GetByDNI(ctx, dni)
		if existing != nil && existing.ID != candidateID {
			return candidate.ErrDNIAlreadyExists().
				WithDetail("dni_type", string(dni.Type)).
				WithDetail("dni_number", dni.Number)
		}
		candidateEntity.DNI = dni
	}

	candidateEntity.UpdatedAt = time.Now()

	if err := s.candidateRepo.Update(ctx, candidateID, candidateEntity); err != nil {
		return errx.Wrap(err, "failed to update candidate profile", errx.TypeInternal)
	}

	return nil
}

// GetCandidateProfile gets the candidate's current profile
func (s *CandidateAuthService) GetCandidateProfile(ctx context.Context, candidateID kernel.CandidateID) (*candidate.Candidate, error) {
	candidateEntity, err := s.candidateRepo.GetByID(ctx, candidateID)
	if err != nil {
		return nil, candidate.ErrCandidateNotFound().WithDetail("candidate_id", candidateID.String())
	}
	return candidateEntity, nil
}

type CandidateSession struct {
	CandidateID kernel.CandidateID `json:"candidate_id"`
	Email       kernel.Email       `json:"email"`
	Phone       kernel.Phone       `json:"phone"`
	Token       string             `json:"token"`
	ExpiresAt   time.Time          `json:"expires_at"`
}
