package rolesrv

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

// RoleService proporciona operaciones de negocio para roles
type RoleService struct {
	roleRepo           role.RoleRepository
	rolePermissionRepo role.RolePermissionRepository
	userRoleRepo       user.UserRoleRepository
	tenantRepo         tenant.TenantRepository
}

// NewRoleService crea una nueva instancia del servicio de roles
func NewRoleService(
	roleRepo role.RoleRepository,
	rolePermissionRepo role.RolePermissionRepository,
	tenantRepo tenant.TenantRepository,
) *RoleService {
	return &RoleService{
		roleRepo:           roleRepo,
		rolePermissionRepo: rolePermissionRepo,
		tenantRepo:         tenantRepo,
	}
}

// CreateRole crea un nuevo rol
func (s *RoleService) CreateRole(ctx context.Context, req role.CreateRoleRequest) (*role.Role, error) {
	// Verificar que el tenant exista y esté activo
	tenantEntity, err := s.tenantRepo.FindByID(ctx, req.TenantID)
	if err != nil {
		return nil, tenant.ErrTenantNotFound()
	}

	if !tenantEntity.IsActive() {
		return nil, tenant.ErrTenantSuspended()
	}

	// Verificar que no exista un rol con el mismo nombre en el tenant
	exists, err := s.roleRepo.ExistsByName(ctx, req.Name, req.TenantID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to check role name existence", errx.TypeInternal)
	}
	if exists {
		return nil, role.ErrRoleAlreadyExists()
	}

	// Crear nuevo rol
	newRole := &role.Role{
		ID:          kernel.NewRoleID(uuid.NewString()),
		TenantID:    req.TenantID,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Guardar rol
	if err := s.roleRepo.Save(ctx, *newRole); err != nil {
		return nil, errx.Wrap(err, "failed to save role", errx.TypeInternal)
	}

	// Asignar permisos si se especificaron
	if len(req.Permissions) > 0 {
		if err := s.assignPermissionsToRole(ctx, newRole.ID, req.Permissions); err != nil {
			// Log error pero no fallar
			// logger.Error("Failed to assign permissions to role", err)
		}
	}

	return newRole, nil
}

// GetRoleByID obtiene un rol por ID
func (s *RoleService) GetRoleByID(ctx context.Context, roleID kernel.RoleID, tenantID kernel.TenantID) (*role.RoleResponse, error) {
	roleEntity, err := s.roleRepo.FindByID(ctx, roleID, tenantID)
	if err != nil {
		return nil, role.ErrRoleNotFound()
	}

	// Obtener permisos del rol
	permissions, err := s.rolePermissionRepo.FindPermissionsByRole(ctx, roleID)
	if err != nil {
		permissions = []string{} // Default to empty slice
	}

	return &role.RoleResponse{
		Role:        *roleEntity,
		Permissions: permissions,
	}, nil
}

// GetRoleByName obtiene un rol por nombre
func (s *RoleService) GetRoleByName(ctx context.Context, name string, tenantID kernel.TenantID) (*role.RoleResponse, error) {
	roleEntity, err := s.roleRepo.FindByName(ctx, name, tenantID)
	if err != nil {
		return nil, role.ErrRoleNotFound()
	}

	// Obtener permisos del rol
	permissions, err := s.rolePermissionRepo.FindPermissionsByRole(ctx, roleEntity.ID)
	if err != nil {
		permissions = []string{}
	}

	return &role.RoleResponse{
		Role:        *roleEntity,
		Permissions: permissions,
	}, nil
}

// GetRolesByTenant obtiene todos los roles de un tenant
func (s *RoleService) GetRolesByTenant(ctx context.Context, tenantID kernel.TenantID) (*role.RoleListResponse, error) {
	roles, err := s.roleRepo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to get roles by tenant", errx.TypeInternal)
	}

	var responses []role.RoleResponse
	for _, r := range roles {
		permissions, _ := s.rolePermissionRepo.FindPermissionsByRole(ctx, r.ID)
		responses = append(responses, role.RoleResponse{
			Role:        *r,
			Permissions: permissions,
		})
	}

	return &role.RoleListResponse{
		Roles: responses,
		Total: len(responses),
	}, nil
}

// GetActiveRoles obtiene todos los roles activos de un tenant
func (s *RoleService) GetActiveRoles(ctx context.Context, tenantID kernel.TenantID) (*role.RoleListResponse, error) {
	roles, err := s.roleRepo.FindActive(ctx, tenantID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to get active roles", errx.TypeInternal)
	}

	var responses []role.RoleResponse
	for _, r := range roles {
		permissions, _ := s.rolePermissionRepo.FindPermissionsByRole(ctx, r.ID)
		responses = append(responses, role.RoleResponse{
			Role:        *r,
			Permissions: permissions,
		})
	}

	return &role.RoleListResponse{
		Roles: responses,
		Total: len(responses),
	}, nil
}

// UpdateRole actualiza un rol
func (s *RoleService) UpdateRole(ctx context.Context, roleID kernel.RoleID, req role.UpdateRoleRequest) (*role.Role, error) {
	roleEntity, err := s.roleRepo.FindByID(ctx, roleID, req.TenantID)
	if err != nil {
		return nil, role.ErrRoleNotFound()
	}

	// Actualizar campos si se proporcionaron
	if req.Name != nil {
		// Verificar que no exista otro rol con el mismo nombre
		if *req.Name != roleEntity.Name {
			exists, err := s.roleRepo.ExistsByName(ctx, *req.Name, req.TenantID)
			if err != nil {
				return nil, errx.Wrap(err, "failed to check role name", errx.TypeInternal)
			}
			if exists {
				return nil, role.ErrRoleAlreadyExists()
			}
			roleEntity.Name = *req.Name
		}
	}

	if req.Description != nil {
		roleEntity.Description = *req.Description
	}

	if req.IsActive != nil {
		if *req.IsActive {
			roleEntity.Activate()
		} else {
			roleEntity.Deactivate()
		}
	}

	roleEntity.UpdatedAt = time.Now()

	// Guardar cambios
	if err := s.roleRepo.Save(ctx, *roleEntity); err != nil {
		return nil, errx.Wrap(err, "failed to update role", errx.TypeInternal)
	}

	return roleEntity, nil
}

// ActivateRole activa un rol
func (s *RoleService) ActivateRole(ctx context.Context, roleID kernel.RoleID, tenantID kernel.TenantID) error {
	roleEntity, err := s.roleRepo.FindByID(ctx, roleID, tenantID)
	if err != nil {
		return role.ErrRoleNotFound()
	}

	roleEntity.Activate()
	return s.roleRepo.Save(ctx, *roleEntity)
}

// DeactivateRole desactiva un rol
func (s *RoleService) DeactivateRole(ctx context.Context, roleID kernel.RoleID, tenantID kernel.TenantID) error {
	roleEntity, err := s.roleRepo.FindByID(ctx, roleID, tenantID)
	if err != nil {
		return role.ErrRoleNotFound()
	}

	roleEntity.Deactivate()
	return s.roleRepo.Save(ctx, *roleEntity)
}

// AssignPermissionToRole asigna un permiso a un rol
func (s *RoleService) AssignPermissionToRole(ctx context.Context, roleID kernel.RoleID, permission string, tenantID kernel.TenantID) error {
	// Verificar que el rol existe
	_, err := s.roleRepo.FindByID(ctx, roleID, tenantID)
	if err != nil {
		return role.ErrRoleNotFound()
	}

	return s.rolePermissionRepo.AssignPermissionToRole(ctx, roleID, permission)
}

// RemovePermissionFromRole remueve un permiso de un rol
func (s *RoleService) RemovePermissionFromRole(ctx context.Context, roleID kernel.RoleID, permission string, tenantID kernel.TenantID) error {
	// Verificar que el rol existe
	_, err := s.roleRepo.FindByID(ctx, roleID, tenantID)
	if err != nil {
		return role.ErrRoleNotFound()
	}

	return s.rolePermissionRepo.RemovePermissionFromRole(ctx, roleID, permission)
}

// GetRolePermissions obtiene todos los permisos de un rol
func (s *RoleService) GetRolePermissions(ctx context.Context, roleID kernel.RoleID, tenantID kernel.TenantID) (*role.RolePermissionsResponse, error) {
	// Verificar que el rol existe
	roleEntity, err := s.roleRepo.FindByID(ctx, roleID, tenantID)
	if err != nil {
		return nil, role.ErrRoleNotFound()
	}

	permissions, err := s.rolePermissionRepo.FindPermissionsByRole(ctx, roleID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to get role permissions", errx.TypeInternal)
	}

	return &role.RolePermissionsResponse{
		RoleID:      roleID,
		RoleName:    roleEntity.Name,
		Permissions: permissions,
	}, nil
}

// SetRolePermissions establece todos los permisos de un rol (reemplaza los existentes)
func (s *RoleService) SetRolePermissions(ctx context.Context, roleID kernel.RoleID, permissions []string, tenantID kernel.TenantID) error {
	// Verificar que el rol existe
	_, err := s.roleRepo.FindByID(ctx, roleID, tenantID)
	if err != nil {
		return role.ErrRoleNotFound()
	}

	// Remover todos los permisos existentes
	if err := s.rolePermissionRepo.RemoveAllRolePermissions(ctx, roleID); err != nil {
		return errx.Wrap(err, "failed to remove existing permissions", errx.TypeInternal)
	}

	// Asignar los nuevos permisos
	return s.assignPermissionsToRole(ctx, roleID, permissions)
}

// CheckRolePermission verifica si un rol tiene un permiso específico
func (s *RoleService) CheckRolePermission(ctx context.Context, roleID kernel.RoleID, permission string, tenantID kernel.TenantID) (*role.CheckPermissionResponse, error) {
	// Verificar que el rol existe
	_, err := s.roleRepo.FindByID(ctx, roleID, tenantID)
	if err != nil {
		return nil, role.ErrRoleNotFound()
	}

	hasPermission, err := s.rolePermissionRepo.HasPermission(ctx, roleID, permission)
	if err != nil {
		return nil, errx.Wrap(err, "failed to check permission", errx.TypeInternal)
	}

	return &role.CheckPermissionResponse{
		RoleID:        roleID,
		Permission:    permission,
		HasPermission: hasPermission,
	}, nil
}

// CopyRole crea una copia de un rol existente
func (s *RoleService) CopyRole(ctx context.Context, req role.CopyRoleRequest) (*role.Role, error) {
	// Verificar que el rol fuente existe
	sourceRole, err := s.roleRepo.FindByID(ctx, req.SourceRoleID, req.TenantID)
	if err != nil {
		return nil, role.ErrRoleNotFound()
	}

	// Verificar que no exista un rol con el nuevo nombre
	exists, err := s.roleRepo.ExistsByName(ctx, req.NewRoleName, req.TenantID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to check new role name", errx.TypeInternal)
	}
	if exists {
		return nil, role.ErrRoleAlreadyExists()
	}

	// Crear nuevo rol
	newRole := &role.Role{
		ID:          kernel.NewRoleID(uuid.NewString()),
		TenantID:    req.TenantID,
		Name:        req.NewRoleName,
		Description: req.NewRoleDescription,
		IsActive:    sourceRole.IsActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Si no se especificó descripción, usar la del rol fuente
	if newRole.Description == "" {
		newRole.Description = "Copia de " + sourceRole.Name
	}

	// Guardar nuevo rol
	if err := s.roleRepo.Save(ctx, *newRole); err != nil {
		return nil, errx.Wrap(err, "failed to save copied role", errx.TypeInternal)
	}

	// Copiar permisos si se especificó
	if req.CopyPermissions {
		permissions, err := s.rolePermissionRepo.FindPermissionsByRole(ctx, req.SourceRoleID)
		if err == nil && len(permissions) > 0 {
			s.assignPermissionsToRole(ctx, newRole.ID, permissions)
		}
	}

	return newRole, nil
}

// DeleteRole elimina un rol
func (s *RoleService) DeleteRole(ctx context.Context, roleID kernel.RoleID, tenantID kernel.TenantID) error {
	// Verificar que el rol existe
	_, err := s.roleRepo.FindByID(ctx, roleID, tenantID)
	if err != nil {
		return role.ErrRoleNotFound()
	}

	// Verificar que el rol no esté siendo usado
	userCount, err := s.userRoleRepo.CountUsersByRole(ctx, roleID)
	if err != nil {
		return errx.Wrap(err, "failed to check role usage", errx.TypeInternal)
	}
	if userCount > 0 {
		return role.ErrRoleInUse()
	}

	// Remover todos los permisos del rol
	if err := s.rolePermissionRepo.RemoveAllRolePermissions(ctx, roleID); err != nil {
		// Log error pero continuar
	}

	// Eliminar rol
	if err := s.roleRepo.Delete(ctx, roleID, tenantID); err != nil {
		return errx.Wrap(err, "failed to delete role", errx.TypeInternal)
	}

	return nil
}

// BulkActivateRoles activa múltiples roles
func (s *RoleService) BulkActivateRoles(ctx context.Context, roleIDs []kernel.RoleID, tenantID kernel.TenantID) (*role.BulkRoleOperationResponse, error) {
	result := &role.BulkRoleOperationResponse{
		Successful: []kernel.RoleID{},
		Failed:     make(map[kernel.RoleID]string),
		Total:      len(roleIDs),
	}

	for _, roleID := range roleIDs {
		if err := s.ActivateRole(ctx, roleID, tenantID); err != nil {
			result.Failed[roleID] = err.Error()
		} else {
			result.Successful = append(result.Successful, roleID)
		}
	}

	return result, nil
}

// BulkDeactivateRoles desactiva múltiples roles
func (s *RoleService) BulkDeactivateRoles(ctx context.Context, roleIDs []kernel.RoleID, tenantID kernel.TenantID) (*role.BulkRoleOperationResponse, error) {
	result := &role.BulkRoleOperationResponse{
		Successful: []kernel.RoleID{},
		Failed:     make(map[kernel.RoleID]string),
		Total:      len(roleIDs),
	}

	for _, roleID := range roleIDs {
		if err := s.DeactivateRole(ctx, roleID, tenantID); err != nil {
			result.Failed[roleID] = err.Error()
		} else {
			result.Successful = append(result.Successful, roleID)
		}
	}

	return result, nil
}

// GetAvailablePermissions obtiene todos los permisos disponibles del sistema
func (s *RoleService) GetAvailablePermissions(ctx context.Context) (*role.AvailablePermissionsResponse, error) {
	// Esta implementación debería obtener los permisos desde una configuración
	// o base de datos que defina todos los permisos disponibles del sistema
	permissions := s.getSystemPermissions()
	categories := s.getPermissionCategories()

	return &role.AvailablePermissionsResponse{
		Permissions: permissions,
		Categories:  categories,
	}, nil
}

// Helper function to assign multiple permissions to role
func (s *RoleService) assignPermissionsToRole(ctx context.Context, roleID kernel.RoleID, permissions []string) error {
	for _, permission := range permissions {
		if err := s.rolePermissionRepo.AssignPermissionToRole(ctx, roleID, permission); err != nil {
			return err
		}
	}
	return nil
}

// Helper functions para permisos del sistema (esto debería venir de configuración)
func (s *RoleService) getSystemPermissions() []role.PermissionInfo {
	return []role.PermissionInfo{
		{Name: "users.create", Description: "Crear usuarios", Category: "Usuarios", IsSystem: true},
		{Name: "users.read", Description: "Ver usuarios", Category: "Usuarios", IsSystem: true},
		{Name: "users.update", Description: "Actualizar usuarios", Category: "Usuarios", IsSystem: true},
		{Name: "users.delete", Description: "Eliminar usuarios", Category: "Usuarios", IsSystem: true},
		{Name: "roles.create", Description: "Crear roles", Category: "Roles", IsSystem: true},
		{Name: "roles.read", Description: "Ver roles", Category: "Roles", IsSystem: true},
		{Name: "roles.update", Description: "Actualizar roles", Category: "Roles", IsSystem: true},
		{Name: "roles.delete", Description: "Eliminar roles", Category: "Roles", IsSystem: true},
		{Name: "invoices.create", Description: "Crear facturas", Category: "Facturas", IsSystem: true},
		{Name: "invoices.read", Description: "Ver facturas", Category: "Facturas", IsSystem: true},
		{Name: "invoices.approve", Description: "Aprobar facturas", Category: "Facturas", IsSystem: true},
		{Name: "reports.view", Description: "Ver reportes", Category: "Reportes", IsSystem: true},
		{Name: "admin.full", Description: "Acceso completo de administrador", Category: "Administración", IsSystem: true},
	}
}

func (s *RoleService) getPermissionCategories() []string {
	return []string{
		"Usuarios",
		"Roles",
		"Facturas",
		"Reportes",
		"Administración",
	}
}
