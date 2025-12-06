-- Resume Processing Jobs Table
CREATE TABLE IF NOT EXISTS resume_processing_jobs (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    resume_id VARCHAR(255) NULL,
    
    -- Job metadata
    status VARCHAR(50) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(10) NOT NULL,
    title VARCHAR(255) NOT NULL,
    
    -- Tracking
    attempt_count INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    
    -- Error details
    error_message TEXT NULL,
    error_details JSONB NULL,
    
    -- Progress tracking
    current_step VARCHAR(100) NULL,
    progress_percentage INT DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    failed_at TIMESTAMP NULL,
    next_retry_at TIMESTAMP NULL,
    
    -- Request details (for retry)
    request_payload JSONB NOT NULL,
    
    -- Foreign keys
    CONSTRAINT fk_resume_processing_tenant FOREIGN KEY (tenant_id) 
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_resume_processing_resume FOREIGN KEY (resume_id) 
        REFERENCES resumes(id) ON DELETE SET NULL,
    
    -- Check constraints
    CONSTRAINT chk_status CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    CONSTRAINT chk_progress CHECK (progress_percentage >= 0 AND progress_percentage <= 100),
    CONSTRAINT chk_attempts CHECK (attempt_count >= 0 AND attempt_count <= max_attempts)
);

-- Indexes for performance
CREATE INDEX idx_resume_jobs_tenant_status ON resume_processing_jobs (tenant_id, status);
CREATE INDEX idx_resume_jobs_status_next_retry ON resume_processing_jobs (status, next_retry_at);
CREATE INDEX idx_resume_jobs_created_at ON resume_processing_jobs (created_at DESC);
CREATE INDEX idx_resume_jobs_resume_id ON resume_processing_jobs (resume_id) WHERE resume_id IS NOT NULL;

-- Comments for documentation
COMMENT ON TABLE resume_processing_jobs IS 'Tracks async resume processing jobs with retry logic';
COMMENT ON COLUMN resume_processing_jobs.status IS 'Job status: pending, processing, completed, failed';
COMMENT ON COLUMN resume_processing_jobs.current_step IS 'Current processing step: uploading, parsing, embedding, saving';
COMMENT ON COLUMN resume_processing_jobs.request_payload IS 'Original ParseResumeRequest for retry purposes';
COMMENT ON COLUMN resume_processing_jobs.error_details IS 'Detailed error information for debugging';

