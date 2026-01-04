-- Create audit table for face search operations (LGPD compliance)
-- This table tracks all search operations without storing biometric data

CREATE TABLE IF NOT EXISTS search_audits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    results_count INTEGER NOT NULL DEFAULT 0,
    top_match_external_id VARCHAR(255),
    top_match_similarity DECIMAL(5,4),
    threshold DECIMAL(3,2) NOT NULL,
    max_results INTEGER NOT NULL,
    latency_ms INTEGER NOT NULL,
    client_ip INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_results_count CHECK (results_count >= 0),
    CONSTRAINT valid_similarity CHECK (top_match_similarity IS NULL OR (top_match_similarity >= 0 AND top_match_similarity <= 1)),
    CONSTRAINT valid_threshold CHECK (threshold >= 0 AND threshold <= 1),
    CONSTRAINT valid_max_results CHECK (max_results > 0),
    CONSTRAINT valid_latency CHECK (latency_ms >= 0)
);

-- Index for tenant-based queries (most common access pattern)
CREATE INDEX idx_search_audits_tenant_created ON search_audits(tenant_id, created_at DESC);

-- Index for analytics queries
CREATE INDEX idx_search_audits_created ON search_audits(created_at DESC);

-- Partial index for searches with matches (for success rate analytics)
CREATE INDEX idx_search_audits_with_matches ON search_audits(tenant_id, created_at DESC)
WHERE results_count > 0;

COMMENT ON TABLE search_audits IS 'Audit log for face search operations (LGPD compliance) - does not store biometric data';
COMMENT ON COLUMN search_audits.results_count IS 'Number of faces found above threshold';
COMMENT ON COLUMN search_audits.top_match_external_id IS 'External ID of the best match (if any)';
COMMENT ON COLUMN search_audits.top_match_similarity IS 'Similarity score of the best match (0-1 scale)';
COMMENT ON COLUMN search_audits.threshold IS 'Minimum similarity threshold used in search';
COMMENT ON COLUMN search_audits.max_results IS 'Maximum number of results requested';
COMMENT ON COLUMN search_audits.latency_ms IS 'Search operation latency in milliseconds';
COMMENT ON COLUMN search_audits.client_ip IS 'Client IP address for audit trail';
