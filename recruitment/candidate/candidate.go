package candidate

import (
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// CandidateStatus represents the status of a candidate
type CandidateStatus string

const (
	CandidateStatusActive   CandidateStatus = "ACTIVE"   // Active in the system
	CandidateStatusInactive CandidateStatus = "INACTIVE" // Deactivated
	CandidateStatusArchived CandidateStatus = "ARCHIVED" // Archived
)

type Candidate struct {
	ID         kernel.CandidateID `db:"id" json:"id"`
	Email      kernel.Email       `db:"email" json:"email"`
	Phone      kernel.Phone       `db:"phone" json:"phone"`
	FirstName  kernel.FirstName   `db:"first_name" json:"first_name"`
	LastName   kernel.LastName    `db:"last_name" json:"last_name"`
	DNI        kernel.DNI         `db:"dni" json:"dni"`
	Status     CandidateStatus    `db:"status" json:"status"`
	ArchivedAt *time.Time         `db:"archived_at" json:"archived_at,omitempty"`
	CreatedAt  time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// Domain Methods
// ============================================================================

// IsActive checks if the candidate is active
func (c *Candidate) IsActive() bool {
	return c.Status == CandidateStatusActive
}

// IsArchived checks if the candidate is archived
func (c *Candidate) IsArchived() bool {
	return c.Status == CandidateStatusArchived
}

// GetFullName returns the candidate's full name
func (c *Candidate) GetFullName() string {
	return fmt.Sprintf("%s %s", c.FirstName, c.LastName)
}

// Archive marks the candidate as archived
func (c *Candidate) Archive() error {
	if c.IsArchived() {
		return ErrCandidateAlreadyArchived()
	}

	now := time.Now()
	c.Status = CandidateStatusArchived
	c.ArchivedAt = &now
	c.UpdatedAt = now
	return nil
}

// Unarchive removes archived status
func (c *Candidate) Unarchive() error {
	if !c.IsArchived() {
		return ErrCandidateNotArchived()
	}

	c.Status = CandidateStatusActive
	c.ArchivedAt = nil
	c.UpdatedAt = time.Now()
	return nil
}

// Deactivate marks the candidate as inactive
func (c *Candidate) Deactivate() {
	c.Status = CandidateStatusInactive
	c.UpdatedAt = time.Now()
}

// Activate marks the candidate as active
func (c *Candidate) Activate() {
	c.Status = CandidateStatusActive
	c.UpdatedAt = time.Now()
}

// UpdateContactInfo updates the candidate's contact information
func (c *Candidate) UpdateContactInfo(email kernel.Email, phone kernel.Phone) {
	if email != "" {
		c.Email = email
	}
	if phone != "" {
		c.Phone = phone
	}
	c.UpdatedAt = time.Now()
}

// UpdatePersonalInfo updates the candidate's personal information
func (c *Candidate) UpdatePersonalInfo(firstName kernel.FirstName, lastName kernel.LastName) {
	if firstName != "" {
		c.FirstName = firstName
	}
	if lastName != "" {
		c.LastName = lastName
	}
	c.UpdatedAt = time.Now()
}

// UpdateDNI updates the candidate's DNI
func (c *Candidate) UpdateDNI(dni kernel.DNI) error {
	if !dni.IsValid() {
		return ErrInvalidDNI()
	}
	c.DNI = dni
	c.UpdatedAt = time.Now()
	return nil
}

// CanApplyToJob checks if candidate can apply to jobs
func (c *Candidate) CanApplyToJob() bool {
	return c.IsActive() && !c.IsArchived()
}
