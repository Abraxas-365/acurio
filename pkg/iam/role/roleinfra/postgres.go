package roleinfra

import (
	"context"
	"database/sql"

	"github.com/Abraxas-365/relay/pkg/errx"
	"github.com/Abraxas-365/relay/pkg/iam/role"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// PostgresRoleRepository implementación de PostgreSQL para RoleRepository
type PostgresRoleRepository struct {
	db *sqlx.DB
}

// NewPostgresRoleRepository crea una nueva instancia del repositorio de roles
func NewPostgresRoleRepository(db *sqlx.DB) role.RoleRepository {
	return &PostgresRoleRepository{
		db: db,
	}
}

// FindByID busca un rol por ID y tenant
func (r *PostgresRoleRepository) FindByID(ctx context.Context, id kernel.RoleID, tenantID kernel.TenantID) (*role.Role, error) {
	query := `
		SELECT 
			id, tenant_id, name, description, is_active, created_at, updated_at
		FROM roles 
		WHERE id = $1 AND tenant_id = $2`

	var roleEntity role.Role
	err := r.db.GetContext(ctx, &roleEntity, query, id.String(), tenantID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, role.ErrRoleNotFound().WithDetail("role_id", id.String())
		}
		return nil, errx.Wrap(err, "failed to find role by id", errx.TypeInternal).
			WithDetail("role_id", id.String()).
			WithDetail("tenant_id", tenantID.String())
	}

	return &roleEntity, nil
}

// FindByTenant busca todos los roles de un tenant
func (r *PostgresRoleRepository) FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*role.Role, error) {
	query := `
		SELECT 
			id, tenant_id, name, description, is_active, created_at, updated_at
		FROM roles 
		WHERE tenant_id = $1
		ORDER BY name ASC`

	var roles []role.Role
	err := r.db.SelectContext(ctx, &roles, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find roles by tenant", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String())
	}

	// Convertir a slice de punteros
	result := make([]*role.Role, len(roles))
	for i := range roles {
		result[i] = &roles[i]
	}

	return result, nil
}

// FindByName busca un rol por nombre y tenant
func (r *PostgresRoleRepository) FindByName(ctx context.Context, name string, tenantID kernel.TenantID) (*role.Role, error) {
	query := `
		SELECT 
			id, tenant_id, name, description, is_active, created_at, updated_at
		FROM roles 
		WHERE name = $1 AND tenant_id = $2`

	var roleEntity role.Role
	err := r.db.GetContext(ctx, &roleEntity, query, name, tenantID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, role.ErrRoleNotFound().WithDetail("name", name)
		}
		return nil, errx.Wrap(err, "failed to find role by name", errx.TypeInternal).
			WithDetail("name", name).
			WithDetail("tenant_id", tenantID.String())
	}

	return &roleEntity, nil
}

// FindActive busca todos los roles activos de un tenant
func (r *PostgresRoleRepository) FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*role.Role, error) {
	query := `
		SELECT 
			id, tenant_id, name, description, is_active, created_at, updated_at
		FROM roles 
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY name ASC`

	var roles []role.Role
	err := r.db.SelectContext(ctx, &roles, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find active roles", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String())
	}

	// Convertir a slice de punteros
	result := make([]*role.Role, len(roles))
	for i := range roles {
		result[i] = &roles[i]
	}

	return result, nil
}

// Save guarda o actualiza un rol
func (r *PostgresRoleRepository) Save(ctx context.Context, roleEntity role.Role) error {
	// Verificar si el rol ya existe
	exists, err := r.roleExists(ctx, roleEntity.ID, roleEntity.TenantID)
	if err != nil {
		return errx.Wrap(err, "failed to check role existence", errx.TypeInternal)
	}

	if exists {
		return r.update(ctx, roleEntity)
	}
	return r.create(ctx, roleEntity)
}

// create crea un nuevo rol
func (r *PostgresRoleRepository) create(ctx context.Context, roleEntity role.Role) error {
	query := `
		INSERT INTO roles (
			id, tenant_id, name, description, is_active, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :name, :description, :is_active, :created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, roleEntity)
	if err != nil {
		// Verificar violación de constraint de nombre único
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" && pqErr.Constraint == "roles_name_tenant_id_key" {
				return role.ErrRoleAlreadyExists().
					WithDetail("name", roleEntity.Name).
					WithDetail("tenant_id", roleEntity.TenantID.String())
			}
		}
		return errx.Wrap(err, "failed to create role", errx.TypeInternal).
			WithDetail("role_id", roleEntity.ID.String()).
			WithDetail("name", roleEntity.Name)
	}

	return nil
}

// update actualiza un rol existente
func (r *PostgresRoleRepository) update(ctx context.Context, roleEntity role.Role) error {
	query := `
		UPDATE roles SET
			name = :name,
			description = :description,
			is_active = :is_active,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id`

	result, err := r.db.NamedExecContext(ctx, query, roleEntity)
	if err != nil {
		// Verificar violación de constraint de nombre único
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" && pqErr.Constraint == "roles_name_tenant_id_key" {
				return role.ErrRoleAlreadyExists().
					WithDetail("name", roleEntity.Name).
					WithDetail("tenant_id", roleEntity.TenantID.String())
			}
		}
		return errx.Wrap(err, "failed to update role", errx.TypeInternal).
			WithDetail("role_id", roleEntity.ID.String())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return role.ErrRoleNotFound().WithDetail("role_id", roleEntity.ID.String())
	}

	return nil
}

// Delete elimina un rol
func (r *PostgresRoleRepository) Delete(ctx context.Context, id kernel.RoleID, tenantID kernel.TenantID) error {
	query := `DELETE FROM roles WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, id.String(), tenantID.String())
	if err != nil {
		return errx.Wrap(err, "failed to delete role", errx.TypeInternal).
			WithDetail("role_id", id.String()).
			WithDetail("tenant_id", tenantID.String())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return role.ErrRoleNotFound().WithDetail("role_id", id.String())
	}

	return nil
}

// ExistsByName verifica si existe un rol con el nombre dado en el tenant
func (r *PostgresRoleRepository) ExistsByName(ctx context.Context, name string, tenantID kernel.TenantID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM roles WHERE name = $1 AND tenant_id = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, name, tenantID.String())
	if err != nil {
		return false, errx.Wrap(err, "failed to check role existence by name", errx.TypeInternal).
			WithDetail("name", name).
			WithDetail("tenant_id", tenantID.String())
	}

	return exists, nil
}

// roleExists verifica si un rol existe por ID y tenant
func (r *PostgresRoleRepository) roleExists(ctx context.Context, id kernel.RoleID, tenantID kernel.TenantID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM roles WHERE id = $1 AND tenant_id = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id.String(), tenantID.String())
	if err != nil {
		return false, errx.Wrap(err, "failed to check role existence", errx.TypeInternal).
			WithDetail("role_id", id.String()).
			WithDetail("tenant_id", tenantID.String())
	}

	return exists, nil
}

// CountByTenant cuenta los roles de un tenant (método adicional útil)
func (r *PostgresRoleRepository) CountByTenant(ctx context.Context, tenantID kernel.TenantID) (int, error) {
	query := `SELECT COUNT(*) FROM roles WHERE tenant_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID.String())
	if err != nil {
		return 0, errx.Wrap(err, "failed to count roles by tenant", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String())
	}

	return count, nil
}

// =============================================================================
// RolePermissionRepository Implementation
// =============================================================================

// PostgresRolePermissionRepository implementación de PostgreSQL para RolePermissionRepository
type PostgresRolePermissionRepository struct {
	db *sqlx.DB
}

// NewPostgresRolePermissionRepository crea una nueva instancia del repositorio de permisos de roles
func NewPostgresRolePermissionRepository(db *sqlx.DB) role.RolePermissionRepository {
	return &PostgresRolePermissionRepository{
		db: db,
	}
}

// FindPermissionsByRole busca todos los permisos de un rol
func (r *PostgresRolePermissionRepository) FindPermissionsByRole(ctx context.Context, roleID kernel.RoleID) ([]string, error) {
	query := `
		SELECT permission 
		FROM role_permissions 
		WHERE role_id = $1
		ORDER BY permission ASC`

	var permissions []string
	err := r.db.SelectContext(ctx, &permissions, query, roleID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find permissions by role", errx.TypeInternal).
			WithDetail("role_id", roleID.String())
	}

	return permissions, nil
}

// AssignPermissionToRole asigna un permiso a un rol
func (r *PostgresRolePermissionRepository) AssignPermissionToRole(ctx context.Context, roleID kernel.RoleID, permission string) error {
	query := `
		INSERT INTO role_permissions (role_id, permission, assigned_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (role_id, permission) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, roleID.String(), permission)
	if err != nil {
		return errx.Wrap(err, "failed to assign permission to role", errx.TypeInternal).
			WithDetail("role_id", roleID.String()).
			WithDetail("permission", permission)
	}

	return nil
}

// RemovePermissionFromRole remueve un permiso de un rol
func (r *PostgresRolePermissionRepository) RemovePermissionFromRole(ctx context.Context, roleID kernel.RoleID, permission string) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission = $2`

	result, err := r.db.ExecContext(ctx, query, roleID.String(), permission)
	if err != nil {
		return errx.Wrap(err, "failed to remove permission from role", errx.TypeInternal).
			WithDetail("role_id", roleID.String()).
			WithDetail("permission", permission)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return role.ErrPermissionNotFound().
			WithDetail("role_id", roleID.String()).
			WithDetail("permission", permission)
	}

	return nil
}

// RemoveAllRolePermissions remueve todos los permisos de un rol
func (r *PostgresRolePermissionRepository) RemoveAllRolePermissions(ctx context.Context, roleID kernel.RoleID) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1`

	_, err := r.db.ExecContext(ctx, query, roleID.String())
	if err != nil {
		return errx.Wrap(err, "failed to remove all role permissions", errx.TypeInternal).
			WithDetail("role_id", roleID.String())
	}

	return nil
}

// HasPermission verifica si un rol tiene un permiso específico
func (r *PostgresRolePermissionRepository) HasPermission(ctx context.Context, roleID kernel.RoleID, permission string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM role_permissions WHERE role_id = $1 AND permission = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, roleID.String(), permission)
	if err != nil {
		return false, errx.Wrap(err, "failed to check role permission", errx.TypeInternal).
			WithDetail("role_id", roleID.String()).
			WithDetail("permission", permission)
	}

	return exists, nil
}

// CountPermissionsByRole cuenta los permisos de un rol (método adicional útil)
func (r *PostgresRolePermissionRepository) CountPermissionsByRole(ctx context.Context, roleID kernel.RoleID) (int, error) {
	query := `SELECT COUNT(*) FROM role_permissions WHERE role_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, roleID.String())
	if err != nil {
		return 0, errx.Wrap(err, "failed to count permissions by role", errx.TypeInternal).
			WithDetail("role_id", roleID.String())
	}

	return count, nil
}

// BatchAssignPermissions asigna múltiples permisos a un rol (método adicional útil)
func (r *PostgresRolePermissionRepository) BatchAssignPermissions(ctx context.Context, roleID kernel.RoleID, permissions []string) error {
	if len(permissions) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errx.Wrap(err, "failed to begin transaction", errx.TypeInternal)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO role_permissions (role_id, permission, assigned_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (role_id, permission) DO NOTHING`

	for _, permission := range permissions {
		_, err := tx.ExecContext(ctx, query, roleID.String(), permission)
		if err != nil {
			return errx.Wrap(err, "failed to assign permission in batch", errx.TypeInternal).
				WithDetail("role_id", roleID.String()).
				WithDetail("permission", permission)
		}
	}

	if err := tx.Commit(); err != nil {
		return errx.Wrap(err, "failed to commit permission batch transaction", errx.TypeInternal)
	}

	return nil
}
