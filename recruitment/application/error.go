package application

import (
	"net/http"

	"github.com/Abraxas-365/relay/pkg/errx"
)

// Error Registry
var ErrRegistry = errx.NewRegistry("APPLICATION")

// Error codes
var (
	CodeApplicationNotFound        = ErrRegistry.Register("NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Application not found")
	CodeApplicationAlreadyExists   = ErrRegistry.Register("ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Application already exists")
	CodeApplicationArchived        = ErrRegistry.Register("ARCHIVED", errx.TypeBusiness, http.StatusForbidden, "Application is archived")
	CodeApplicationNotArchived     = ErrRegistry.Register("NOT_ARCHIVED", errx.TypeBusiness, http.StatusBadRequest, "Application is not archived")
	CodeApplicationAlreadyArchived = ErrRegistry.Register("ALREADY_ARCHIVED", errx.TypeBusiness, http.StatusConflict, "Application is already archived")
	CodeInsufficientPermissions    = ErrRegistry.Register("INSUFFICIENT_PERMISSIONS", errx.TypeAuthorization, http.StatusForbidden, "Insufficient permissions")
	CodeCandidateCannotApply       = ErrRegistry.Register("CANDIDATE_CANNOT_APPLY", errx.TypeBusiness, http.StatusForbidden, "Candidate cannot apply to jobs")
	CodeJobNotPublished            = ErrRegistry.Register("JOB_NOT_PUBLISHED", errx.TypeBusiness, http.StatusForbidden, "Job is not published")
	CodeJobArchived                = ErrRegistry.Register("JOB_ARCHIVED", errx.TypeBusiness, http.StatusForbidden, "Job is archived")
	CodeResumeNotFound             = ErrRegistry.Register("RESUME_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Resume not found")
	CodeFileSizeTooLarge           = ErrRegistry.Register("FILE_SIZE_TOO_LARGE", errx.TypeValidation, http.StatusBadRequest, "File size exceeds maximum allowed")
	CodeInvalidFileType            = ErrRegistry.Register("INVALID_FILE_TYPE", errx.TypeValidation, http.StatusBadRequest, "Invalid file type")
	CodeReviewerInvalidPermissions = ErrRegistry.Register("REVIEWER_INVALID_PERMISSIONS", errx.TypeAuthorization, http.StatusForbidden, "Reviewer does not have review permissions")
	CodeInvalidStatusTransition    = ErrRegistry.Register("INVALID_STATUS_TRANSITION", errx.TypeBusiness, http.StatusBadRequest, "Invalid status transition")
	CodeCannotWithdraw             = ErrRegistry.Register("CANNOT_WITHDRAW", errx.TypeBusiness, http.StatusBadRequest, "Cannot withdraw application in current state")
	CodeInvalidRequest             = ErrRegistry.Register("INVALID_REQUEST", errx.TypeValidation, http.StatusBadRequest, "Invalid request data")
	CodeValidationFailed           = ErrRegistry.Register("VALIDATION_FAILED", errx.TypeValidation, http.StatusBadRequest, "Request validation failed")
	CodeInvalidPagination          = ErrRegistry.Register("INVALID_PAGINATION", errx.TypeValidation, http.StatusBadRequest, "Invalid pagination parameters")
)

// Helper functions
func ErrApplicationNotFound() *errx.Error {
	return ErrRegistry.New(CodeApplicationNotFound)
}

func ErrApplicationAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeApplicationAlreadyExists)
}

func ErrApplicationArchived() *errx.Error {
	return ErrRegistry.New(CodeApplicationArchived)
}

func ErrApplicationNotArchived() *errx.Error {
	return ErrRegistry.New(CodeApplicationNotArchived)
}

func ErrApplicationAlreadyArchived() *errx.Error {
	return ErrRegistry.New(CodeApplicationAlreadyArchived)
}

func ErrInsufficientPermissions() *errx.Error {
	return ErrRegistry.New(CodeInsufficientPermissions)
}

func ErrCandidateCannotApply() *errx.Error {
	return ErrRegistry.New(CodeCandidateCannotApply)
}

func ErrJobNotPublished() *errx.Error {
	return ErrRegistry.New(CodeJobNotPublished)
}

func ErrJobArchived() *errx.Error {
	return ErrRegistry.New(CodeJobArchived)
}

func ErrResumeNotFound() *errx.Error {
	return ErrRegistry.New(CodeResumeNotFound)
}

func ErrFileSizeTooLarge() *errx.Error {
	return ErrRegistry.New(CodeFileSizeTooLarge)
}

func ErrInvalidFileType() *errx.Error {
	return ErrRegistry.New(CodeInvalidFileType)
}

func ErrReviewerInvalidPermissions() *errx.Error {
	return ErrRegistry.New(CodeReviewerInvalidPermissions)
}

func ErrInvalidStatusTransition() *errx.Error {
	return ErrRegistry.New(CodeInvalidStatusTransition)
}

func ErrCannotWithdraw() *errx.Error {
	return ErrRegistry.New(CodeCannotWithdraw)
}

func ErrInvalidRequest() *errx.Error {
	return ErrRegistry.New(CodeInvalidRequest)
}

func ErrValidationFailed() *errx.Error {
	return ErrRegistry.New(CodeValidationFailed)
}

func ErrInvalidPagination() *errx.Error {
	return ErrRegistry.New(CodeInvalidPagination)
}
