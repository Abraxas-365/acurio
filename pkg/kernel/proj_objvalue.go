package kernel

type JobTitle string

type JobDescription string

type JobPosition string

type ResumeEmbedding []float32

type ResumeSummary string

type JobRequirement string

type JobBenefit string

// DNIType tipos de documentos de identidad en Perú
type DNIType string

const (
	// DNITypeNacional - Documento Nacional de Identidad (peruanos)
	DNITypeNacional DNIType = "DNI"

	// DNITypeCarnetExtranjeria - Carnet de Extranjería (residentes extranjeros)
	DNITypeCarnetExtranjeria DNIType = "CE"

	// DNITypePasaporte - Pasaporte (extranjeros sin carnet)
	DNITypePasaporte DNIType = "PASAPORTE"

	// DNITypePTP - Permiso Temporal de Permanencia (principalmente venezolanos)
	DNITypePTP DNIType = "PTP"

	// DNITypeCPP - Carné de Permiso Temporal de Permanencia
	DNITypeCPP DNIType = "CPP"

	// DNITypeRUC - RUC (para empresas/independientes, raramente usado para personas)
	DNITypeRUC DNIType = "RUC"
)

// DNI representa un documento de identidad
type DNI struct {
	Type   DNIType `json:"type"`
	Number string  `json:"number"`
}

// IsValid valida el formato del documento según su tipo
func (d DNI) IsValid() bool {
	switch d.Type {
	case DNITypeNacional:
		// DNI peruano: 8 dígitos
		return len(d.Number) == 8 && isNumeric(d.Number)

	case DNITypeCarnetExtranjeria:
		// Carnet de Extranjería: formato variable, 9-12 caracteres alfanuméricos
		return len(d.Number) >= 9 && len(d.Number) <= 12

	case DNITypePasaporte:
		// Pasaporte: formato variable según país, 6-12 caracteres alfanuméricos
		return len(d.Number) >= 6 && len(d.Number) <= 12

	case DNITypePTP:
		// PTP: 9 dígitos
		return len(d.Number) == 9 && isNumeric(d.Number)

	case DNITypeCPP:
		// CPP: 12 caracteres alfanuméricos
		return len(d.Number) == 12

	case DNITypeRUC:
		// RUC: 11 dígitos
		return len(d.Number) == 11 && isNumeric(d.Number)

	default:
		return false
	}
}

// GetDisplayName retorna el nombre legible del tipo de documento
func (t DNIType) GetDisplayName() string {
	switch t {
	case DNITypeNacional:
		return "DNI"
	case DNITypeCarnetExtranjeria:
		return "Carnet de Extranjería"
	case DNITypePasaporte:
		return "Pasaporte"
	case DNITypePTP:
		return "Permiso Temporal de Permanencia"
	case DNITypeCPP:
		return "Carné de Permiso Temporal"
	case DNITypeRUC:
		return "RUC"
	default:
		return "Desconocido"
	}
}

// IsPeruvianDocument verifica si es un documento peruano
func (t DNIType) IsPeruvianDocument() bool {
	return t == DNITypeNacional || t == DNITypeRUC
}

// RequiresWorkPermit verifica si el documento requiere permiso de trabajo
func (t DNIType) RequiresWorkPermit() bool {
	return t == DNITypePasaporte || t == DNITypePTP
}

// Helper function
func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

type BucketURL string
