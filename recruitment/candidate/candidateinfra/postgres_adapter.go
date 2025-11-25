package candidateinfra

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/candidate"
	"github.com/jmoiron/sqlx"
)

type PostgresCandidateRepository struct {
	db *sqlx.DB
}

func NewPostgresCandidateRepository(db *sqlx.DB) candidate.Repository {
	return &PostgresCandidateRepository{db: db}
}

// Create creates a new candidate
func (r *PostgresCandidateRepository) Create(ctx context.Context, c *candidate.Candidate) error {
	query := `
		INSERT INTO candidates (
			id, email, phone, first_name, last_name, 
			dni_type, dni_number, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		c.ID,
		c.Email,
		c.Phone,
		c.FirstName,
		c.LastName,
		c.DNI.Type,
		c.DNI.Number,
		c.Status,
		c.CreatedAt,
		c.UpdatedAt,
	)

	return err
}

// Update updates an existing candidate
func (r *PostgresCandidateRepository) Update(ctx context.Context, id kernel.CandidateID, c *candidate.Candidate) error {
	query := `
		UPDATE candidates 
		SET 
			email = $2,
			phone = $3,
			first_name = $4,
			last_name = $5,
			dni_type = $6,
			dni_number = $7,
			status = $8,
			archived_at = $9,
			updated_at = $10
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		id,
		c.Email,
		c.Phone,
		c.FirstName,
		c.LastName,
		c.DNI.Type,
		c.DNI.Number,
		c.Status,
		c.ArchivedAt,
		c.UpdatedAt,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return candidate.ErrCandidateNotFound()
	}

	return nil
}

// GetByID retrieves a candidate by ID
func (r *PostgresCandidateRepository) GetByID(ctx context.Context, id kernel.CandidateID) (*candidate.Candidate, error) {
	query := `
		SELECT 
			id, email, phone, first_name, last_name,
			dni_type, dni_number, status, archived_at,
			created_at, updated_at
		FROM candidates
		WHERE id = $1
	`

	var c candidate.Candidate
	var dniType sql.NullString
	var dniNumber sql.NullString
	var archivedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID,
		&c.Email,
		&c.Phone,
		&c.FirstName,
		&c.LastName,
		&dniType,
		&dniNumber,
		&c.Status,
		&archivedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, candidate.ErrCandidateNotFound()
	}
	if err != nil {
		return nil, err
	}

	// Map DNI
	if dniType.Valid && dniNumber.Valid {
		c.DNI = kernel.DNI{
			Type:   kernel.DNIType(dniType.String),
			Number: dniNumber.String,
		}
	}

	if archivedAt.Valid {
		c.ArchivedAt = &archivedAt.Time
	}

	return &c, nil
}

// Delete deletes a candidate by ID
func (r *PostgresCandidateRepository) Delete(ctx context.Context, id kernel.CandidateID) error {
	query := `DELETE FROM candidates WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return candidate.ErrCandidateNotFound()
	}

	return nil
}

// List retrieves all candidates with pagination
func (r *PostgresCandidateRepository) List(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[candidate.Candidate], error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM candidates`
	if err := r.db.GetContext(ctx, &total, countQuery); err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (pagination.Page - 1) * pagination.PageSize

	// Fetch candidates
	query := `
		SELECT 
			id, email, phone, first_name, last_name,
			dni_type, dni_number, status, archived_at,
			created_at, updated_at
		FROM candidates
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryxContext(ctx, query, pagination.PageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]candidate.Candidate, 0)
	for rows.Next() {
		var c candidate.Candidate
		var dniType sql.NullString
		var dniNumber sql.NullString
		var archivedAt sql.NullTime

		err := rows.Scan(
			&c.ID,
			&c.Email,
			&c.Phone,
			&c.FirstName,
			&c.LastName,
			&dniType,
			&dniNumber,
			&c.Status,
			&archivedAt,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Map DNI
		if dniType.Valid && dniNumber.Valid {
			c.DNI = kernel.DNI{
				Type:   kernel.DNIType(dniType.String),
				Number: dniNumber.String,
			}
		}

		if archivedAt.Valid {
			c.ArchivedAt = &archivedAt.Time
		}

		candidates = append(candidates, c)
	}

	return &kernel.Paginated[candidate.Candidate]{
		Items: candidates,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
			Pages:  (total + pagination.PageSize - 1) / pagination.PageSize,
		},
		Empty: len(candidates) == 0,
	}, nil
}

// Search searches candidates by various criteria
func (r *PostgresCandidateRepository) Search(ctx context.Context, req candidate.SearchCandidatesRequest) (*kernel.Paginated[candidate.Candidate], error) {
	// Build WHERE clause dynamically
	whereClauses := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Query != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(`(
			first_name ILIKE $%d OR 
			last_name ILIKE $%d OR 
			email ILIKE $%d
		)`, argCount, argCount, argCount))
		args = append(args, "%"+req.Query+"%")
		argCount++
	}

	if req.Email != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(`email ILIKE $%d`, argCount))
		args = append(args, "%"+req.Email+"%")
		argCount++
	}

	if req.Phone != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(`phone ILIKE $%d`, argCount))
		args = append(args, "%"+req.Phone+"%")
		argCount++
	}

	if req.DNIType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(`dni_type = $%d`, argCount))
		args = append(args, req.DNIType)
		argCount++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM candidates %s`, whereSQL)
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (req.Pagination.Page - 1) * req.Pagination.PageSize

	// Fetch candidates
	query := fmt.Sprintf(`
		SELECT 
			id, email, phone, first_name, last_name,
			dni_type, dni_number, status, archived_at,
			created_at, updated_at
		FROM candidates
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argCount, argCount+1)

	args = append(args, req.Pagination.PageSize, offset)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]candidate.Candidate, 0)
	for rows.Next() {
		var c candidate.Candidate
		var dniType sql.NullString
		var dniNumber sql.NullString
		var archivedAt sql.NullTime

		err := rows.Scan(
			&c.ID,
			&c.Email,
			&c.Phone,
			&c.FirstName,
			&c.LastName,
			&dniType,
			&dniNumber,
			&c.Status,
			&archivedAt,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if dniType.Valid && dniNumber.Valid {
			c.DNI = kernel.DNI{
				Type:   kernel.DNIType(dniType.String),
				Number: dniNumber.String,
			}
		}

		if archivedAt.Valid {
			c.ArchivedAt = &archivedAt.Time
		}

		candidates = append(candidates, c)
	}

	return &kernel.Paginated[candidate.Candidate]{
		Items: candidates,
		Page: kernel.Page{
			Number: req.Pagination.Page,
			Size:   req.Pagination.PageSize,
			Total:  total,
			Pages:  (total + req.Pagination.PageSize - 1) / req.Pagination.PageSize,
		},
		Empty: len(candidates) == 0,
	}, nil
}

// GetByEmail retrieves a candidate by email
func (r *PostgresCandidateRepository) GetByEmail(ctx context.Context, email kernel.Email) (*candidate.Candidate, error) {
	query := `
		SELECT 
			id, email, phone, first_name, last_name,
			dni_type, dni_number, status, archived_at,
			created_at, updated_at
		FROM candidates
		WHERE email = $1
	`

	var c candidate.Candidate
	var dniType sql.NullString
	var dniNumber sql.NullString
	var archivedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&c.ID,
		&c.Email,
		&c.Phone,
		&c.FirstName,
		&c.LastName,
		&dniType,
		&dniNumber,
		&c.Status,
		&archivedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, candidate.ErrCandidateNotFound()
	}
	if err != nil {
		return nil, err
	}

	if dniType.Valid && dniNumber.Valid {
		c.DNI = kernel.DNI{
			Type:   kernel.DNIType(dniType.String),
			Number: dniNumber.String,
		}
	}

	if archivedAt.Valid {
		c.ArchivedAt = &archivedAt.Time
	}

	return &c, nil
}

// GetByDNI retrieves a candidate by DNI
func (r *PostgresCandidateRepository) GetByDNI(ctx context.Context, dni kernel.DNI) (*candidate.Candidate, error) {
	query := `
		SELECT 
			id, email, phone, first_name, last_name,
			dni_type, dni_number, status, archived_at,
			created_at, updated_at
		FROM candidates
		WHERE dni_type = $1 AND dni_number = $2
	`

	var c candidate.Candidate
	var archivedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, dni.Type, dni.Number).Scan(
		&c.ID,
		&c.Email,
		&c.Phone,
		&c.FirstName,
		&c.LastName,
		&c.DNI.Type,
		&c.DNI.Number,
		&c.Status,
		&archivedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, candidate.ErrCandidateNotFound()
	}
	if err != nil {
		return nil, err
	}

	if archivedAt.Valid {
		c.ArchivedAt = &archivedAt.Time
	}

	return &c, nil
}

// Exists checks if a candidate exists by ID
func (r *PostgresCandidateRepository) Exists(ctx context.Context, id kernel.CandidateID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM candidates WHERE id = $1)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id)
	return exists, err
}

// CountApplications counts applications for a candidate
func (r *PostgresCandidateRepository) CountApplications(ctx context.Context, candidateID kernel.CandidateID) (int64, error) {
	query := `SELECT COUNT(*) FROM applications WHERE candidate_id = $1`

	var count int64
	err := r.db.GetContext(ctx, &count, query, candidateID)
	return count, err
}

// ListArchived retrieves archived candidates with pagination
func (r *PostgresCandidateRepository) ListArchived(ctx context.Context, pagination kernel.PaginationOptions) (*kernel.Paginated[candidate.Candidate], error) {
	// Count total archived
	var total int
	countQuery := `SELECT COUNT(*) FROM candidates WHERE status = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, candidate.CandidateStatusArchived); err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (pagination.Page - 1) * pagination.PageSize

	// Fetch archived candidates
	query := `
		SELECT 
			id, email, phone, first_name, last_name,
			dni_type, dni_number, status, archived_at,
			created_at, updated_at
		FROM candidates
		WHERE status = $1
		ORDER BY archived_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryxContext(ctx, query, candidate.CandidateStatusArchived, pagination.PageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]candidate.Candidate, 0)
	for rows.Next() {
		var c candidate.Candidate
		var dniType sql.NullString
		var dniNumber sql.NullString
		var archivedAt sql.NullTime

		err := rows.Scan(
			&c.ID,
			&c.Email,
			&c.Phone,
			&c.FirstName,
			&c.LastName,
			&dniType,
			&dniNumber,
			&c.Status,
			&archivedAt,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if dniType.Valid && dniNumber.Valid {
			c.DNI = kernel.DNI{
				Type:   kernel.DNIType(dniType.String),
				Number: dniNumber.String,
			}
		}

		if archivedAt.Valid {
			c.ArchivedAt = &archivedAt.Time
		}

		candidates = append(candidates, c)
	}

	return &kernel.Paginated[candidate.Candidate]{
		Items: candidates,
		Page: kernel.Page{
			Number: pagination.Page,
			Size:   pagination.PageSize,
			Total:  total,
			Pages:  (total + pagination.PageSize - 1) / pagination.PageSize,
		},
		Empty: len(candidates) == 0,
	}, nil
}

// Archive archives a candidate
func (r *PostgresCandidateRepository) Archive(ctx context.Context, id kernel.CandidateID) error {
	query := `
		UPDATE candidates 
		SET status = $2, archived_at = $3, updated_at = $4
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, id, candidate.CandidateStatusArchived, now, now)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return candidate.ErrCandidateNotFound()
	}

	return nil
}

// Unarchive unarchives a candidate
func (r *PostgresCandidateRepository) Unarchive(ctx context.Context, id kernel.CandidateID) error {
	query := `
		UPDATE candidates 
		SET status = $2, archived_at = NULL, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, candidate.CandidateStatusActive, time.Now())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return candidate.ErrCandidateNotFound()
	}

	return nil
}
