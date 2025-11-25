package job

import (
	"github.com/Abraxas-365/relay/pkg/errx"
	"net/http"
)

// Error Registry
var ErrRegistry = errx.NewRegistry("JOB")

// Error codes
var (
	CodeJobNotFound             = ErrRegistry.Register("NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Job not found")
	CodeJobAlreadyExists        = ErrRegistry.Register("ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Job already exists")
	CodeJobArchived             = ErrRegistry.Register("ARCHIVED", errx.TypeBusiness, http.StatusForbidden, "Job is archived")
	CodeJobNotArchived          = ErrRegistry.Register("NOT_ARCHIVED", errx.TypeBusiness, http.StatusBadRequest, "Job is not archived")
	CodeJobAlreadyArchived      = ErrRegistry.Register("ALREADY_ARCHIVED", errx.TypeBusiness, http.StatusConflict, "Job is already archived")
	CodeJobAlreadyPublished     = ErrRegistry.Register("ALREADY_PUBLISHED", errx.TypeBusiness, http.StatusConflict, "Job is already published")
	CodeJobHasApplications      = ErrRegistry.Register("HAS_APPLICATIONS", errx.TypeBusiness, http.StatusConflict, "Cannot delete job with applications")
	CodeInsufficientPermissions = ErrRegistry.Register("INSUFFICIENT_PERMISSIONS", errx.TypeAuthorization, http.StatusForbidden, "Insufficient permissions")
	CodeUnauthorizedUpdate      = ErrRegistry.Register("UNAUTHORIZED_UPDATE", errx.TypeAuthorization, http.StatusForbidden, "Unauthorized to update this job")
	CodeCannotPublish           = ErrRegistry.Register("CANNOT_PUBLISH", errx.TypeBusiness, http.StatusBadRequest, "Job cannot be published in current state")
)

// Helper functions
func ErrJobNotFound() *errx.Error {
	return ErrRegistry.New(CodeJobNotFound)
}

func ErrJobAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeJobAlreadyExists)
}

func ErrJobArchived() *errx.Error {
	return ErrRegistry.New(CodeJobArchived)
}

func ErrJobNotArchived() *errx.Error {
	return ErrRegistry.New(CodeJobNotArchived)
}

func ErrJobAlreadyArchived() *errx.Error {
	return ErrRegistry.New(CodeJobAlreadyArchived)
}

func ErrJobAlreadyPublished() *errx.Error {
	return ErrRegistry.New(CodeJobAlreadyPublished)
}

func ErrJobHasApplications() *errx.Error {
	return ErrRegistry.New(CodeJobHasApplications)
}

func ErrInsufficientPermissions() *errx.Error {
	return ErrRegistry.New(CodeInsufficientPermissions)
}

func ErrUnauthorizedUpdate() *errx.Error {
	return ErrRegistry.New(CodeUnauthorizedUpdate)
}

func ErrCannotPublish() *errx.Error {
	return ErrRegistry.New(CodeCannotPublish)
}
