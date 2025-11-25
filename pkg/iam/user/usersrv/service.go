package usersrv

import (
	"context"
	"time"

	"github.com/Abraxas-365/relay/pkg/errx"
	"github.com/Abraxas-365/relay/pkg/iam/role"
	"github.com/Abraxas-365/relay/pkg/iam/tenant"
	"github.com/Abraxas-365/relay/pkg/iam/user"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/google/uuid"
)

// UserService proporciona operaciones de negocio para usuarios
type UserService struct {
	userRepo     user.UserRepository
	userRoleRepo user.UserRoleRepository
	tenantRepo   tenant.TenantRepository
	roleRepo     role.RoleRepository
	passwordSvc  user.PasswordService
}

// NewUserService crea una nueva instancia del servicio de usuarios
func NewUserService(
	userRepo user.UserRepository,
	userRoleRepo user.UserRoleRepository,
	tenantRepo tenant.TenantRepository,
	roleRepo role.RoleRepository,
	passwordSvc user.PasswordService,
) *UserService {
	return &UserService{
		userRepo:     userRepo,
		userRoleRepo: userRoleRepo,
		tenantRepo:   tenantRepo,
		roleRepo:     roleRepo,
		passwordSvc:  passwordSvc,
	}
}

// CreateUser crea un nuevo usuario
func (s *UserService) CreateUser(ctx context.Context, req user.CreateUserRequest, creatorID kernel.UserID) (*user.User, error) {
	// Validar que el tenant exista y esté activo
	tenantEntity, err := s.tenantRepo.FindByID(ctx, req.TenantID)
	if err != nil {
		return nil, tenant.ErrTenantNotFound()
	}

	if !tenantEntity.IsActive() {
		return nil, tenant.ErrTenantSuspended()
	}

	// Verificar que el tenant puede agregar más usuarios
	if !tenantEntity.CanAddUser() {
		return nil, tenant.ErrMaxUsersReached()
	}

	// Verificar que no exista un usuario con el mismo email
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email, req.TenantID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to check email existence", errx.TypeInternal)
	}
	if exists {
		return nil, user.ErrUserAlreadyExists()
	}

	// Determinar scopes por defecto si no se proporcionaron
	scopes := req.Scopes
	if len(scopes) == 0 {
		// Default scopes for new users
		scopes = []string{
			"channels:read",
			"channels:write",
			"messages:read",
			"messages:send",
			"tools:read",
			"workflows:read",
		}
	}

	// Crear nuevo usuario
	newUser := &user.User{
		ID:            kernel.NewUserID(uuid.NewString()),
		TenantID:      req.TenantID,
		Email:         req.Email,
		Name:          req.Name,
		Status:        user.UserStatusPending, // Pendiente hasta completar onboarding
		Scopes:        scopes,
		EmailVerified: false, // Se verificará después
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Guardar usuario
	if err := s.userRepo.Save(ctx, *newUser); err != nil {
		return nil, errx.Wrap(err, "failed to save user", errx.TypeInternal)
	}

	// Asignar roles si se especificaron
	if len(req.RoleIDs) > 0 {
		if err := s.assignRolesToUser(ctx, newUser.ID, req.RoleIDs); err != nil {
			// Log error pero no fallar
			// logger.Error("Failed to assign roles to user", err)
		}
	}

	// Incrementar contador de usuarios del tenant
	if err := tenantEntity.AddUser(); err == nil {
		s.tenantRepo.Save(ctx, *tenantEntity)
	}

	return newUser, nil
}

// GetUserByID obtiene un usuario por ID
func (s *UserService) GetUserByID(ctx context.Context, userID kernel.UserID, tenantID kernel.TenantID) (*user.UserResponse, error) {
	userEntity, err := s.userRepo.FindByID(ctx, userID, tenantID)
	if err != nil {
		return nil, user.ErrUserNotFound()
	}

	// Obtener roles del usuario
	roleIDs, err := s.userRoleRepo.FindRolesByUser(ctx, userID)
	if err != nil {
		roleIDs = []kernel.RoleID{} // Default a empty slice
	}

	return &user.UserResponse{
		User:    *userEntity,
		RoleIDs: roleIDs,
	}, nil
}

// GetUserByEmail obtiene un usuario por email
func (s *UserService) GetUserByEmail(ctx context.Context, email string, tenantID kernel.TenantID) (*user.UserResponse, error) {
	userEntity, err := s.userRepo.FindByEmail(ctx, email, tenantID)
	if err != nil {
		return nil, user.ErrUserNotFound()
	}

	// Obtener roles del usuario
	roleIDs, err := s.userRoleRepo.FindRolesByUser(ctx, userEntity.ID)
	if err != nil {
		roleIDs = []kernel.RoleID{}
	}

	return &user.UserResponse{
		User:    *userEntity,
		RoleIDs: roleIDs,
	}, nil
}

// GetUsersByTenant obtiene todos los usuarios de un tenant
func (s *UserService) GetUsersByTenant(ctx context.Context, tenantID kernel.TenantID) (*user.UserListResponse, error) {
	users, err := s.userRepo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to get users by tenant", errx.TypeInternal)
	}

	var userResponses []user.UserResponse
	for _, u := range users {
		roleIDs, _ := s.userRoleRepo.FindRolesByUser(ctx, u.ID)
		userResponses = append(userResponses, user.UserResponse{
			User:    *u,
			RoleIDs: roleIDs,
		})
	}

	return &user.UserListResponse{
		Users: userResponses,
		Total: len(userResponses),
	}, nil
}

// UpdateUser actualiza un usuario
func (s *UserService) UpdateUser(ctx context.Context, userID kernel.UserID, req user.UpdateUserRequest, updaterID kernel.UserID) (*user.User, error) {
	userEntity, err := s.userRepo.FindByID(ctx, userID, req.TenantID)
	if err != nil {
		return nil, user.ErrUserNotFound()
	}

	// Actualizar campos si se proporcionaron
	if req.Name != nil {
		userEntity.Name = *req.Name
	}
	if req.Status != nil {
		switch *req.Status {
		case user.UserStatusActive:
			if err := userEntity.Activate(); err != nil {
				return nil, err
			}
		case user.UserStatusSuspended:
			if err := userEntity.Suspend("Updated by admin"); err != nil {
				return nil, err
			}
		}
	}
	if req.Scopes != nil && len(req.Scopes) > 0 {
		userEntity.SetScopes(req.Scopes)
	}

	userEntity.UpdatedAt = time.Now()

	// Guardar cambios
	if err := s.userRepo.Save(ctx, *userEntity); err != nil {
		return nil, errx.Wrap(err, "failed to update user", errx.TypeInternal)
	}

	return userEntity, nil
}

// ActivateUser activa un usuario pendiente
func (s *UserService) ActivateUser(ctx context.Context, userID kernel.UserID, tenantID kernel.TenantID) error {
	userEntity, err := s.userRepo.FindByID(ctx, userID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	if err := userEntity.Activate(); err != nil {
		return err
	}

	return s.userRepo.Save(ctx, *userEntity)
}

// SuspendUser suspende un usuario
func (s *UserService) SuspendUser(ctx context.Context, userID kernel.UserID, tenantID kernel.TenantID, reason string) error {
	userEntity, err := s.userRepo.FindByID(ctx, userID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	if err := userEntity.Suspend(reason); err != nil {
		return err
	}

	return s.userRepo.Save(ctx, *userEntity)
}

// AssignUserToRole asigna un usuario a un rol
func (s *UserService) AssignUserToRole(ctx context.Context, userID kernel.UserID, roleID kernel.RoleID, tenantID kernel.TenantID) error {
	// Verificar que el usuario existe
	_, err := s.userRepo.FindByID(ctx, userID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	// Verificar que el rol existe y pertenece al mismo tenant
	_, err = s.roleRepo.FindByID(ctx, roleID, tenantID)
	if err != nil {
		return role.ErrRoleNotFound()
	}

	return s.userRoleRepo.AssignUserToRole(ctx, userID, roleID)
}

// RemoveUserFromRole remueve un usuario de un rol
func (s *UserService) RemoveUserFromRole(ctx context.Context, userID kernel.UserID, roleID kernel.RoleID, tenantID kernel.TenantID) error {
	// Verificar que el usuario existe
	_, err := s.userRepo.FindByID(ctx, userID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	return s.userRoleRepo.RemoveUserFromRole(ctx, userID, roleID)
}

// GetUserRoles obtiene los roles de un usuario
func (s *UserService) GetUserRoles(ctx context.Context, userID kernel.UserID, tenantID kernel.TenantID) ([]*role.Role, error) {
	// Verificar que el usuario existe
	_, err := s.userRepo.FindByID(ctx, userID, tenantID)
	if err != nil {
		return nil, user.ErrUserNotFound()
	}

	roleIDs, err := s.userRoleRepo.FindRolesByUser(ctx, userID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to get user roles", errx.TypeInternal)
	}

	var roles []*role.Role
	for _, roleID := range roleIDs {
		roleEntity, err := s.roleRepo.FindByID(ctx, roleID, tenantID)
		if err != nil {
			continue // Skip invalid roles
		}
		roles = append(roles, roleEntity)
	}

	return roles, nil
}

// DeleteUser elimina un usuario
func (s *UserService) DeleteUser(ctx context.Context, userID kernel.UserID, tenantID kernel.TenantID) error {
	// Verificar que el usuario existe
	_, err := s.userRepo.FindByID(ctx, userID, tenantID)
	if err != nil {
		return user.ErrUserNotFound()
	}

	// Remover todos los roles del usuario
	if err := s.userRoleRepo.RemoveAllUserRoles(ctx, userID); err != nil {
		// Log error pero continúar
	}

	// Eliminar usuario
	if err := s.userRepo.Delete(ctx, userID, tenantID); err != nil {
		return errx.Wrap(err, "failed to delete user", errx.TypeInternal)
	}

	// Decrementar contador de usuarios del tenant
	tenantEntity, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err == nil {
		tenantEntity.RemoveUser()
		s.tenantRepo.Save(ctx, *tenantEntity)
	}

	return nil
}

// Helper function to assign multiple roles to user
func (s *UserService) assignRolesToUser(ctx context.Context, userID kernel.UserID, roleIDs []kernel.RoleID) error {
	for _, roleID := range roleIDs {
		if err := s.userRoleRepo.AssignUserToRole(ctx, userID, roleID); err != nil {
			return err
		}
	}
	return nil
}
