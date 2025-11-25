package role

import (
	"context"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// RoleRepository define el contrato para la persistencia de roles
type RoleRepository interface {
	FindByID(ctx context.Context, id kernel.RoleID, tenantID kernel.TenantID) (*Role, error)
	FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*Role, error)
	FindByName(ctx context.Context, name string, tenantID kernel.TenantID) (*Role, error)
	FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*Role, error)
	Save(ctx context.Context, r Role) error
	Delete(ctx context.Context, id kernel.RoleID, tenantID kernel.TenantID) error
	ExistsByName(ctx context.Context, name string, tenantID kernel.TenantID) (bool, error)
}

// RolePermissionRepository define el contrato para la relación rol-permiso
type RolePermissionRepository interface {
	FindPermissionsByRole(ctx context.Context, roleID kernel.RoleID) ([]string, error)
	AssignPermissionToRole(ctx context.Context, roleID kernel.RoleID, permission string) error
	RemovePermissionFromRole(ctx context.Context, roleID kernel.RoleID, permission string) error
	RemoveAllRolePermissions(ctx context.Context, roleID kernel.RoleID) error
	HasPermission(ctx context.Context, roleID kernel.RoleID, permission string) (bool, error)
}
