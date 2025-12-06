package resume

import (
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// Resume represents a parsed resume with embeddings
type Resume struct {
	ID       kernel.ResumeID `db:"id" json:"id"`
	TenantID kernel.TenantID `db:"tenant_id" json:"tenant_id"`

	// Resume metadata
	Title     string `db:"title" json:"title"`           // e.g., "Software Engineer Resume", "Senior Developer"
	IsActive  bool   `db:"is_active" json:"is_active"`   // Active for job search
	IsDefault bool   `db:"is_default" json:"is_default"` // Default resume
	Version   int    `db:"version" json:"version"`       // Version number (auto-incremented)

	// Personal Information
	PersonalInfo PersonalInfo `db:"personal_info" json:"personal_info"`

	// Work Experience (most important - 70% weight)
	WorkExperience []WorkExperience `db:"work_experience" json:"work_experience"`

	// Education (20% weight)
	Education []Education `db:"education" json:"education"`

	// Skills (5% weight each)
	Skills Skills `db:"skills" json:"skills"`

	// Languages (5% weight)
	Languages []Language `db:"languages" json:"languages"`

	// Certifications
	Certifications []Certification `db:"certifications" json:"certifications"`

	// Projects (optional)
	Projects []Project `db:"projects" json:"projects,omitempty"`

	// Achievements/Awards (optional)
	Achievements []string `db:"achievements" json:"achievements,omitempty"`

	// Volunteer Work (optional)
	VolunteerWork []VolunteerExperience `db:"volunteer_work" json:"volunteer_work,omitempty"`

	// Professional Summary (optional)
	ProfessionalSummary string `db:"professional_summary" json:"professional_summary,omitempty"`

	// Personal Statement/Essay
	PersonalStatement PersonalStatement `db:"personal_statement" json:"personal_statement,omitempty"`

	// Multi-Section Embeddings
	Embeddings ResumeEmbeddings `db:"embeddings" json:"embeddings"`

	// File Metadata
	FileURL       string    `db:"file_url" json:"file_url"`
	FileName      string    `db:"file_name" json:"file_name"`
	FileType      string    `db:"file_type" json:"file_type"`
	ParsedAt      time.Time `db:"parsed_at" json:"parsed_at"`
	LastUpdatedAt time.Time `db:"last_updated_at" json:"last_updated_at"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// PersonalInfo contains personal information
type PersonalInfo struct {
	FullName    string            `json:"full_name"`
	Email       string            `json:"email"`
	Phone       string            `json:"phone"`
	Location    Location          `json:"location"`
	LinkedIn    string            `json:"linkedin,omitempty"`
	GitHub      string            `json:"github,omitempty"`
	Portfolio   string            `json:"portfolio,omitempty"`
	Website     string            `json:"website,omitempty"`
	SocialLinks map[string]string `json:"social_links,omitempty"`
}

type Location struct {
	City    string `json:"city"`
	State   string `json:"state,omitempty"`
	Country string `json:"country"`
	ZipCode string `json:"zip_code,omitempty"`
}

// WorkExperience represents a single work experience entry
type WorkExperience struct {
	Company               string   `json:"company"`
	Title                 string   `json:"title"`
	StartDate             string   `json:"start_date"` // YYYY-MM format
	EndDate               string   `json:"end_date"`   // YYYY-MM or "Present"
	DurationMonths        int      `json:"duration_months"`
	DescriptionNormalized string   `json:"description_normalized"`
	Achievements          []string `json:"achievements,omitempty"`
	SkillsUsed            []string `json:"skills_used,omitempty"`
	Industry              string   `json:"industry,omitempty"`
	Location              string   `json:"location,omitempty"`
	EmploymentType        string   `json:"employment_type,omitempty"`
}

// Education represents a single education entry
type Education struct {
	Institution           string   `json:"institution"`
	Degree                string   `json:"degree"`
	Field                 string   `json:"field"`
	GraduationDate        string   `json:"graduation_date"`
	GPA                   *float64 `json:"gpa,omitempty"`
	Honors                []string `json:"honors,omitempty"`
	Coursework            []string `json:"coursework,omitempty"`
	DescriptionNormalized string   `json:"description_normalized"`
}

// Skills categorized for better matching
type Skills struct {
	HardSkills []Skill `json:"hard_skills"`
	SoftSkills []Skill `json:"soft_skills"`
}

type Skill struct {
	Name             string `json:"name"`
	ProficiencyLevel string `json:"proficiency_level,omitempty"`
	YearsExperience  *int   `json:"years_experience,omitempty"`
}

// Language represents language proficiency
type Language struct {
	Language    string `json:"language"`
	Proficiency string `json:"proficiency"`
}

// Certification represents professional certifications
type Certification struct {
	Name           string `json:"name"`
	Issuer         string `json:"issuer"`
	IssueDate      string `json:"date"`
	ExpirationDate string `json:"expiration,omitempty"`
	CredentialID   string `json:"credential_id,omitempty"`
	CredentialURL  string `json:"credential_url,omitempty"`
}

// Project represents projects
type Project struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Technologies []string `json:"technologies,omitempty"`
	Duration     string   `json:"duration,omitempty"`
	URL          string   `json:"url,omitempty"`
	Outcomes     []string `json:"outcomes,omitempty"`
	Role         string   `json:"role,omitempty"`
}

// VolunteerExperience represents volunteer work
type VolunteerExperience struct {
	Organization string   `json:"organization"`
	Role         string   `json:"role"`
	StartDate    string   `json:"start_date"`
	EndDate      string   `json:"end_date"`
	Description  string   `json:"description,omitempty"`
	Achievements []string `json:"achievements,omitempty"`
}

// PersonalStatement for essays
type PersonalStatement struct {
	WhyThisCompany string     `json:"why_this_company,omitempty"`
	WhyThisRole    string     `json:"why_this_role,omitempty"`
	CareerGoals    string     `json:"career_goals,omitempty"`
	UniqueValue    string     `json:"unique_value,omitempty"`
	Essay          string     `json:"essay,omitempty"`
	WrittenAt      *time.Time `json:"written_at,omitempty"`
}

// ResumeEmbeddings - Multi-section embeddings for semantic search
type ResumeEmbeddings struct {
	ExperienceEmbedding        []float32 `json:"experience_embedding"`
	EducationEmbedding         []float32 `json:"education_embedding"`
	SkillsEmbedding            []float32 `json:"skills_embedding"`
	LanguagesEmbedding         []float32 `json:"languages_embedding"`
	PersonalStatementEmbedding []float32 `json:"personal_statement_embedding,omitempty"`
	ModelUsed                  string    `json:"model_used"`
	EmbeddingDim               int       `json:"embedding_dim"`
	GeneratedAt                time.Time `json:"generated_at"`
}

// ============================================================================
// Domain Methods
// ============================================================================

// Activate sets the resume as active
func (r *Resume) Activate() {
	r.IsActive = true
	r.LastUpdatedAt = time.Now()
}

// Deactivate sets the resume as inactive
func (r *Resume) Deactivate() {
	r.IsActive = false
	r.LastUpdatedAt = time.Now()
}

// SetAsDefault sets the resume as default
func (r *Resume) SetAsDefault() {
	r.IsDefault = true
	r.LastUpdatedAt = time.Now()
}

// UnsetAsDefault removes default status
func (r *Resume) UnsetAsDefault() {
	r.IsDefault = false
	r.LastUpdatedAt = time.Now()
}

// HasWorkExperience checks if resume has work experience
func (r *Resume) HasWorkExperience() bool {
	return len(r.WorkExperience) > 0
}

// HasEducation checks if resume has education
func (r *Resume) HasEducation() bool {
	return len(r.Education) > 0
}

// TotalYearsOfExperience calculates total years of work experience
func (r *Resume) TotalYearsOfExperience() float64 {
	totalMonths := 0
	for _, exp := range r.WorkExperience {
		totalMonths += exp.DurationMonths
	}
	return float64(totalMonths) / 12.0
}

// HasSkill checks if resume has a specific skill
func (r *Resume) HasSkill(skillName string) bool {
	for _, skill := range r.Skills.HardSkills {
		if skill.Name == skillName {
			return true
		}
	}
	for _, skill := range r.Skills.SoftSkills {
		if skill.Name == skillName {
			return true
		}
	}
	return false
}

// GetAllSkills returns all skills as a flat list
func (r *Resume) GetAllSkills() []string {
	skills := make([]string, 0)
	for _, skill := range r.Skills.HardSkills {
		skills = append(skills, skill.Name)
	}
	for _, skill := range r.Skills.SoftSkills {
		skills = append(skills, skill.Name)
	}
	return skills
}

// HasCertification checks if resume has a specific certification
func (r *Resume) HasCertification(certName string) bool {
	for _, cert := range r.Certifications {
		if cert.Name == certName {
			return true
		}
	}
	return false
}

// SpeaksLanguage checks if the person speaks a specific language
func (r *Resume) SpeaksLanguage(lang string) bool {
	for _, language := range r.Languages {
		if language.Language == lang {
			return true
		}
	}
	return false
}

// HasPersonalStatement checks if personal statement is provided
func (r *Resume) HasPersonalStatement() bool {
	return r.PersonalStatement.Essay != "" ||
		r.PersonalStatement.WhyThisCompany != "" ||
		r.PersonalStatement.WhyThisRole != "" ||
		r.PersonalStatement.CareerGoals != "" ||
		r.PersonalStatement.UniqueValue != ""
}

// HasEmbeddings checks if resume has been processed for embeddings
func (r *Resume) HasEmbeddings() bool {
	return len(r.Embeddings.ExperienceEmbedding) > 0 ||
		len(r.Embeddings.EducationEmbedding) > 0 ||
		len(r.Embeddings.SkillsEmbedding) > 0 ||
		len(r.Embeddings.LanguagesEmbedding) > 0 ||
		len(r.Embeddings.PersonalStatementEmbedding) > 0
}

// IsComplete checks if resume has minimum required information
func (r *Resume) IsComplete() bool {
	return r.PersonalInfo.FullName != "" &&
		r.PersonalInfo.Email != "" &&
		(r.HasWorkExperience() || r.HasEducation())
}

// GetLatestPosition returns the most recent work position
func (r *Resume) GetLatestPosition() *WorkExperience {
	if !r.HasWorkExperience() {
		return nil
	}
	return &r.WorkExperience[0]
}

// GetHighestEducation returns the highest level of education
func (r *Resume) GetHighestEducation() *Education {
	if !r.HasEducation() {
		return nil
	}
	return &r.Education[0]
}

// UpdateEmbeddings updates the embeddings for the resume
func (r *Resume) UpdateEmbeddings(embeddings ResumeEmbeddings) {
	r.Embeddings = embeddings
	r.LastUpdatedAt = time.Now()
}

// AddPersonalStatement adds or updates the personal statement
func (r *Resume) AddPersonalStatement(statement PersonalStatement) {
	now := time.Now()
	statement.WrittenAt = &now
	r.PersonalStatement = statement
	r.LastUpdatedAt = now
}
