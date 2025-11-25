package application

import (
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"slices"
)

// ApplicationStatus represents the status of an application
type ApplicationStatus string

const (
	ApplicationStatusSubmitted    ApplicationStatus = "SUBMITTED"    // Initial submission
	ApplicationStatusUnderReview  ApplicationStatus = "UNDER_REVIEW" // Being reviewed
	ApplicationStatusShortlisted  ApplicationStatus = "SHORTLISTED"  // Passed initial review
	ApplicationStatusInterviewing ApplicationStatus = "INTERVIEWING" // In interview process
	ApplicationStatusApproved     ApplicationStatus = "APPROVED"     // Approved/Accepted
	ApplicationStatusRejected     ApplicationStatus = "REJECTED"     // Rejected
	ApplicationStatusWithdrawn    ApplicationStatus = "WITHDRAWN"    // Withdrawn by candidate
	ApplicationStatusArchived     ApplicationStatus = "ARCHIVED"     // Archived
)

type Application struct {
	ID              kernel.ApplicationID   `db:"id" json:"id"`
	JobID           kernel.JobID           `db:"job_id" json:"job_id"`
	CandidateID     kernel.CandidateID     `db:"candidate_id" json:"candidate_id"`
	ResumeSummary   kernel.ResumeSummary   `db:"resume_summary" json:"resume_summary"`
	ResumeEmbedding kernel.ResumeEmbedding `db:"resume_embedding" json:"resume_embedding"`
	ResumeBucketUrl kernel.BucketURL       `db:"resume_bucket_url" json:"resume_bucket_url"`
	Status          ApplicationStatus      `db:"status" json:"status"`
	ReviewerID      *kernel.UserID         `db:"reviewer_id" json:"reviewer_id,omitempty"`
	SubmittedBy     *kernel.UserID         `db:"submitted_by" json:"submitted_by"`
	StatusChangedAt *time.Time             `db:"status_changed_at" json:"status_changed_at,omitempty"`
	ArchivedAt      *time.Time             `db:"archived_at" json:"archived_at,omitempty"`
	CreatedAt       time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time              `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// Domain Methods
// ============================================================================

// IsArchived checks if the application is archived
func (a *Application) IsArchived() bool {
	return a.Status == ApplicationStatusArchived
}

// IsActive checks if the application is in an active state
func (a *Application) IsActive() bool {
	return !a.IsArchived() &&
		a.Status != ApplicationStatusRejected &&
		a.Status != ApplicationStatusWithdrawn
}

// HasReviewer checks if a reviewer is assigned
func (a *Application) HasReviewer() bool {
	return a.ReviewerID != nil
}

// CanBeReviewed checks if the application can be reviewed
func (a *Application) CanBeReviewed() bool {
	return a.IsActive() && a.Status != ApplicationStatusApproved
}

// CanUpdateStatus checks if status can be changed
func (a *Application) CanUpdateStatus(newStatus ApplicationStatus) bool {
	if a.IsArchived() {
		return false
	}

	// Define valid state transitions
	validTransitions := map[ApplicationStatus][]ApplicationStatus{
		ApplicationStatusSubmitted: {
			ApplicationStatusUnderReview,
			ApplicationStatusRejected,
			ApplicationStatusWithdrawn,
		},
		ApplicationStatusUnderReview: {
			ApplicationStatusShortlisted,
			ApplicationStatusRejected,
			ApplicationStatusWithdrawn,
		},
		ApplicationStatusShortlisted: {
			ApplicationStatusInterviewing,
			ApplicationStatusRejected,
			ApplicationStatusWithdrawn,
		},
		ApplicationStatusInterviewing: {
			ApplicationStatusApproved,
			ApplicationStatusRejected,
			ApplicationStatusWithdrawn,
		},
	}

	allowedStatuses, ok := validTransitions[a.Status]
	if !ok {
		return false // Current status doesn't allow transitions
	}

	return slices.Contains(allowedStatuses, newStatus)
}

// UpdateStatus updates the application status
func (a *Application) UpdateStatus(newStatus ApplicationStatus) error {
	if !a.CanUpdateStatus(newStatus) {
		return ErrInvalidStatusTransition().
			WithDetail("current_status", a.Status).
			WithDetail("new_status", newStatus)
	}

	now := time.Now()
	a.Status = newStatus
	a.StatusChangedAt = &now
	a.UpdatedAt = now
	return nil
}

// Archive marks the application as archived
func (a *Application) Archive() error {
	if a.IsArchived() {
		return ErrApplicationAlreadyArchived()
	}

	now := time.Now()
	a.Status = ApplicationStatusArchived
	a.ArchivedAt = &now
	a.UpdatedAt = now
	return nil
}

// Unarchive removes archived status
func (a *Application) Unarchive() error {
	if !a.IsArchived() {
		return ErrApplicationNotArchived()
	}

	a.Status = ApplicationStatusSubmitted
	a.ArchivedAt = nil
	a.UpdatedAt = time.Now()
	return nil
}

// AssignReviewer assigns a reviewer to the application
func (a *Application) AssignReviewer(reviewerID kernel.UserID) error {
	if a.IsArchived() {
		return ErrApplicationArchived()
	}

	a.ReviewerID = &reviewerID
	a.UpdatedAt = time.Now()
	return nil
}

// Withdraw marks the application as withdrawn
func (a *Application) Withdraw() error {
	if a.Status == ApplicationStatusApproved || a.Status == ApplicationStatusRejected {
		return ErrCannotWithdraw().
			WithDetail("status", a.Status).
			WithDetail("message", "Cannot withdraw approved or rejected applications")
	}

	now := time.Now()
	a.Status = ApplicationStatusWithdrawn
	a.StatusChangedAt = &now
	a.UpdatedAt = now
	return nil
}

// Approve approves the application
func (a *Application) Approve() error {
	if err := a.UpdateStatus(ApplicationStatusApproved); err != nil {
		return err
	}
	return nil
}

// Reject rejects the application
func (a *Application) Reject() error {
	if err := a.UpdateStatus(ApplicationStatusRejected); err != nil {
		return err
	}
	return nil
}
