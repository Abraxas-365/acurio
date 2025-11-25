package role

import (
	"net/http"
	"time"

	"github.com/Abraxas-365/relay/pkg/errx"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Role Entity
// ============================================================================

// Role representa un rol en el sistema
type Role struct {
	ID          kernel.RoleID   `db:"id" json:"id"`
	TenantID    kernel.TenantID `db:"tenant_id" json:"tenant_id"`
	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description,omitempty"`
	IsActive    bool            `db:"is_active" json:"is_active"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// Domain Methods
// ============================================================================

// IsValid verifica si el rol es válido
func (r *Role) IsValid() bool {
	return r.Name != "" && !r.TenantID.IsEmpty()
}

// Activate activa el rol
func (r *Role) Activate() {
	r.IsActive = true
	r.UpdatedAt = time.Now()
}

// Deactivate desactiva el rol
func (r *Role) Deactivate() {
	r.IsActive = false
	r.UpdatedAt = time.Now()
}

// UpdateDetails actualiza los detalles del rol
func (r *Role) UpdateDetails(name, description string) {
	if name != "" {
		r.Name = name
	}
	if description != "" {
		r.Description = description
	}
	r.UpdatedAt = time.Now()
}

// ============================================================================
// DTOs
// ============================================================================

// RoleDetailsDTO contiene información básica de un rol para otros módulos
type RoleDetailsDTO struct {
	ID          kernel.RoleID   `json:"id"`
	TenantID    kernel.TenantID `json:"tenant_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	IsActive    bool            `json:"is_active"`
}

// ToDTO convierte la entidad Role a RoleDetailsDTO
func (r *Role) ToDTO() RoleDetailsDTO {
	return RoleDetailsDTO{
		ID:          r.ID,
		TenantID:    r.TenantID,
		Name:        r.Name,
		Description: r.Description,
		IsActive:    r.IsActive,
	}
}

// ============================================================================
// Service DTOs - Para operaciones de la capa de servicio
// ============================================================================

// CreateRoleRequest representa la petición para crear un rol
type CreateRoleRequest struct {
	TenantID    kernel.TenantID `json:"tenant_id" validate:"required"`
	Name        string          `json:"name" validate:"required,min=2"`
	Description string          `json:"description"`
	Permissions []string        `json:"permissions,omitempty"`
}

// UpdateRoleRequest representa la petición para actualizar un rol
type UpdateRoleRequest struct {
	TenantID    kernel.TenantID `json:"tenant_id" validate:"required"`
	Name        *string         `json:"name,omitempty" validate:"omitempty,min=2"`
	Description *string         `json:"description,omitempty"`
	IsActive    *bool           `json:"is_active,omitempty"`
}

// RoleResponse representa la respuesta completa de un rol con sus permisos
type RoleResponse struct {
	Role        Role     `json:"role"`
	Permissions []string `json:"permissions"`
}

// ToDTO convierte RoleResponse a RoleResponseDTO
func (rr *RoleResponse) ToDTO() RoleResponseDTO {
	return RoleResponseDTO{
		ID:          rr.Role.ID,
		TenantID:    rr.Role.TenantID,
		Name:        rr.Role.Name,
		Description: rr.Role.Description,
		IsActive:    rr.Role.IsActive,
		Permissions: rr.Permissions,
		CreatedAt:   rr.Role.CreatedAt,
		UpdatedAt:   rr.Role.UpdatedAt,
	}
}

// RoleResponseDTO es la versión DTO de RoleResponse
type RoleResponseDTO struct {
	ID          kernel.RoleID   `json:"id"`
	TenantID    kernel.TenantID `json:"tenant_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	IsActive    bool            `json:"is_active"`
	Permissions []string        `json:"permissions"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// AssignPermissionRequest para asignar un permiso a un rol
type AssignPermissionRequest struct {
	TenantID   kernel.TenantID `json:"tenant_id" validate:"required"`
	Permission string          `json:"permission" validate:"required"`
}

// RemovePermissionRequest para remover un permiso de un rol
type RemovePermissionRequest struct {
	TenantID   kernel.TenantID `json:"tenant_id" validate:"required"`
	Permission string          `json:"permission" validate:"required"`
}

// SetPermissionsRequest para establecer todos los permisos de un rol
type SetPermissionsRequest struct {
	TenantID    kernel.TenantID `json:"tenant_id" validate:"required"`
	Permissions []string        `json:"permissions" validate:"required"`
}

// ActivateRoleRequest para activar un rol
type ActivateRoleRequest struct {
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
}

// DeactivateRoleRequest para desactivar un rol
type DeactivateRoleRequest struct {
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
}

// RoleListResponse para listas de roles
type RoleListResponse struct {
	Roles []RoleResponse `json:"roles"`
	Total int            `json:"total"`
}

// ToDTO convierte RoleListResponse a RoleListResponseDTO
func (rlr *RoleListResponse) ToDTO() RoleListResponseDTO {
	var rolesDTO []RoleResponseDTO
	for _, r := range rlr.Roles {
		rolesDTO = append(rolesDTO, r.ToDTO())
	}

	return RoleListResponseDTO{
		Roles: rolesDTO,
		Total: rlr.Total,
	}
}

// RoleListResponseDTO es la versión DTO de RoleListResponse
type RoleListResponseDTO struct {
	Roles []RoleResponseDTO `json:"roles"`
	Total int               `json:"total"`
}

// RolePermissionsResponse para respuesta de permisos de un rol
type RolePermissionsResponse struct {
	RoleID      kernel.RoleID `json:"role_id"`
	RoleName    string        `json:"role_name"`
	Permissions []string      `json:"permissions"`
}

// RoleUsersResponse para usuarios que tienen un rol específico
type RoleUsersResponse struct {
	RoleID    kernel.RoleID   `json:"role_id"`
	RoleName  string          `json:"role_name"`
	UserIDs   []kernel.UserID `json:"user_ids"`
	UserCount int             `json:"user_count"`
}

// RoleStatsResponse para estadísticas del rol
type RoleStatsResponse struct {
	RoleID           kernel.RoleID `json:"role_id"`
	RoleName         string        `json:"role_name"`
	IsActive         bool          `json:"is_active"`
	TotalPermissions int           `json:"total_permissions"`
	TotalUsers       int           `json:"total_users"`
	CreatedAt        time.Time     `json:"created_at"`
	LastUpdated      time.Time     `json:"last_updated"`
}

// BulkRoleOperationRequest para operaciones masivas en roles
type BulkRoleOperationRequest struct {
	TenantID  kernel.TenantID `json:"tenant_id" validate:"required"`
	RoleIDs   []kernel.RoleID `json:"role_ids" validate:"required,min=1"`
	Operation string          `json:"operation" validate:"required,oneof=activate deactivate delete"`
}

// BulkRoleOperationResponse resultado de operaciones masivas
type BulkRoleOperationResponse struct {
	Successful []kernel.RoleID          `json:"successful"`
	Failed     map[kernel.RoleID]string `json:"failed"`
	Total      int                      `json:"total"`
}

// CheckPermissionRequest para verificar si un rol tiene un permiso
type CheckPermissionRequest struct {
	TenantID   kernel.TenantID `json:"tenant_id" validate:"required"`
	Permission string          `json:"permission" validate:"required"`
}

// CheckPermissionResponse respuesta de verificación de permiso
type CheckPermissionResponse struct {
	RoleID        kernel.RoleID `json:"role_id"`
	Permission    string        `json:"permission"`
	HasPermission bool          `json:"has_permission"`
}

// RolePermissionAuditResponse para auditoría de permisos
type RolePermissionAuditResponse struct {
	RoleID      kernel.RoleID `json:"role_id"`
	RoleName    string        `json:"role_name"`
	Permission  string        `json:"permission"`
	Action      string        `json:"action"` // ASSIGNED, REMOVED
	PerformedBy kernel.UserID `json:"performed_by"`
	PerformedAt time.Time     `json:"performed_at"`
}

// AvailablePermissionsResponse para listar permisos disponibles
type AvailablePermissionsResponse struct {
	Permissions []PermissionInfo `json:"permissions"`
	Categories  []string         `json:"categories"`
}

// PermissionInfo información detallada de un permiso
type PermissionInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	IsSystem    bool   `json:"is_system"` // Si es un permiso del sistema que no se puede eliminar
}

// RoleTemplateRequest para crear roles desde plantillas
type RoleTemplateRequest struct {
	TenantID     kernel.TenantID `json:"tenant_id" validate:"required"`
	TemplateName string          `json:"template_name" validate:"required"`
	RoleName     string          `json:"role_name" validate:"required,min=2"`
	Description  string          `json:"description"`
}

// RoleTemplateResponse plantillas de roles disponibles
type RoleTemplateResponse struct {
	Templates []RoleTemplate `json:"templates"`
}

// RoleTemplate plantilla de rol predefinida
type RoleTemplate struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Category    string   `json:"category"`
}

// CopyRoleRequest para copiar un rol existente
type CopyRoleRequest struct {
	TenantID           kernel.TenantID `json:"tenant_id" validate:"required"`
	SourceRoleID       kernel.RoleID   `json:"source_role_id" validate:"required"`
	NewRoleName        string          `json:"new_role_name" validate:"required,min=2"`
	NewRoleDescription string          `json:"new_role_description"`
	CopyPermissions    bool            `json:"copy_permissions"`
}

// ============================================================================
// Error Registry - Errores específicos de Role
// ============================================================================

var ErrRegistry = errx.NewRegistry("ROLE")

// Códigos de error
var (
	CodeRoleNotFound         = ErrRegistry.Register("NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Rol no encontrado")
	CodeRoleAlreadyExists    = ErrRegistry.Register("ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "El rol ya existe")
	CodeRoleInUse            = ErrRegistry.Register("IN_USE", errx.TypeBusiness, http.StatusConflict, "El rol está siendo usado y no puede ser eliminado")
	CodeInvalidRoleName      = ErrRegistry.Register("INVALID_NAME", errx.TypeValidation, http.StatusBadRequest, "Nombre de rol inválido")
	CodeInvalidPermission    = ErrRegistry.Register("INVALID_PERMISSION", errx.TypeValidation, http.StatusBadRequest, "Permiso inválido")
	CodePermissionNotFound   = ErrRegistry.Register("PERMISSION_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Permiso no encontrado")
	CodeSystemRoleProtected  = ErrRegistry.Register("SYSTEM_ROLE_PROTECTED", errx.TypeBusiness, http.StatusForbidden, "No se puede modificar un rol del sistema")
	CodeRoleTemplateNotFound = ErrRegistry.Register("TEMPLATE_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Plantilla de rol no encontrada")
)

// Helper functions para crear errores
func ErrRoleNotFound() *errx.Error {
	return ErrRegistry.New(CodeRoleNotFound)
}

func ErrRoleAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeRoleAlreadyExists)
}

func ErrRoleInUse() *errx.Error {
	return ErrRegistry.New(CodeRoleInUse)
}

func ErrInvalidRoleName() *errx.Error {
	return ErrRegistry.New(CodeInvalidRoleName)
}

func ErrInvalidPermission() *errx.Error {
	return ErrRegistry.New(CodeInvalidPermission)
}

func ErrPermissionNotFound() *errx.Error {
	return ErrRegistry.New(CodePermissionNotFound)
}

func ErrSystemRoleProtected() *errx.Error {
	return ErrRegistry.New(CodeSystemRoleProtected)
}

func ErrRoleTemplateNotFound() *errx.Error {
	return ErrRegistry.New(CodeRoleTemplateNotFound)
}
