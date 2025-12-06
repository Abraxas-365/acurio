-- ============================================================================
-- Recruitment: Resume Management Migration
-- ============================================================================

-- Enable pgvector extension for embeddings (vector similarity search)
CREATE EXTENSION IF NOT EXISTS vector;

-- ============================================================================
-- RESUMES
-- ============================================================================

CREATE TABLE resumes (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    
    -- Resume metadata
    title VARCHAR(500) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    version INTEGER NOT NULL DEFAULT 1,
    
    -- Personal Information (JSONB for flexibility)
    personal_info JSONB NOT NULL,
    
    -- Resume sections (JSONB arrays)
    work_experience JSONB NOT NULL DEFAULT '[]'::jsonb,
    education JSONB NOT NULL DEFAULT '[]'::jsonb,
    skills JSONB NOT NULL DEFAULT '{"hard_skills":[],"soft_skills":[]}'::jsonb,
    languages JSONB NOT NULL DEFAULT '[]'::jsonb,
    certifications JSONB NOT NULL DEFAULT '[]'::jsonb,
    projects JSONB NOT NULL DEFAULT '[]'::jsonb,
    achievements JSONB NOT NULL DEFAULT '[]'::jsonb,
    volunteer_work JSONB NOT NULL DEFAULT '[]'::jsonb,
    
    -- Text sections
    professional_summary TEXT,
    personal_statement JSONB,
    
    -- File metadata
    file_url TEXT NOT NULL,
    file_name VARCHAR(500) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    
    -- Timestamps
    parsed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_resumes_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT chk_resume_version CHECK (version > 0),
    CONSTRAINT chk_resume_file_type CHECK (file_type IN ('pdf', 'jpg', 'jpeg', 'png'))
);

-- Indexes for common queries
CREATE INDEX idx_resumes_tenant_id ON resumes(tenant_id);
CREATE INDEX idx_resumes_tenant_active ON resumes(tenant_id, is_active);
CREATE INDEX idx_resumes_tenant_default ON resumes(tenant_id, is_default);
CREATE INDEX idx_resumes_is_active ON resumes(is_active);
CREATE INDEX idx_resumes_is_default ON resumes(is_default);
CREATE INDEX idx_resumes_created_at ON resumes(created_at DESC);
CREATE INDEX idx_resumes_last_updated ON resumes(last_updated_at DESC);
CREATE INDEX idx_resumes_version ON resumes(version);

-- JSONB GIN indexes for fast JSON queries
CREATE INDEX idx_resumes_personal_info ON resumes USING gin (personal_info);
CREATE INDEX idx_resumes_work_experience ON resumes USING gin (work_experience);
CREATE INDEX idx_resumes_education ON resumes USING gin (education);
CREATE INDEX idx_resumes_skills ON resumes USING gin (skills);
CREATE INDEX idx_resumes_languages ON resumes USING gin (languages);

-- Specific JSONB path indexes for common searches
CREATE INDEX idx_resumes_email ON resumes ((personal_info->>'email'));
CREATE INDEX idx_resumes_full_name ON resumes ((personal_info->>'full_name'));

-- Ensure only one default resume per tenant
CREATE UNIQUE INDEX idx_resumes_unique_default 
    ON resumes(tenant_id) 
    WHERE is_default = TRUE;

-- ============================================================================
-- RESUME EMBEDDINGS
-- ============================================================================

CREATE TABLE resume_embeddings (
    id VARCHAR(255) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    resume_id VARCHAR(255) NOT NULL,
    
    -- Multi-section embeddings for semantic search
    experience_embedding vector(1536),     -- OpenAI text-embedding-3-small dimension
    education_embedding vector(1536),
    skills_embedding vector(1536),
    languages_embedding vector(1536),
    personal_statement_embedding vector(1536),
    
    -- Embedding metadata
    model_used VARCHAR(100) NOT NULL DEFAULT 'text-embedding-3-small',
    embedding_dim INTEGER NOT NULL DEFAULT 1536,
    generated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_resume_embeddings_resume FOREIGN KEY (resume_id) REFERENCES resumes(id) ON DELETE CASCADE,
    CONSTRAINT uq_resume_embeddings_resume UNIQUE (resume_id)
);

-- Vector similarity search indexes (HNSW for fast approximate search)
CREATE INDEX idx_resume_embeddings_experience 
    ON resume_embeddings USING hnsw (experience_embedding vector_cosine_ops);

CREATE INDEX idx_resume_embeddings_education 
    ON resume_embeddings USING hnsw (education_embedding vector_cosine_ops);

CREATE INDEX idx_resume_embeddings_skills 
    ON resume_embeddings USING hnsw (skills_embedding vector_cosine_ops);

CREATE INDEX idx_resume_embeddings_languages 
    ON resume_embeddings USING hnsw (languages_embedding vector_cosine_ops);

CREATE INDEX idx_resume_embeddings_personal_statement 
    ON resume_embeddings USING hnsw (personal_statement_embedding vector_cosine_ops);

-- Regular indexes
CREATE INDEX idx_resume_embeddings_resume_id ON resume_embeddings(resume_id);
CREATE INDEX idx_resume_embeddings_generated_at ON resume_embeddings(generated_at);

-- ============================================================================
-- RESUME STATISTICS (Computed View)
-- ============================================================================

CREATE VIEW resume_stats AS
SELECT 
    tenant_id,
    COUNT(*) as total_resumes,
    COUNT(*) FILTER (WHERE is_active = TRUE) as active_resumes,
    COUNT(*) FILTER (WHERE is_active = FALSE) as inactive_resumes,
    COUNT(*) FILTER (WHERE is_default = TRUE) as default_resumes,
    COUNT(DISTINCT (personal_info->>'email')) as unique_candidates,
    AVG(version) as avg_version,
    MAX(created_at) as latest_resume_at,
    MAX(last_updated_at) as last_updated_at
FROM resumes
GROUP BY tenant_id;

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to get total years of experience from work_experience JSONB
CREATE OR REPLACE FUNCTION calculate_total_experience_months(work_exp JSONB)
RETURNS INTEGER AS $$
DECLARE
    total_months INTEGER := 0;
    exp JSONB;
BEGIN
    FOR exp IN SELECT * FROM jsonb_array_elements(work_exp)
    LOOP
        total_months := total_months + COALESCE((exp->>'duration_months')::INTEGER, 0);
    END LOOP;
    RETURN total_months;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to extract all skills from skills JSONB
CREATE OR REPLACE FUNCTION extract_all_skills(skills_json JSONB)
RETURNS TEXT[] AS $$
DECLARE
    all_skills TEXT[] := '{}';
    skill JSONB;
BEGIN
    -- Extract hard skills
    FOR skill IN SELECT * FROM jsonb_array_elements(skills_json->'hard_skills')
    LOOP
        all_skills := array_append(all_skills, skill->>'name');
    END LOOP;
    
    -- Extract soft skills
    FOR skill IN SELECT * FROM jsonb_array_elements(skills_json->'soft_skills')
    LOOP
        all_skills := array_append(all_skills, skill->>'name');
    END LOOP;
    
    RETURN all_skills;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function for semantic search combining all embeddings
CREATE OR REPLACE FUNCTION search_resumes_semantic(
    query_embedding vector(1536),
    tenant_filter VARCHAR(255) DEFAULT NULL,
    active_only BOOLEAN DEFAULT TRUE,
    result_limit INTEGER DEFAULT 10
)
RETURNS TABLE(
    resume_id VARCHAR(255),
    similarity_score FLOAT,
    experience_score FLOAT,
    education_score FLOAT,
    skills_score FLOAT,
    combined_score FLOAT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        r.id as resume_id,
        (
            (1 - (e.experience_embedding <=> query_embedding)) * 0.4 +
            (1 - (e.education_embedding <=> query_embedding)) * 0.2 +
            (1 - (e.skills_embedding <=> query_embedding)) * 0.3 +
            (1 - (e.languages_embedding <=> query_embedding)) * 0.1
        ) as similarity_score,
        (1 - (e.experience_embedding <=> query_embedding)) as experience_score,
        (1 - (e.education_embedding <=> query_embedding)) as education_score,
        (1 - (e.skills_embedding <=> query_embedding)) as skills_score,
        (
            (1 - (e.experience_embedding <=> query_embedding)) * 0.4 +
            (1 - (e.education_embedding <=> query_embedding)) * 0.2 +
            (1 - (e.skills_embedding <=> query_embedding)) * 0.3 +
            (1 - (e.languages_embedding <=> query_embedding)) * 0.1
        ) as combined_score
    FROM resumes r
    INNER JOIN resume_embeddings e ON r.id = e.resume_id
    WHERE 
        (tenant_filter IS NULL OR r.tenant_id = tenant_filter)
        AND (NOT active_only OR r.is_active = TRUE)
    ORDER BY combined_score DESC
    LIMIT result_limit;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Trigger to update last_updated_at on resume changes
CREATE TRIGGER update_resumes_updated_at 
    BEFORE UPDATE ON resumes
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger to prevent multiple default resumes per tenant
CREATE OR REPLACE FUNCTION ensure_single_default_resume()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_default = TRUE THEN
        -- Unset other default resumes for this tenant
        UPDATE resumes 
        SET is_default = FALSE 
        WHERE tenant_id = NEW.tenant_id 
          AND id != NEW.id 
          AND is_default = TRUE;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER enforce_single_default_resume
    BEFORE INSERT OR UPDATE ON resumes
    FOR EACH ROW
    WHEN (NEW.is_default = TRUE)
    EXECUTE FUNCTION ensure_single_default_resume();

-- Trigger to increment tenant resume count (if you want to track this)
CREATE OR REPLACE FUNCTION update_tenant_resume_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- Increment count on insert
        UPDATE tenants 
        SET updated_at = CURRENT_TIMESTAMP 
        WHERE id = NEW.tenant_id;
    ELSIF TG_OP = 'DELETE' THEN
        -- Decrement count on delete
        UPDATE tenants 
        SET updated_at = CURRENT_TIMESTAMP 
        WHERE id = OLD.tenant_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_tenant_on_resume_change
    AFTER INSERT OR DELETE ON resumes
    FOR EACH ROW
    EXECUTE FUNCTION update_tenant_resume_count();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE resumes IS 'Parsed and structured resume data with multi-version support';
COMMENT ON TABLE resume_embeddings IS 'Vector embeddings for semantic resume search';
COMMENT ON VIEW resume_stats IS 'Aggregated statistics about resumes per tenant';

COMMENT ON COLUMN resumes.id IS 'Unique resume identifier';
COMMENT ON COLUMN resumes.tenant_id IS 'Tenant/company that owns this resume';
COMMENT ON COLUMN resumes.title IS 'Resume title/name (e.g., "Software Engineer Resume")';
COMMENT ON COLUMN resumes.is_active IS 'Whether this resume is active for job search';
COMMENT ON COLUMN resumes.is_default IS 'Whether this is the default resume for the tenant';
COMMENT ON COLUMN resumes.version IS 'Resume version number (auto-incremented on updates)';
COMMENT ON COLUMN resumes.personal_info IS 'JSONB: name, email, phone, location, links';
COMMENT ON COLUMN resumes.work_experience IS 'JSONB array: work history with achievements';
COMMENT ON COLUMN resumes.education IS 'JSONB array: educational background';
COMMENT ON COLUMN resumes.skills IS 'JSONB: hard_skills and soft_skills arrays';
COMMENT ON COLUMN resumes.languages IS 'JSONB array: languages with proficiency levels';
COMMENT ON COLUMN resumes.certifications IS 'JSONB array: professional certifications';
COMMENT ON COLUMN resumes.projects IS 'JSONB array: personal/professional projects';
COMMENT ON COLUMN resumes.achievements IS 'JSONB array: awards and achievements';
COMMENT ON COLUMN resumes.volunteer_work IS 'JSONB array: volunteer experience';
COMMENT ON COLUMN resumes.professional_summary IS 'Brief professional summary';
COMMENT ON COLUMN resumes.personal_statement IS 'JSONB: why_this_company, career_goals, etc';
COMMENT ON COLUMN resumes.file_url IS 'S3/storage path to original resume file';
COMMENT ON COLUMN resumes.file_name IS 'Original filename';
COMMENT ON COLUMN resumes.file_type IS 'File type: pdf, jpg, jpeg, png';

COMMENT ON COLUMN resume_embeddings.experience_embedding IS 'Vector embedding of work experience section';
COMMENT ON COLUMN resume_embeddings.education_embedding IS 'Vector embedding of education section';
COMMENT ON COLUMN resume_embeddings.skills_embedding IS 'Vector embedding of skills section';
COMMENT ON COLUMN resume_embeddings.languages_embedding IS 'Vector embedding of languages section';
COMMENT ON COLUMN resume_embeddings.personal_statement_embedding IS 'Vector embedding of personal statement';
COMMENT ON COLUMN resume_embeddings.model_used IS 'OpenAI model used for embeddings';
COMMENT ON COLUMN resume_embeddings.embedding_dim IS 'Dimension of embedding vectors (1536 for text-embedding-3-small)';

-- ============================================================================
-- SAMPLE DATA (Optional - for development/testing)
-- ============================================================================

-- Uncomment to insert sample data for testing
/*
INSERT INTO resumes (id, tenant_id, title, personal_info, work_experience, education, skills, file_url, file_name, file_type)
VALUES (
    'resume-sample-001',
    'tenant-001',
    'Software Engineer Resume',
    '{"full_name": "John Doe", "email": "john@example.com", "phone": "+1234567890", "location": {"city": "San Francisco", "country": "USA"}}',
    '[{"company": "Tech Corp", "title": "Senior Software Engineer", "start_date": "2020-01", "end_date": "Present", "duration_months": 48, "description_normalized": "Led development of microservices architecture"}]',
    '[{"institution": "MIT", "degree": "Bachelor of Science", "field": "Computer Science", "graduation_date": "2019-05"}]',
    '{"hard_skills": [{"name": "Python"}, {"name": "Go"}, {"name": "PostgreSQL"}], "soft_skills": [{"name": "Leadership"}, {"name": "Communication"}]}',
    'resumes/tenant-001/2024/01/sample.pdf',
    'john_doe_resume.pdf',
    'pdf'
);
*/

