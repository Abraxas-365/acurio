package auth

// ============================================================================
// DOMAIN-SPECIFIC SCOPES - ATS (Applicant Tracking System)
// ============================================================================

const (
	// Job scopes
	ScopeJobsAll     = "jobs:*"
	ScopeJobsRead    = "jobs:read"
	ScopeJobsWrite   = "jobs:write"
	ScopeJobsDelete  = "jobs:delete"
	ScopeJobsPublish = "jobs:publish" // Publish/unpublish jobs
	ScopeJobsArchive = "jobs:archive" // Archive jobs

	// Candidate scopes
	ScopeCandidatesAll    = "candidates:*"
	ScopeCandidatesRead   = "candidates:read"
	ScopeCandidatesWrite  = "candidates:write"
	ScopeCandidatesDelete = "candidates:delete"
	ScopeCandidatesExport = "candidates:export" // Export candidate data
	ScopeCandidatesImport = "candidates:import" // Import candidates

	// Application scopes
	ScopeApplicationsAll     = "applications:*"
	ScopeApplicationsRead    = "applications:read"
	ScopeApplicationsWrite   = "applications:write"
	ScopeApplicationsDelete  = "applications:delete"
	ScopeApplicationsReview  = "applications:review"  // Review/evaluate applications
	ScopeApplicationsApprove = "applications:approve" // Approve/reject applications
	ScopeApplicationsAssign  = "applications:assign"  // Assign to reviewers

	// Interview scopes (if you add interviews later)
	ScopeInterviewsAll      = "interviews:*"
	ScopeInterviewsRead     = "interviews:read"
	ScopeInterviewsWrite    = "interviews:write"
	ScopeInterviewsDelete   = "interviews:delete"
	ScopeInterviewsSchedule = "interviews:schedule"
	ScopeInterviewsConduct  = "interviews:conduct"

	// Offer scopes (if you add offers later)
	ScopeOffersAll     = "offers:*"
	ScopeOffersRead    = "offers:read"
	ScopeOffersWrite   = "offers:write"
	ScopeOffersDelete  = "offers:delete"
	ScopeOffersApprove = "offers:approve"
	ScopeOffersSend    = "offers:send"
)

// DomainScopeCategories organizes domain-specific scopes
var DomainScopeCategories = map[string][]string{
	"Jobs": {
		ScopeJobsAll,
		ScopeJobsRead,
		ScopeJobsWrite,
		ScopeJobsDelete,
		ScopeJobsPublish,
		ScopeJobsArchive,
	},
	"Candidates": {
		ScopeCandidatesAll,
		ScopeCandidatesRead,
		ScopeCandidatesWrite,
		ScopeCandidatesDelete,
		ScopeCandidatesExport,
		ScopeCandidatesImport,
	},
	"Applications": {
		ScopeApplicationsAll,
		ScopeApplicationsRead,
		ScopeApplicationsWrite,
		ScopeApplicationsDelete,
		ScopeApplicationsReview,
		ScopeApplicationsApprove,
		ScopeApplicationsAssign,
	},
	"Interviews": {
		ScopeInterviewsAll,
		ScopeInterviewsRead,
		ScopeInterviewsWrite,
		ScopeInterviewsDelete,
		ScopeInterviewsSchedule,
		ScopeInterviewsConduct,
	},
	"Offers": {
		ScopeOffersAll,
		ScopeOffersRead,
		ScopeOffersWrite,
		ScopeOffersDelete,
		ScopeOffersApprove,
		ScopeOffersSend,
	},
}

// DomainScopeDescriptions provides descriptions for domain scopes
var DomainScopeDescriptions = map[string]string{
	// Jobs
	ScopeJobsAll:     "Full access to job management",
	ScopeJobsRead:    "View jobs",
	ScopeJobsWrite:   "Create and edit jobs",
	ScopeJobsDelete:  "Delete jobs",
	ScopeJobsPublish: "Publish and unpublish jobs",
	ScopeJobsArchive: "Archive jobs",

	// Candidates
	ScopeCandidatesAll:    "Full access to candidate management",
	ScopeCandidatesRead:   "View candidates",
	ScopeCandidatesWrite:  "Create and edit candidates",
	ScopeCandidatesDelete: "Delete candidates",
	ScopeCandidatesExport: "Export candidate data",
	ScopeCandidatesImport: "Import candidate data",

	// Applications
	ScopeApplicationsAll:     "Full access to application management",
	ScopeApplicationsRead:    "View applications",
	ScopeApplicationsWrite:   "Create and edit applications",
	ScopeApplicationsDelete:  "Delete applications",
	ScopeApplicationsReview:  "Review and evaluate applications",
	ScopeApplicationsApprove: "Approve or reject applications",
	ScopeApplicationsAssign:  "Assign applications to reviewers",

	// Interviews
	ScopeInterviewsAll:      "Full access to interview management",
	ScopeInterviewsRead:     "View interviews",
	ScopeInterviewsWrite:    "Create and edit interviews",
	ScopeInterviewsDelete:   "Delete interviews",
	ScopeInterviewsSchedule: "Schedule interviews",
	ScopeInterviewsConduct:  "Conduct interviews",

	// Offers
	ScopeOffersAll:     "Full access to offer management",
	ScopeOffersRead:    "View offers",
	ScopeOffersWrite:   "Create and edit offers",
	ScopeOffersDelete:  "Delete offers",
	ScopeOffersApprove: "Approve offers",
	ScopeOffersSend:    "Send offers to candidates",
}

// DomainScopeGroups defines domain-specific role groupings
var DomainScopeGroups = map[string][]string{
	// Recruitment roles
	"recruiter": {
		ScopeJobsRead,
		ScopeJobsWrite,
		ScopeCandidatesAll,
		ScopeApplicationsAll,
		ScopeInterviewsAll,
		ScopeReportsView,
	},
	"senior_recruiter": {
		ScopeJobsAll,
		ScopeCandidatesAll,
		ScopeApplicationsAll,
		ScopeInterviewsAll,
		ScopeOffersRead,
		ScopeOffersWrite,
		ScopeReportsAll,
	},
	"hiring_manager": {
		ScopeJobsRead,
		ScopeCandidatesRead,
		ScopeApplicationsRead,
		ScopeApplicationsReview,
		ScopeApplicationsApprove,
		ScopeInterviewsRead,
		ScopeInterviewsSchedule,
		ScopeOffersRead,
		ScopeOffersApprove,
		ScopeReportsView,
	},
	"interviewer": {
		ScopeJobsRead,
		ScopeCandidatesRead,
		ScopeApplicationsRead,
		ScopeApplicationsReview,
		ScopeInterviewsRead,
		ScopeInterviewsConduct,
	},

	// Job management roles
	"job_manager": {
		ScopeJobsAll,
		ScopeApplicationsRead,
		ScopeApplicationsReview,
		ScopeCandidatesRead,
		ScopeReportsView,
	},
	"job_creator": {
		ScopeJobsRead,
		ScopeJobsWrite,
		ScopeJobsPublish,
		ScopeApplicationsRead,
		ScopeCandidatesRead,
	},
	"job_viewer": {
		ScopeJobsRead,
		ScopeApplicationsRead,
		ScopeCandidatesRead,
	},

	// Candidate management roles
	"candidate_manager": {
		ScopeCandidatesAll,
		ScopeApplicationsRead,
		ScopeApplicationsWrite,
		ScopeJobsRead,
	},
	"candidate_viewer": {
		ScopeCandidatesRead,
		ScopeApplicationsRead,
		ScopeJobsRead,
	},

	// Application management roles
	"application_reviewer": {
		ScopeApplicationsRead,
		ScopeApplicationsReview,
		ScopeApplicationsWrite,
		ScopeCandidatesRead,
		ScopeJobsRead,
	},
	"application_manager": {
		ScopeApplicationsAll,
		ScopeCandidatesRead,
		ScopeJobsRead,
	},

	// HR roles
	"hr_admin": {
		ScopeJobsAll,
		ScopeCandidatesAll,
		ScopeApplicationsAll,
		ScopeInterviewsAll,
		ScopeOffersAll,
		ScopeUsersRead,
		ScopeReportsAll,
	},
	"hr_coordinator": {
		ScopeJobsRead,
		ScopeCandidatesRead,
		ScopeApplicationsRead,
		ScopeInterviewsAll,
		ScopeOffersRead,
		ScopeReportsView,
	},
}
