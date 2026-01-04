-- Remove HNSW index for vector similarity search
DROP INDEX CONCURRENTLY IF EXISTS idx_faces_embedding_hnsw;
