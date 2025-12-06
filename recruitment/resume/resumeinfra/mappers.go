package resumeinfra

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/resume"
)

// resumeRow represents a row from the resumes table
type resumeRow struct {
	ID                  string         `db:"id"`
	TenantID            string         `db:"tenant_id"`
	Title               string         `db:"title"`
	IsActive            bool           `db:"is_active"`
	IsDefault           bool           `db:"is_default"`
	Version             int            `db:"version"`
	PersonalInfo        []byte         `db:"personal_info"`
	WorkExperience      []byte         `db:"work_experience"`
	Education           []byte         `db:"education"`
	Skills              []byte         `db:"skills"`
	Languages           []byte         `db:"languages"`
	Certifications      []byte         `db:"certifications"`
	Projects            []byte         `db:"projects"`
	Achievements        []byte         `db:"achievements"`
	VolunteerWork       []byte         `db:"volunteer_work"`
	ProfessionalSummary sql.NullString `db:"professional_summary"`
	PersonalStatement   []byte         `db:"personal_statement"`
	FileURL             string         `db:"file_url"`
	FileName            string         `db:"file_name"`
	FileType            string         `db:"file_type"`
	ParsedAt            time.Time      `db:"parsed_at"`
	LastUpdatedAt       time.Time      `db:"last_updated_at"`
	CreatedAt           time.Time      `db:"created_at"`
}

// ToDomain converts a resumeRow to a resume.Resume domain model
func (r *resumeRow) ToDomain() (*resume.Resume, error) {
	resumeModel := &resume.Resume{
		ID:            kernel.ResumeID(r.ID),
		TenantID:      kernel.TenantID(r.TenantID),
		Title:         r.Title,
		IsActive:      r.IsActive,
		IsDefault:     r.IsDefault,
		Version:       r.Version,
		FileURL:       r.FileURL,
		FileName:      r.FileName,
		FileType:      r.FileType,
		ParsedAt:      r.ParsedAt,
		LastUpdatedAt: r.LastUpdatedAt,
		CreatedAt:     r.CreatedAt,
	}

	// Unmarshal JSONB fields
	if err := json.Unmarshal(r.PersonalInfo, &resumeModel.PersonalInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal personal_info: %w", err)
	}

	if err := json.Unmarshal(r.WorkExperience, &resumeModel.WorkExperience); err != nil {
		return nil, fmt.Errorf("failed to unmarshal work_experience: %w", err)
	}

	if err := json.Unmarshal(r.Education, &resumeModel.Education); err != nil {
		return nil, fmt.Errorf("failed to unmarshal education: %w", err)
	}

	if err := json.Unmarshal(r.Skills, &resumeModel.Skills); err != nil {
		return nil, fmt.Errorf("failed to unmarshal skills: %w", err)
	}

	if err := json.Unmarshal(r.Languages, &resumeModel.Languages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal languages: %w", err)
	}

	if err := json.Unmarshal(r.Certifications, &resumeModel.Certifications); err != nil {
		return nil, fmt.Errorf("failed to unmarshal certifications: %w", err)
	}

	if err := json.Unmarshal(r.Projects, &resumeModel.Projects); err != nil {
		return nil, fmt.Errorf("failed to unmarshal projects: %w", err)
	}

	if err := json.Unmarshal(r.Achievements, &resumeModel.Achievements); err != nil {
		return nil, fmt.Errorf("failed to unmarshal achievements: %w", err)
	}

	if err := json.Unmarshal(r.VolunteerWork, &resumeModel.VolunteerWork); err != nil {
		return nil, fmt.Errorf("failed to unmarshal volunteer_work: %w", err)
	}

	if err := json.Unmarshal(r.PersonalStatement, &resumeModel.PersonalStatement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal personal_statement: %w", err)
	}

	if r.ProfessionalSummary.Valid {
		resumeModel.ProfessionalSummary = r.ProfessionalSummary.String
	}

	return resumeModel, nil
}

// embeddingsRow represents a row from the resume_embeddings table
type embeddingsRow struct {
	ExperienceEmbedding        string         `db:"experience_embedding"`
	EducationEmbedding         string         `db:"education_embedding"`
	SkillsEmbedding            string         `db:"skills_embedding"`
	LanguagesEmbedding         string         `db:"languages_embedding"`
	PersonalStatementEmbedding sql.NullString `db:"personal_statement_embedding"`
	ModelUsed                  string         `db:"model_used"`
	EmbeddingDim               int            `db:"embedding_dim"`
	GeneratedAt                time.Time      `db:"generated_at"`
}

// ToDomain converts an embeddingsRow to resume.ResumeEmbeddings
func (e *embeddingsRow) ToDomain() *resume.ResumeEmbeddings {
	embeddings := &resume.ResumeEmbeddings{
		ExperienceEmbedding: vectorToFloat32Slice(e.ExperienceEmbedding),
		EducationEmbedding:  vectorToFloat32Slice(e.EducationEmbedding),
		SkillsEmbedding:     vectorToFloat32Slice(e.SkillsEmbedding),
		LanguagesEmbedding:  vectorToFloat32Slice(e.LanguagesEmbedding),
		ModelUsed:           e.ModelUsed,
		EmbeddingDim:        e.EmbeddingDim,
		GeneratedAt:         e.GeneratedAt,
	}

	if e.PersonalStatementEmbedding.Valid {
		embeddings.PersonalStatementEmbedding = vectorToFloat32Slice(e.PersonalStatementEmbedding.String)
	}

	return embeddings
}
