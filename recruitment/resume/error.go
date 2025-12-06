package resume

import (
	"net/http"

	"github.com/Abraxas-365/relay/pkg/errx"
)

var ErrRegistry = errx.NewRegistry("RESUME")

// Error codes - Resume Operations
var (
	CodeResumeNotFound            = ErrRegistry.Register("NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Resume not found")
	CodeResumeAlreadyExists       = ErrRegistry.Register("ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Resume already exists")
	CodeInvalidResumeData         = ErrRegistry.Register("INVALID_DATA", errx.TypeValidation, http.StatusBadRequest, "Invalid resume data")
	CodeResumeParseFailed         = ErrRegistry.Register("PARSE_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Failed to parse resume")
	CodeEmbeddingGenerationFailed = ErrRegistry.Register("EMBEDDING_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Failed to generate embeddings")
	CodeInsufficientPermissions   = ErrRegistry.Register("INSUFFICIENT_PERMISSIONS", errx.TypeAuthorization, http.StatusForbidden, "Insufficient permissions")
	CodeResumeIncomplete          = ErrRegistry.Register("INCOMPLETE", errx.TypeValidation, http.StatusBadRequest, "Resume is incomplete")
	CodeMaxResumesExceeded        = ErrRegistry.Register("MAX_RESUMES_EXCEEDED", errx.TypeBusiness, http.StatusUnprocessableEntity, "Maximum number of resumes exceeded")
	CodeDefaultResumeRequired     = ErrRegistry.Register("DEFAULT_REQUIRED", errx.TypeBusiness, http.StatusUnprocessableEntity, "Cannot delete default resume")
	CodeFileReadFailed            = ErrRegistry.Register("FILE_READ_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Failed to read file")
	CodeFileNotFound              = ErrRegistry.Register("FILE_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "File not found")
	CodeInvalidFileFormat         = ErrRegistry.Register("INVALID_FILE_FORMAT", errx.TypeValidation, http.StatusBadRequest, "Invalid file format")
	CodeTenantMismatch            = ErrRegistry.Register("TENANT_MISMATCH", errx.TypeAuthorization, http.StatusForbidden, "Resume does not belong to this tenant")
	CodeSearchFailed              = ErrRegistry.Register("SEARCH_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Search operation failed")
)

// Error codes - Job/Queue Operations
var (
	CodeJobNotFound          = ErrRegistry.Register("JOB_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Processing job not found")
	CodeJobAlreadyProcessing = ErrRegistry.Register("JOB_ALREADY_PROCESSING", errx.TypeConflict, http.StatusConflict, "Job is already being processed")
	CodeJobAlreadyCompleted  = ErrRegistry.Register("JOB_ALREADY_COMPLETED", errx.TypeBusiness, http.StatusUnprocessableEntity, "Job has already been completed")
	CodeJobFailed            = ErrRegistry.Register("JOB_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Job processing failed")
	CodeJobCancelled         = ErrRegistry.Register("JOB_CANCELLED", errx.TypeBusiness, http.StatusUnprocessableEntity, "Job has been cancelled")
	CodeJobTimeout           = ErrRegistry.Register("JOB_TIMEOUT", errx.TypeInternal, http.StatusRequestTimeout, "Job processing timeout")
	CodeJobMaxRetriesReached = ErrRegistry.Register("JOB_MAX_RETRIES", errx.TypeInternal, http.StatusInternalServerError, "Job exceeded maximum retry attempts")
	CodeQueueEnqueueFailed   = ErrRegistry.Register("QUEUE_ENQUEUE_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Failed to enqueue job")
	CodeQueueDequeueFailed   = ErrRegistry.Register("QUEUE_DEQUEUE_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Failed to dequeue job")
	CodeQueueConnectionError = ErrRegistry.Register("QUEUE_CONNECTION_ERROR", errx.TypeInternal, http.StatusServiceUnavailable, "Queue service unavailable")
	CodeJobCreationFailed    = ErrRegistry.Register("JOB_CREATION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Failed to create job record")
	CodeJobUpdateFailed      = ErrRegistry.Register("JOB_UPDATE_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Failed to update job status")
	CodeInvalidJobStatus     = ErrRegistry.Register("INVALID_JOB_STATUS", errx.TypeValidation, http.StatusBadRequest, "Invalid job status")
	CodeJobRetryFailed       = ErrRegistry.Register("JOB_RETRY_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Failed to schedule job retry")
)

// Helper functions - Resume Operations
func ErrResumeNotFound() *errx.Error {
	return ErrRegistry.New(CodeResumeNotFound)
}

func ErrResumeAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeResumeAlreadyExists)
}

func ErrInvalidResumeData() *errx.Error {
	return ErrRegistry.New(CodeInvalidResumeData)
}

func ErrResumeParseFailed() *errx.Error {
	return ErrRegistry.New(CodeResumeParseFailed)
}

func ErrEmbeddingGenerationFailed() *errx.Error {
	return ErrRegistry.New(CodeEmbeddingGenerationFailed)
}

func ErrInsufficientPermissions() *errx.Error {
	return ErrRegistry.New(CodeInsufficientPermissions)
}

func ErrResumeIncomplete() *errx.Error {
	return ErrRegistry.New(CodeResumeIncomplete)
}

func ErrMaxResumesExceeded() *errx.Error {
	return ErrRegistry.New(CodeMaxResumesExceeded)
}

func ErrDefaultResumeRequired() *errx.Error {
	return ErrRegistry.New(CodeDefaultResumeRequired)
}

func ErrFileReadFailed() *errx.Error {
	return ErrRegistry.New(CodeFileReadFailed)
}

func ErrFileNotFound() *errx.Error {
	return ErrRegistry.New(CodeFileNotFound)
}

func ErrInvalidFileFormat() *errx.Error {
	return ErrRegistry.New(CodeInvalidFileFormat)
}

func ErrTenantMismatch() *errx.Error {
	return ErrRegistry.New(CodeTenantMismatch)
}

func ErrSearchFailed() *errx.Error {
	return ErrRegistry.New(CodeSearchFailed)
}

// Helper functions - Job/Queue Operations
func ErrJobNotFound() *errx.Error {
	return ErrRegistry.New(CodeJobNotFound)
}

func ErrJobAlreadyProcessing() *errx.Error {
	return ErrRegistry.New(CodeJobAlreadyProcessing)
}

func ErrJobAlreadyCompleted() *errx.Error {
	return ErrRegistry.New(CodeJobAlreadyCompleted)
}

func ErrJobFailed() *errx.Error {
	return ErrRegistry.New(CodeJobFailed)
}

func ErrJobCancelled() *errx.Error {
	return ErrRegistry.New(CodeJobCancelled)
}

func ErrJobTimeout() *errx.Error {
	return ErrRegistry.New(CodeJobTimeout)
}

func ErrJobMaxRetriesReached() *errx.Error {
	return ErrRegistry.New(CodeJobMaxRetriesReached)
}

func ErrQueueEnqueueFailed() *errx.Error {
	return ErrRegistry.New(CodeQueueEnqueueFailed)
}

func ErrQueueDequeueFailed() *errx.Error {
	return ErrRegistry.New(CodeQueueDequeueFailed)
}

func ErrQueueConnectionError() *errx.Error {
	return ErrRegistry.New(CodeQueueConnectionError)
}

func ErrJobCreationFailed() *errx.Error {
	return ErrRegistry.New(CodeJobCreationFailed)
}

func ErrJobUpdateFailed() *errx.Error {
	return ErrRegistry.New(CodeJobUpdateFailed)
}

func ErrInvalidJobStatus() *errx.Error {
	return ErrRegistry.New(CodeInvalidJobStatus)
}

func ErrJobRetryFailed() *errx.Error {
	return ErrRegistry.New(CodeJobRetryFailed)
}
