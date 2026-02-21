package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Canvas content limits.
const (
	MaxCanvasNodes       = 500
	MaxCanvasEdges       = 2000
	MaxCanvasContentSize = 100 * 1024 // 100KB
)

var hexColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// canvasContent represents the top-level JSON Canvas structure for validation.
type canvasContent struct {
	Nodes []canvasNodeV `json:"nodes"`
	Edges []canvasEdgeV `json:"edges"`
}

type canvasNodeV struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	X      *int   `json:"x"`
	Y      *int   `json:"y"`
	Width  *int   `json:"width"`
	Height *int   `json:"height"`
	Color  string `json:"color,omitempty"`
	// Type-specific fields
	Text  string `json:"text,omitempty"`  // text node
	File  string `json:"file,omitempty"`  // file node
	URL   string `json:"url,omitempty"`   // link node
	Label string `json:"label,omitempty"` // group node
}

type canvasEdgeV struct {
	ID       string `json:"id"`
	FromNode string `json:"fromNode"`
	ToNode   string `json:"toNode"`
	FromSide string `json:"fromSide,omitempty"`
	ToSide   string `json:"toSide,omitempty"`
	FromEnd  string `json:"fromEnd,omitempty"`
	ToEnd    string `json:"toEnd,omitempty"`
	Color    string `json:"color,omitempty"`
	Label    string `json:"label,omitempty"`
}

var validNodeTypes = map[string]bool{
	"text": true, "file": true, "link": true, "group": true,
}

var validSides = map[string]bool{
	"top": true, "right": true, "bottom": true, "left": true, "": true,
}

var validEndpoints = map[string]bool{
	"none": true, "arrow": true, "": true,
}

// ValidateCanvasContent validates JSON Canvas content.
func ValidateCanvasContent(content string) error {
	if len(content) > MaxCanvasContentSize {
		return fmt.Errorf("canvas content too large: %d bytes (max %d)", len(content), MaxCanvasContentSize)
	}

	// Allow empty canvas
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || trimmed == "{}" {
		return nil
	}

	var data canvasContent
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if len(data.Nodes) > MaxCanvasNodes {
		return fmt.Errorf("too many nodes: %d (max %d)", len(data.Nodes), MaxCanvasNodes)
	}
	if len(data.Edges) > MaxCanvasEdges {
		return fmt.Errorf("too many edges: %d (max %d)", len(data.Edges), MaxCanvasEdges)
	}

	nodeIDs, err := validateCanvasNodes(data.Nodes)
	if err != nil {
		return err
	}

	return validateCanvasEdges(data.Edges, nodeIDs)
}

// validateCanvasNodes validates all nodes and returns the set of node IDs for edge validation.
func validateCanvasNodes(nodes []canvasNodeV) (map[string]bool, error) {
	nodeIDs := make(map[string]bool, len(nodes))
	for i, node := range nodes {
		if node.ID == "" {
			return nil, fmt.Errorf("node[%d]: missing id", i)
		}
		if nodeIDs[node.ID] {
			return nil, fmt.Errorf("node[%d]: duplicate id %q", i, node.ID)
		}
		nodeIDs[node.ID] = true

		if !validNodeTypes[node.Type] {
			return nil, fmt.Errorf("node[%d] (%s): invalid type %q", i, node.ID, node.Type)
		}
		if node.X == nil || node.Y == nil {
			return nil, fmt.Errorf("node[%d] (%s): x and y are required", i, node.ID)
		}
		if node.Width == nil || node.Height == nil {
			return nil, fmt.Errorf("node[%d] (%s): width and height are required", i, node.ID)
		}
		if *node.Width < 1 || *node.Height < 1 {
			return nil, fmt.Errorf("node[%d] (%s): width and height must be >= 1", i, node.ID)
		}

		// Type-specific validation
		switch node.Type {
		case "text":
			// text field is optional (can be empty)
		case "file":
			if node.File == "" {
				return nil, fmt.Errorf("node[%d] (%s): file node requires 'file' field", i, node.ID)
			}
		case "link":
			if node.URL == "" {
				return nil, fmt.Errorf("node[%d] (%s): link node requires 'url' field", i, node.ID)
			}
		case "group":
			// label is optional
		}

		if err := validateCanvasColor(node.Color); err != nil {
			return nil, fmt.Errorf("node[%d] (%s): %w", i, node.ID, err)
		}
	}
	return nodeIDs, nil
}

// validateCanvasEdges validates all edges against the known nodeIDs set.
func validateCanvasEdges(edges []canvasEdgeV, nodeIDs map[string]bool) error {
	edgeIDs := make(map[string]bool, len(edges))
	for i, edge := range edges {
		if edge.ID == "" {
			return fmt.Errorf("edge[%d]: missing id", i)
		}
		if edgeIDs[edge.ID] {
			return fmt.Errorf("edge[%d]: duplicate id %q", i, edge.ID)
		}
		edgeIDs[edge.ID] = true

		if edge.FromNode == "" || edge.ToNode == "" {
			return fmt.Errorf("edge[%d] (%s): fromNode and toNode are required", i, edge.ID)
		}
		if !nodeIDs[edge.FromNode] {
			return fmt.Errorf("edge[%d] (%s): fromNode %q not found", i, edge.ID, edge.FromNode)
		}
		if !nodeIDs[edge.ToNode] {
			return fmt.Errorf("edge[%d] (%s): toNode %q not found", i, edge.ID, edge.ToNode)
		}
		if !validSides[edge.FromSide] {
			return fmt.Errorf("edge[%d] (%s): invalid fromSide %q", i, edge.ID, edge.FromSide)
		}
		if !validSides[edge.ToSide] {
			return fmt.Errorf("edge[%d] (%s): invalid toSide %q", i, edge.ID, edge.ToSide)
		}
		if !validEndpoints[edge.FromEnd] {
			return fmt.Errorf("edge[%d] (%s): invalid fromEnd %q", i, edge.ID, edge.FromEnd)
		}
		if !validEndpoints[edge.ToEnd] {
			return fmt.Errorf("edge[%d] (%s): invalid toEnd %q", i, edge.ID, edge.ToEnd)
		}
		if err := validateCanvasColor(edge.Color); err != nil {
			return fmt.Errorf("edge[%d] (%s): %w", i, edge.ID, err)
		}
	}
	return nil
}

// validateCanvasColor validates a JSON Canvas color value.
// Valid values: preset "1"-"6" or #RRGGBB hex color.
func validateCanvasColor(color string) error {
	if color == "" {
		return nil
	}
	// Preset colors: "1" through "6"
	if len(color) == 1 && color[0] >= '1' && color[0] <= '6' {
		return nil
	}
	// Hex color
	if hexColorRegex.MatchString(color) {
		return nil
	}
	return fmt.Errorf("invalid color %q (must be preset 1-6 or #RRGGBB)", color)
}
