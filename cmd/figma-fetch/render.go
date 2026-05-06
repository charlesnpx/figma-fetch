package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

func renderNode(ctx context.Context, client figmaClient, ref figmaRef, opts fetchOptions) (figmaAsset, error) {
	if !validRenderFormat(opts.render) {
		return figmaAsset{}, fmt.Errorf("--render must be png, svg, pdf, or jpg")
	}
	fileName := safeID(ref.NodeID) + "." + opts.render
	assetRel := filepath.Join("assets", fileName)
	outPath := filepath.Join(opts.outDir, assetRel)
	cachePath := filepath.Join(opts.cacheDir, ref.FileKey, cacheKey(map[string]string{
		"kind":    "render",
		"fileKey": ref.FileKey,
		"nodeID":  ref.NodeID,
		"format":  opts.render,
	})+"."+opts.render)

	if !opts.noCache {
		if err := copyFile(cachePath, outPath); err == nil {
			return figmaAsset{Kind: "rendered-node", SourceID: ref.NodeID, AssetPath: filepath.ToSlash(assetRel)}, nil
		}
	}

	raw, err := client.getJSON(ctx, figmaImagesPath(ref.FileKey), url.Values{
		"ids":    []string{ref.NodeID},
		"format": []string{opts.render},
	})
	if err != nil {
		return figmaAsset{}, err
	}
	var response struct {
		Images map[string]string `json:"images"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return figmaAsset{}, err
	}
	imageURL := response.Images[ref.NodeID]
	if imageURL == "" {
		return figmaAsset{}, fmt.Errorf("Figma did not return a render URL for node %s", ref.NodeID)
	}
	downloadPath := outPath
	if !opts.noCache {
		downloadPath = cachePath
	}
	if err := client.download(ctx, imageURL, downloadPath); err != nil {
		return figmaAsset{}, err
	}
	if !opts.noCache {
		if err := copyFile(cachePath, outPath); err != nil {
			return figmaAsset{}, err
		}
	}
	return figmaAsset{Kind: "rendered-node", SourceID: ref.NodeID, AssetPath: filepath.ToSlash(assetRel)}, nil
}

func copyFile(src, dst string) error {
	body, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, body, 0o644)
}

func validRenderFormat(format string) bool {
	switch format {
	case "png", "svg", "pdf", "jpg":
		return true
	default:
		return false
	}
}
