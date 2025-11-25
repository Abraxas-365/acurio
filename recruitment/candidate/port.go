package candidate

import (
	"context"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

type Repository interface {
	// Create creates a new candidate
	Create(ctx context.Context, candidate *Candidate) error

	// Update updates an existing candidate
	Update(ctx context.Context, id kernel.CandidateID, candidate *Candidate) error

	// GetByID retrieves a candidate by ID
	GetByID(ctx context.Context, id kernel.CandidateID) (*Candidate, error)

	// Delete deletes a candidate by ID
	Delete(ctx context.Context, id kernel.CandidateID) error

	// List retrieves all candidates with pagination
	List(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[Candidate], error)

	// Search searches candidates by various criteria
	Search(ctx context.Context, req SearchCandidatesRequest) (*kernel.Paginated[Candidate], error)

	// GetByEmail retrieves a candidate by email
	GetByEmail(ctx context.Context, email kernel.Email) (*Candidate, error)

	// GetByDNI retrieves a candidate by DNI
	GetByDNI(ctx context.Context, dni kernel.DNI) (*Candidate, error)

	// Exists checks if a candidate exists by ID
	Exists(ctx context.Context, id kernel.CandidateID) (bool, error)

	// CountApplications counts applications for a candidate
	CountApplications(ctx context.Context, candidateID kernel.CandidateID) (int64, error)

	// ListArchived retrieves archived candidates with pagination
	ListArchived(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[Candidate], error)

	// Archive archives a candidate (soft delete alternative)
	Archive(ctx context.Context, id kernel.CandidateID) error

	// Unarchive unarchives a candidate
	Unarchive(ctx context.Context, id kernel.CandidateID) error
}
