package db

import (
	"fmt"
)

// GraphNode represents a node in the graph visualization.
type GraphNode struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	FolderPath string `json:"folder_path"`
	IsResolved bool   `json:"is_resolved"`
}

// GraphEdge represents an edge (link) in the graph visualization.
type GraphEdge struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Type     string `json:"type"` // "resolved" | "unresolved"
}

// GraphMetadata contains metadata about the graph.
type GraphMetadata struct {
	NodeCount int  `json:"node_count"`
	EdgeCount int  `json:"edge_count"`
	Truncated bool `json:"truncated"`
}

// GraphData contains the complete graph data.
type GraphData struct {
	Nodes    []GraphNode   `json:"nodes"`
	Edges    []GraphEdge   `json:"edges"`
	Metadata GraphMetadata `json:"metadata"`
}

const (
	// MaxGraphNodes is the maximum number of nodes to return in a graph query.
	MaxGraphNodes = 1000
	// MaxGraphEdges is the maximum number of edges to return in a graph query.
	MaxGraphEdges = 5000
)

// GetGlobalGraph returns the complete graph for a user with resolved and unresolved nodes.
// It applies MAX_NODES and MAX_EDGES caps and sets the truncated flag if limits are hit.
//
// Optimized version: 2 queries instead of 4 (PERF P3)
// - Query 1: Combined nodes query (resolved + unresolved via CTE+UNION)
// - Query 2: Combined edges query (resolved + unresolved via UNION ALL)
func (db *DB) GetGlobalGraph(userID int, maxNodes int) (*GraphData, error) {
	if maxNodes <= 0 {
		maxNodes = MaxGraphNodes
	}

	// Query 1: Get all nodes (resolved + unresolved) in a single query
	// Uses CTE to first load truncated resolved nodes, then INNER JOIN for unresolved
	nodesQuery := `
		WITH resolved AS (
			SELECT id, title, folder_path, 1 as is_resolved
			FROM notes
			WHERE user_id = ? AND is_deleted = 0
			ORDER BY title ASC
			LIMIT ?
		)
		SELECT id, title, folder_path, is_resolved FROM resolved
		UNION ALL
		SELECT
			'unresolved:' || ul.target_ref_norm as id,
			COALESCE(MIN(ul.target_ref), ul.target_ref_norm) as title,
			'' as folder_path,
			0 as is_resolved
		FROM unresolved_links ul
		INNER JOIN resolved r ON ul.source_id = r.id
		GROUP BY ul.target_ref_norm
		ORDER BY is_resolved DESC, title ASC
	`

	rows, err := db.Query(nodesQuery, userID, maxNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer rows.Close()

	nodes := []GraphNode{}
	nodeIDSet := make(map[string]bool)
	resolvedNodeCount := 0

	for rows.Next() {
		var node GraphNode
		var isResolvedInt int
		if err := rows.Scan(&node.ID, &node.Title, &node.FolderPath, &isResolvedInt); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		node.IsResolved = isResolvedInt == 1
		if node.IsResolved {
			resolvedNodeCount++
		}
		nodes = append(nodes, node)
		nodeIDSet[node.ID] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nodes: %w", err)
	}

	nodeTruncated := resolvedNodeCount >= maxNodes

	// Query 2: Get all edges (resolved + unresolved) in a single query
	edgesQuery := `
		SELECT l.source_id, l.target_id, 'resolved' as type
		FROM links l
		JOIN notes src ON src.id = l.source_id
		JOIN notes tgt ON tgt.id = l.target_id
		WHERE src.user_id = ? AND tgt.user_id = ?
		  AND src.is_deleted = 0 AND tgt.is_deleted = 0
		UNION ALL
		SELECT ul.source_id, 'unresolved:' || ul.target_ref_norm, 'unresolved'
		FROM unresolved_links ul
		JOIN notes n ON n.id = ul.source_id
		WHERE n.user_id = ? AND n.is_deleted = 0
	`

	rows, err = db.Query(edgesQuery, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query edges: %w", err)
	}
	defer rows.Close()

	edges := []GraphEdge{}

	for rows.Next() {
		var edge GraphEdge
		if err := rows.Scan(&edge.SourceID, &edge.TargetID, &edge.Type); err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}
		// Filter: only include if both nodes are in our set
		// (needed because LIMIT on nodes can truncate, but edges query loads ALL edges)
		if nodeIDSet[edge.SourceID] && nodeIDSet[edge.TargetID] {
			edges = append(edges, edge)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating edges: %w", err)
	}

	// Apply edge cap
	edgeTruncated := false
	if len(edges) > MaxGraphEdges {
		edges = edges[:MaxGraphEdges]
		edgeTruncated = true
	}

	return &GraphData{
		Nodes: nodes,
		Edges: edges,
		Metadata: GraphMetadata{
			NodeCount: len(nodes),
			EdgeCount: len(edges),
			Truncated: nodeTruncated || edgeTruncated,
		},
	}, nil
}

// GetFilteredGraph returns a filtered graph for a user based on folder path.
// It uses the same node/edge capping logic as GetGlobalGraph.
//
// Optimized version: 2 queries instead of 4 (PERF P3)
// - Query 1: Combined nodes query (resolved + unresolved via CTE+UNION)
// - Query 2: Combined edges query (resolved + unresolved via UNION ALL)
func (db *DB) GetFilteredGraph(userID int, folderPath string, maxNodes int) (*GraphData, error) {
	if maxNodes <= 0 {
		maxNodes = MaxGraphNodes
	}

	folderPattern := folderPath + "%"

	// Query 1: Get all nodes (resolved + unresolved) in a single query
	// Uses CTE to first load truncated resolved nodes (filtered by folder), then INNER JOIN for unresolved
	nodesQuery := `
		WITH resolved AS (
			SELECT id, title, folder_path, 1 as is_resolved
			FROM notes
			WHERE user_id = ? AND is_deleted = 0 AND folder_path LIKE ?
			ORDER BY title ASC
			LIMIT ?
		)
		SELECT id, title, folder_path, is_resolved FROM resolved
		UNION ALL
		SELECT
			'unresolved:' || ul.target_ref_norm as id,
			COALESCE(MIN(ul.target_ref), ul.target_ref_norm) as title,
			'' as folder_path,
			0 as is_resolved
		FROM unresolved_links ul
		INNER JOIN resolved r ON ul.source_id = r.id
		GROUP BY ul.target_ref_norm
		ORDER BY is_resolved DESC, title ASC
	`

	rows, err := db.Query(nodesQuery, userID, folderPattern, maxNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer rows.Close()

	nodes := []GraphNode{}
	nodeIDSet := make(map[string]bool)
	resolvedNodeCount := 0

	for rows.Next() {
		var node GraphNode
		var isResolvedInt int
		if err := rows.Scan(&node.ID, &node.Title, &node.FolderPath, &isResolvedInt); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		node.IsResolved = isResolvedInt == 1
		if node.IsResolved {
			resolvedNodeCount++
		}
		nodes = append(nodes, node)
		nodeIDSet[node.ID] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nodes: %w", err)
	}

	nodeTruncated := resolvedNodeCount >= maxNodes

	// Query 2: Get all edges (resolved + unresolved) in a single query
	edgesQuery := `
		SELECT l.source_id, l.target_id, 'resolved' as type
		FROM links l
		JOIN notes src ON src.id = l.source_id
		JOIN notes tgt ON tgt.id = l.target_id
		WHERE src.user_id = ? AND tgt.user_id = ?
		  AND src.is_deleted = 0 AND tgt.is_deleted = 0
		  AND src.folder_path LIKE ? AND tgt.folder_path LIKE ?
		UNION ALL
		SELECT ul.source_id, 'unresolved:' || ul.target_ref_norm, 'unresolved'
		FROM unresolved_links ul
		JOIN notes n ON n.id = ul.source_id
		WHERE n.user_id = ? AND n.is_deleted = 0 AND n.folder_path LIKE ?
	`

	rows, err = db.Query(edgesQuery, userID, userID, folderPattern, folderPattern, userID, folderPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query edges: %w", err)
	}
	defer rows.Close()

	edges := []GraphEdge{}

	for rows.Next() {
		var edge GraphEdge
		if err := rows.Scan(&edge.SourceID, &edge.TargetID, &edge.Type); err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}
		// Filter: only include if both nodes are in our set
		// (needed because LIMIT on nodes can truncate, but edges query loads ALL edges)
		if nodeIDSet[edge.SourceID] && nodeIDSet[edge.TargetID] {
			edges = append(edges, edge)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating edges: %w", err)
	}

	// Apply edge cap
	edgeTruncated := false
	if len(edges) > MaxGraphEdges {
		edges = edges[:MaxGraphEdges]
		edgeTruncated = true
	}

	return &GraphData{
		Nodes: nodes,
		Edges: edges,
		Metadata: GraphMetadata{
			NodeCount: len(nodes),
			EdgeCount: len(edges),
			Truncated: nodeTruncated || edgeTruncated,
		},
	}, nil
}
