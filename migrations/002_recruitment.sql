-- ============================================================================
-- Recruitment System Migration
-- ============================================================================

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;
-- ============================================================================
-- CANDIDATES
-- ============================================================================

CREATE TABLE candidates (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    dni_type VARCHAR(50),
    dni_number VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    archived_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_candidate_status CHECK (status IN ('ACTIVE', 'INACTIVE', 'ARCHIVED')),
    CONSTRAINT chk_dni_type CHECK (dni_type IN ('DNI', 'CE', 'PASSPORT', 'RUC') OR dni_type IS NULL),
    CONSTRAINT uq_candidates_email UNIQUE (email),
    CONSTRAINT uq_candidates_dni UNIQUE (dni_type, dni_number)
);

CREATE INDEX idx_candidates_email ON candidates(email);
CREATE INDEX idx_candidates_phone ON candidates(phone);
CREATE INDEX idx_candidates_status ON candidates(status);
CREATE INDEX idx_candidates_dni ON candidates(dni_type, dni_number);
CREATE INDEX idx_candidates_created_at ON candidates(created_at);

-- ============================================================================
-- JOBS
-- ============================================================================

CREATE TABLE jobs (
    id VARCHAR(255) PRIMARY KEY,
    job_title VARCHAR(500) NOT NULL,
    job_description TEXT NOT NULL,
    job_position VARCHAR(255) NOT NULL,
    general_requirements JSONB DEFAULT '[]'::jsonb,
    benefits JSONB DEFAULT '[]'::jsonb,
    posted_by VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    published_at TIMESTAMP,
    archived_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_jobs_posted_by FOREIGN KEY (posted_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT chk_job_status CHECK (status IN ('DRAFT', 'PUBLISHED', 'CLOSED', 'ARCHIVED'))
);

CREATE INDEX idx_jobs_posted_by ON jobs(posted_by);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_published_at ON jobs(published_at);
CREATE INDEX idx_jobs_archived_at ON jobs(archived_at);
CREATE INDEX idx_jobs_created_at ON jobs(created_at);
CREATE INDEX idx_jobs_job_title ON jobs(job_title);
CREATE INDEX idx_jobs_job_position ON jobs(job_position);

-- ============================================================================
-- APPLICATIONS
-- ============================================================================

CREATE TABLE applications (
    id VARCHAR(255) PRIMARY KEY,
    job_id VARCHAR(255) NOT NULL,
    candidate_id VARCHAR(255) NOT NULL,
    resume_summary TEXT,
    resume_embedding vector(1536), -- For OpenAI embeddings, adjust dimension as needed
    resume_bucket_url TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'SUBMITTED',
    reviewer_id VARCHAR(255),
    submitted_by VARCHAR(255),
    status_changed_at TIMESTAMP,
    archived_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_applications_job FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    CONSTRAINT fk_applications_candidate FOREIGN KEY (candidate_id) REFERENCES candidates(id) ON DELETE CASCADE,
    CONSTRAINT fk_applications_reviewer FOREIGN KEY (reviewer_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_applications_submitted_by FOREIGN KEY (submitted_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_applications_job_candidate UNIQUE (job_id, candidate_id),
    CONSTRAINT chk_application_status CHECK (status IN (
        'SUBMITTED', 
        'UNDER_REVIEW', 
        'SHORTLISTED', 
        'INTERVIEWING', 
        'APPROVED', 
        'REJECTED', 
        'WITHDRAWN', 
        'ARCHIVED'
    ))
);

CREATE INDEX idx_applications_job_id ON applications(job_id);
CREATE INDEX idx_applications_candidate_id ON applications(candidate_id);
CREATE INDEX idx_applications_reviewer_id ON applications(reviewer_id);
CREATE INDEX idx_applications_status ON applications(status);
CREATE INDEX idx_applications_created_at ON applications(created_at);
CREATE INDEX idx_applications_status_changed_at ON applications(status_changed_at);


-- Create vector index for fast similarity search
CREATE INDEX idx_applications_resume_embedding ON applications 
USING ivfflat (resume_embedding vector_cosine_ops) WITH (lists = 100);
-- ============================================================================
-- TRIGGERS for updated_at
-- ============================================================================

CREATE TRIGGER update_candidates_updated_at BEFORE UPDATE ON candidates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_applications_updated_at BEFORE UPDATE ON applications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- TRIGGER: Auto-update tenant user count (optional)
-- ============================================================================

-- Increment user count when a candidate is created via submitted_by
CREATE OR REPLACE FUNCTION increment_application_count()
RETURNS TRIGGER AS $$
BEGIN
    -- You can add business logic here if needed
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE candidates IS 'Candidate profiles for job applications';
COMMENT ON TABLE jobs IS 'Job postings created by tenant users';
COMMENT ON TABLE applications IS 'Job applications linking candidates to jobs';

COMMENT ON COLUMN candidates.status IS 'Candidate account status: ACTIVE, INACTIVE, ARCHIVED';
COMMENT ON COLUMN candidates.dni_type IS 'Document type: DNI, CE (Foreign ID), PASSPORT, RUC';
COMMENT ON COLUMN jobs.status IS 'Job status: DRAFT, PUBLISHED, CLOSED, ARCHIVED';
COMMENT ON COLUMN jobs.general_requirements IS 'JSON array of job requirements';
COMMENT ON COLUMN jobs.benefits IS 'JSON array of job benefits';
COMMENT ON COLUMN applications.status IS 'Application workflow status';
COMMENT ON COLUMN applications.resume_embedding IS 'Vector embedding of resume for semantic search';
COMMENT ON COLUMN applications.resume_bucket_url IS 'Cloud storage path to resume file';

-- ============================================================================
-- OPTIONAL: Sample Data for Testing
-- ============================================================================

-- Uncomment to insert sample data
/*
-- Insert a sample candidate
INSERT INTO candidates (id, email, phone, first_name, last_name, dni_type, dni_number, status)
VALUES 
    ('cand-001', 'john.doe@example.com', '+51999888777', 'John', 'Doe', 'DNI', '12345678', 'ACTIVE');

-- Insert a sample job (assuming user 'user-001' exists)
INSERT INTO jobs (id, job_title, job_description, job_position, posted_by, status)
VALUES 
    ('job-001', 'Senior Software Engineer', 'Looking for experienced backend developer', 'Backend Developer', 'user-001', 'PUBLISHED');

-- Insert a sample application
INSERT INTO applications (id, job_id, candidate_id, resume_summary, status)
VALUES 
    ('app-001', 'job-001', 'cand-001', 'Experienced software engineer with 5+ years', 'SUBMITTED');
*/
