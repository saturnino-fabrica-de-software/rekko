-- Create HNSW index for vector similarity search
-- HNSW (Hierarchical Navigable Small World) provides better recall than IVFFlat
-- and doesn't require training data

-- Drop old IVFFlat index if exists (from initial setup)
DROP INDEX CONCURRENTLY IF EXISTS idx_faces_embedding;

-- Create HNSW index with optimized parameters:
-- m = 16: number of bi-directional links (balance between recall and build time)
-- ef_construction = 64: size of dynamic candidate list (higher = better recall, slower build)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_faces_embedding_hnsw
ON faces USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

COMMENT ON INDEX idx_faces_embedding_hnsw IS 'HNSW index for fast cosine similarity search on face embeddings';
