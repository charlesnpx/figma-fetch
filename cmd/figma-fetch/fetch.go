package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type fetchOptions struct {
	url      string
	nodeID   string
	outDir   string
	cacheDir string
	noCache  bool
	render   string
	token    string
}

func fetchAndWrite(opts fetchOptions) error {
	ref, err := parseFigmaURL(opts.url)
	if err != nil {
		return err
	}
	if opts.nodeID != "" {
		ref.NodeID = strings.ReplaceAll(opts.nodeID, "-", ":")
	}
	if opts.cacheDir == "" {
		opts.cacheDir, err = defaultCacheDir("figma-fetch")
		if err != nil {
			return err
		}
	}
	if opts.outDir == "" {
		opts.outDir = defaultOutDir(opts.cacheDir, ref)
	}
	if opts.token == "" {
		opts.token = os.Getenv("FIGMA_TOKEN")
	}
	if opts.token == "" {
		return fmt.Errorf("missing Figma token; set FIGMA_TOKEN or pass --token")
	}
	if opts.render != "" && ref.NodeID == "" {
		return fmt.Errorf("--render requires a selected node or --node")
	}

	ctx := context.Background()
	client := newFigmaClient(opts.token)
	raw, err := fetchRaw(ctx, client, ref, opts)
	if err != nil {
		return err
	}
	nodes, err := extractFigmaNodes(raw)
	if err != nil {
		return err
	}
	assets := []figmaAsset{}
	if opts.render != "" {
		asset, err := renderNode(ctx, client, ref, opts)
		if err != nil {
			return err
		}
		assets = append(assets, asset)
	}
	if err := writeOutput(opts.outDir, ref, raw, nodes, assets); err != nil {
		return err
	}
	fmt.Println(opts.outDir)
	return nil
}

func fetchRaw(ctx context.Context, client figmaClient, ref figmaRef, opts fetchOptions) (json.RawMessage, error) {
	cachePath := filepath.Join(opts.cacheDir, ref.FileKey, cacheKey(map[string]string{
		"kind":    "nodes",
		"fileKey": ref.FileKey,
		"nodeID":  ref.NodeID,
	})+".json")
	if !opts.noCache {
		if body, err := os.ReadFile(cachePath); err == nil {
			return json.RawMessage(body), nil
		}
	}
	var (
		raw json.RawMessage
		err error
	)
	if ref.NodeID != "" {
		raw, err = client.getJSON(ctx, figmaNodesPath(ref.FileKey), url.Values{"ids": []string{ref.NodeID}})
	} else {
		raw, err = client.getJSON(ctx, figmaFilePath(ref.FileKey), nil)
	}
	if err != nil {
		return nil, err
	}
	if !opts.noCache {
		if err := writeRawJSON(cachePath, raw); err != nil {
			return nil, err
		}
	}
	return raw, nil
}

func cacheKey(values map[string]string) string {
	keys := sortedMapKeys(values)
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(values[key])
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])[:16]
}

func defaultOutDir(cacheDir string, ref figmaRef) string {
	node := "root"
	if ref.NodeID != "" {
		node = safeID(ref.NodeID)
	}
	return filepath.Join(cacheDir, "outputs", safeID(ref.FileKey), node)
}

func defaultCacheDir(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", name), nil
}
