package candidate

import (
	"net/http"

	"github.com/Abraxas-365/relay/pkg/errx"
)

var ErrRegistry = errx.NewRegistry("CANDIDATE")

// Error codes
var (
	CodeCandidateNotFound        = ErrRegistry.Register("NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Candidate not found")
	CodeCandidateAlreadyExists   = ErrRegistry.Register("ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Candidate already exists")
	CodeEmailAlreadyExists       = ErrRegistry.Register("EMAIL_ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Email already registered")
	CodeDNIAlreadyExists         = ErrRegistry.Register("DNI_ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "DNI already registered")
	CodeInvalidDNI               = ErrRegistry.Register("INVALID_DNI", errx.TypeValidation, http.StatusBadRequest, "Invalid DNI format")
	CodeCandidateArchived        = ErrRegistry.Register("ARCHIVED", errx.TypeBusiness, http.StatusForbidden, "Candidate is archived")
	CodeCandidateNotArchived     = ErrRegistry.Register("NOT_ARCHIVED", errx.TypeBusiness, http.StatusBadRequest, "Candidate is not archived")
	CodeCandidateAlreadyArchived = ErrRegistry.Register("ALREADY_ARCHIVED", errx.TypeBusiness, http.StatusConflict, "Candidate is already archived")
	CodeCandidateHasApplications = ErrRegistry.Register("HAS_APPLICATIONS", errx.TypeBusiness, http.StatusConflict, "Cannot delete candidate with applications")
	CodeInsufficientPermissions  = ErrRegistry.Register("INSUFFICIENT_PERMISSIONS", errx.TypeAuthorization, http.StatusForbidden, "Insufficient permissions")
	CodeInvalidEmail             = ErrRegistry.Register("INVALID_EMAIL", errx.TypeValidation, http.StatusBadRequest, "Invalid email format")
	CodeInvalidPhone             = ErrRegistry.Register("INVALID_PHONE", errx.TypeValidation, http.StatusBadRequest, "Invalid phone format")
	CodeInvalidRequest           = ErrRegistry.Register("INVALID_REQUEST", errx.TypeValidation, http.StatusBadRequest, "Invalid request data")
	CodeValidationFailed         = ErrRegistry.Register("VALIDATION_FAILED", errx.TypeValidation, http.StatusBadRequest, "Request validation failed")
	CodeInvalidPagination        = ErrRegistry.Register("INVALID_PAGINATION", errx.TypeValidation, http.StatusBadRequest, "Invalid pagination parameters")
	CodeExportFailed             = ErrRegistry.Register("EXPORT_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Failed to export candidates")
	CodeInvalidStatus            = ErrRegistry.Register("INVALID_STATUS", errx.TypeValidation, http.StatusBadRequest, "Invalid candidate status")
	CodeInvalidDNIType           = ErrRegistry.Register("INVALID_DNI_TYPE", errx.TypeValidation, http.StatusBadRequest, "Invalid DNI type")

	CodeCandidateInactive    = ErrRegistry.Register("INACTIVE", errx.TypeBusiness, http.StatusForbidden, "Candidate is inactive")
	CodeCandidateBlacklisted = ErrRegistry.Register("BLACKLISTED", errx.TypeBusiness, http.StatusForbidden, "Candidate is blacklisted")
	CodeEmailNotVerified     = ErrRegistry.Register("EMAIL_NOT_VERIFIED", errx.TypeBusiness, http.StatusPreconditionFailed, "Email not verified")
)

// Helper functions
func ErrCandidateNotFound() *errx.Error {
	return ErrRegistry.New(CodeCandidateNotFound)
}

func ErrCandidateAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeCandidateAlreadyExists)
}

func ErrEmailAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeEmailAlreadyExists)
}

func ErrDNIAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeDNIAlreadyExists)
}

func ErrInvalidDNI() *errx.Error {
	return ErrRegistry.New(CodeInvalidDNI)
}

func ErrCandidateArchived() *errx.Error {
	return ErrRegistry.New(CodeCandidateArchived)
}

func ErrCandidateNotArchived() *errx.Error {
	return ErrRegistry.New(CodeCandidateNotArchived)
}

func ErrCandidateAlreadyArchived() *errx.Error {
	return ErrRegistry.New(CodeCandidateAlreadyArchived)
}

func ErrCandidateHasApplications() *errx.Error {
	return ErrRegistry.New(CodeCandidateHasApplications)
}

func ErrInsufficientPermissions() *errx.Error {
	return ErrRegistry.New(CodeInsufficientPermissions)
}

func ErrInvalidEmail() *errx.Error {
	return ErrRegistry.New(CodeInvalidEmail)
}

func ErrInvalidPhone() *errx.Error {
	return ErrRegistry.New(CodeInvalidPhone)
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

func ErrExportFailed() *errx.Error {
	return ErrRegistry.New(CodeExportFailed)
}

func ErrInvalidStatus() *errx.Error {
	return ErrRegistry.New(CodeInvalidStatus)
}

func ErrInvalidDNIType() *errx.Error {
	return ErrRegistry.New(CodeInvalidDNIType)
}

func ErrCandidateInactive() *errx.Error {
	return ErrRegistry.New(CodeCandidateInactive)
}

func ErrCandidateBlacklisted() *errx.Error {
	return ErrRegistry.New(CodeCandidateBlacklisted)
}

func ErrEmailNotVerified() *errx.Error {
	return ErrRegistry.New(CodeEmailNotVerified)
}
