package user

import (
	"context"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// UserRepository define el contrato para la persistencia de usuarios
type UserRepository interface {
	FindByID(ctx context.Context, id kernel.UserID, tenantID kernel.TenantID) (*User, error)
	FindByEmail(ctx context.Context, email string, tenantID kernel.TenantID) (*User, error)
	FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*User, error)
	Save(ctx context.Context, u User) error
	Delete(ctx context.Context, id kernel.UserID, tenantID kernel.TenantID) error
	ExistsByEmail(ctx context.Context, email string, tenantID kernel.TenantID) (bool, error)
}

// UserRoleRepository define el contrato para la relación usuario-rol
type UserRoleRepository interface {
	FindRolesByUser(ctx context.Context, userID kernel.UserID) ([]kernel.RoleID, error)
	AssignUserToRole(ctx context.Context, userID kernel.UserID, roleID kernel.RoleID) error
	RemoveUserFromRole(ctx context.Context, userID kernel.UserID, roleID kernel.RoleID) error
	RemoveAllUserRoles(ctx context.Context, userID kernel.UserID) error
	FindUsersByRole(ctx context.Context, roleID kernel.RoleID) ([]kernel.UserID, error)
	CountUsersByRole(ctx context.Context, roleID kernel.RoleID) (int, error)
}

// PasswordService define el contrato para el manejo de contraseñas
type PasswordService interface {
	HashPassword(password string) (string, error)
	VerifyPassword(hashedPassword, password string) bool
}
