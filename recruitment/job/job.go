package job

import (
	"github.com/Abraxas-365/relay/pkg/kernel"
	"time"
)

// JobStatus represents the status of a job posting
type JobStatus string

const (
	JobStatusDraft     JobStatus = "DRAFT"     // Created but not published
	JobStatusPublished JobStatus = "PUBLISHED" // Active and accepting applications
	JobStatusClosed    JobStatus = "CLOSED"    // No longer accepting applications
	JobStatusArchived  JobStatus = "ARCHIVED"  // Archived
)

type Job struct {
	ID                  kernel.JobID            `db:"id" json:"id"`
	Title               kernel.JobTitle         `db:"job_title" json:"job_title"`
	Description         kernel.JobDescription   `db:"job_description" json:"job_description"`
	Position            kernel.JobPosition      `db:"job_position" json:"job_position"`
	GeneralRequirements []kernel.JobRequirement `db:"general_requirements" json:"general_requirements"`
	Benefits            []kernel.JobBenefit     `db:"benefits" json:"benefits"`
	PostedBy            kernel.UserID           `db:"posted_by" json:"posted_by"`
	Status              JobStatus               `db:"status" json:"status"`
	PublishedAt         *time.Time              `db:"published_at" json:"published_at,omitempty"`
	ArchivedAt          *time.Time              `db:"archived_at" json:"archived_at,omitempty"`
	CreatedAt           time.Time               `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time               `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// Domain Methods
// ============================================================================

// IsPublished checks if the job is currently published
func (j *Job) IsPublished() bool {
	return j.Status == JobStatusPublished
}

// IsArchived checks if the job is archived
func (j *Job) IsArchived() bool {
	return j.Status == JobStatusArchived
}

// IsDraft checks if the job is in draft status
func (j *Job) IsDraft() bool {
	return j.Status == JobStatusDraft
}

// IsClosed checks if the job is closed
func (j *Job) IsClosed() bool {
	return j.Status == JobStatusClosed
}

// CanBePublished checks if a job can be published
func (j *Job) CanBePublished() bool {
	return j.Status == JobStatusDraft && !j.IsArchived()
}

// CanBeEdited checks if a job can be edited
func (j *Job) CanBeEdited() bool {
	return !j.IsArchived()
}

// Publish marks the job as published
func (j *Job) Publish() error {
	if !j.CanBePublished() {
		return ErrCannotPublish().WithDetail("current_status", j.Status)
	}

	now := time.Now()
	j.Status = JobStatusPublished
	j.PublishedAt = &now
	j.UpdatedAt = now
	return nil
}

// Unpublish marks the job as draft
func (j *Job) Unpublish() {
	j.Status = JobStatusDraft
	j.UpdatedAt = time.Now()
}

// Close marks the job as closed
func (j *Job) Close() {
	j.Status = JobStatusClosed
	j.UpdatedAt = time.Now()
}

// Archive marks the job as archived
func (j *Job) Archive() error {
	if j.IsArchived() {
		return ErrJobAlreadyArchived()
	}

	now := time.Now()
	j.Status = JobStatusArchived
	j.ArchivedAt = &now
	j.UpdatedAt = now
	return nil
}

// Unarchive removes archived status
func (j *Job) Unarchive() error {
	if !j.IsArchived() {
		return ErrJobNotArchived()
	}

	j.Status = JobStatusDraft
	j.ArchivedAt = nil
	j.UpdatedAt = time.Now()
	return nil
}

// UpdateDetails updates job details
func (j *Job) UpdateDetails(title kernel.JobTitle, description kernel.JobDescription, position kernel.JobPosition) {
	if title != "" {
		j.Title = title
	}
	if description != "" {
		j.Description = description
	}
	if position != "" {
		j.Position = position
	}
	j.UpdatedAt = time.Now()
}
