package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type figmaNode struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name,omitempty"`
	Type     string         `json:"type,omitempty"`
	Text     string         `json:"text,omitempty"`
	Bounds   any            `json:"bounds,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Children []figmaNode    `json:"children,omitempty"`
}

type figmaAsset struct {
	Kind      string `json:"kind"`
	SourceID  string `json:"sourceId,omitempty"`
	AssetPath string `json:"assetPath"`
}

type figmaContent struct {
	Reference figmaRef     `json:"reference"`
	Nodes     []figmaNode  `json:"nodes"`
	Assets    []figmaAsset `json:"assets,omitempty"`
}

func extractFigmaNodes(raw json.RawMessage) ([]figmaNode, error) {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	var roots []figmaNode
	collectFigmaRoots(data, &roots)
	return roots, nil
}

func collectFigmaRoots(value any, roots *[]figmaNode) {
	switch typed := value.(type) {
	case map[string]any:
		if doc, ok := typed["document"]; ok {
			*roots = append(*roots, figmaNodeFrom(doc))
			return
		}
		if nodes, ok := typed["nodes"].(map[string]any); ok {
			keys := sortedMapKeys(nodes)
			for _, key := range keys {
				if nodeMap, ok := nodes[key].(map[string]any); ok {
					if doc, ok := nodeMap["document"]; ok {
						*roots = append(*roots, figmaNodeFrom(doc))
					}
				}
			}
			return
		}
		for _, child := range typed {
			collectFigmaRoots(child, roots)
		}
	case []any:
		for _, child := range typed {
			collectFigmaRoots(child, roots)
		}
	}
}

func figmaNodeFrom(value any) figmaNode {
	nodeMap, _ := value.(map[string]any)
	node := figmaNode{
		ID:     stringValue(nodeMap["id"]),
		Name:   stringValue(nodeMap["name"]),
		Type:   stringValue(nodeMap["type"]),
		Text:   stringValue(nodeMap["characters"]),
		Bounds: firstPresent(nodeMap, "absoluteBoundingBox", "absoluteRenderBounds"),
	}
	node.Metadata = figmaMetadata(nodeMap)
	for _, child := range anySlice(nodeMap["children"]) {
		node.Children = append(node.Children, figmaNodeFrom(child))
	}
	return node
}

func figmaMetadata(nodeMap map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"connectorStart", "connectorEnd", "connectorLineType", "tableCellProperties"} {
		if value, ok := nodeMap[key]; ok {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func writeOutput(outDir string, ref figmaRef, raw json.RawMessage, nodes []figmaNode, assets []figmaAsset) error {
	if err := writeRawJSON(filepath.Join(outDir, "raw", "nodes.json"), raw); err != nil {
		return err
	}
	content := figmaContent{Reference: ref, Nodes: nodes, Assets: assets}
	if err := writeJSON(filepath.Join(outDir, "content.json"), content); err != nil {
		return err
	}
	return writeContentMarkdown(filepath.Join(outDir, "content.md"), content)
}

func writeContentMarkdown(path string, content figmaContent) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Figma Extract\n\n")
	fmt.Fprintf(&b, "- File key: `%s`\n", content.Reference.FileKey)
	if content.Reference.NodeID != "" {
		fmt.Fprintf(&b, "- Node: `%s`\n", content.Reference.NodeID)
	}
	fmt.Fprintf(&b, "- Roots: %d\n", len(content.Nodes))
	if len(content.Assets) > 0 {
		fmt.Fprintf(&b, "- Assets: %d\n", len(content.Assets))
	}
	b.WriteString("\n")
	for _, node := range content.Nodes {
		writeNodeMarkdown(&b, node, 0)
	}
	if len(content.Assets) > 0 {
		b.WriteString("\n## Assets\n\n")
		for _, asset := range content.Assets {
			fmt.Fprintf(&b, "- `%s` from `%s`\n", asset.AssetPath, asset.SourceID)
		}
	}
	return writeFile(path, []byte(b.String()))
}

func writeNodeMarkdown(b *strings.Builder, node figmaNode, depth int) {
	indent := strings.Repeat("  ", depth)
	label := firstNonEmpty(node.Name, node.ID, "(unnamed)")
	if node.Type != "" {
		fmt.Fprintf(b, "%s- %s `%s`", indent, label, node.Type)
	} else {
		fmt.Fprintf(b, "%s- %s", indent, label)
	}
	if node.Text != "" {
		fmt.Fprintf(b, ": %q", node.Text)
	}
	b.WriteByte('\n')
	for _, child := range node.Children {
		writeNodeMarkdown(b, child, depth+1)
	}
}

func writeJSON(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, append(body, '\n'))
}

func writeRawJSON(path string, raw json.RawMessage) error {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	return writeFile(path, append([]byte(strings.TrimSpace(string(raw))), '\n'))
}

func writeFile(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}
