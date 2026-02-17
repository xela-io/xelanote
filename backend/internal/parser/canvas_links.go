package parser

import (
	"encoding/json"
)

// canvasData represents the top-level JSON Canvas structure (for link extraction only).
type canvasData struct {
	Nodes []canvasNode `json:"nodes"`
}

type canvasNode struct {
	Type string `json:"type"`
	File string `json:"file"`
}

// ExtractCanvasFileRefs parses canvas JSON and returns the "file" field
// values from all nodes with type "file". These are note titles (not UUIDs).
// Image file paths (containing "/" or common image extensions) are excluded.
func ExtractCanvasFileRefs(canvasJSON string) ([]string, error) {
	var data canvasData
	if err := json.Unmarshal([]byte(canvasJSON), &data); err != nil {
		return nil, err
	}

	var refs []string
	seen := make(map[string]bool)
	for _, node := range data.Nodes {
		if node.Type != "file" || node.File == "" {
			continue
		}
		// Skip file paths that look like image uploads (contain "/")
		if isFilePath(node.File) {
			continue
		}
		if !seen[node.File] {
			seen[node.File] = true
			refs = append(refs, node.File)
		}
	}
	return refs, nil
}

// isFilePath returns true if the file reference looks like a file path (contains "/").
// Note titles in Xelanote never contain "/".
func isFilePath(ref string) bool {
	for _, c := range ref {
		if c == '/' {
			return true
		}
	}
	return false
}
