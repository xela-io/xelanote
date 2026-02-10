-- Migration 027: Add indexes for graph query optimization
-- Context: PERF P3 - Graph 4-Query Split Optimierung
--
-- This index supports the INNER JOIN optimization in graph queries
-- where we filter unresolved links to only include those from loaded (truncated) sources.
--
-- Note: PRIMARY KEY (source_id, target_ref) already exists on unresolved_links.
-- SQLite can use the PK index for source_id-only lookups (leftmost column of PK).
-- This dedicated index MIGHT be more efficient - benchmark will decide if it helps.

CREATE INDEX IF NOT EXISTS idx_unresolved_links_source
ON unresolved_links(source_id);

-- Note: idx_unresolved_norm (target_ref_norm) already exists from earlier migrations
-- and is used for GROUP BY operations in the new query structure.
