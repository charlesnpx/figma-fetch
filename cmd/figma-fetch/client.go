package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type figmaRef struct {
	OriginalURL string `json:"originalUrl"`
	FileKey     string `json:"fileKey"`
	NodeID      string `json:"nodeId,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Name        string `json:"name,omitempty"`
}

type figmaClient struct {
	token string
	base  string
	http  *http.Client
}

func newFigmaClient(token string) figmaClient {
	base := strings.TrimRight(os.Getenv("FIGMA_API_BASE"), "/")
	if base == "" {
		base = "https://api.figma.com/v1"
	}
	return figmaClient{token: token, base: base, http: &http.Client{Timeout: 60 * time.Second}}
}

func parseFigmaURL(raw string) (figmaRef, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return figmaRef{}, err
	}
	parts := splitPath(u.Path)
	if len(parts) < 2 {
		return figmaRef{}, errors.New("figma URL path does not contain a file key")
	}
	mode := parts[0]
	if mode != "file" && mode != "design" && mode != "proto" && mode != "board" {
		return figmaRef{}, errors.New("URL is not a supported Figma file/design/proto/board URL")
	}
	ref := figmaRef{OriginalURL: raw, FileKey: parts[1], Mode: mode}
	if len(parts) > 2 {
		ref.Name = parts[2]
	}
	if nodeID := u.Query().Get("node-id"); nodeID != "" {
		ref.NodeID = strings.ReplaceAll(nodeID, "-", ":")
	}
	if ref.FileKey == "" {
		return figmaRef{}, errors.New("figma file key is empty")
	}
	return ref, nil
}

func splitPath(path string) []string {
	raw := strings.Split(strings.Trim(path, "/"), "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func (c figmaClient) getJSON(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	endpoint := c.base + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Figma-Token", c.token)
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: %s: %s", endpoint, res.Status, strings.TrimSpace(string(body)))
	}
	return json.RawMessage(body), nil
}

func (c figmaClient) download(ctx context.Context, sourceURL, destination string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("download %s: %s", sourceURL, res.Status)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, res.Body)
	return err
}

func figmaFilePath(fileKey string) string {
	return "/files/" + url.PathEscape(fileKey)
}

func figmaNodesPath(fileKey string) string {
	return "/files/" + url.PathEscape(fileKey) + "/nodes"
}

func figmaImagesPath(fileKey string) string {
	return "/images/" + url.PathEscape(fileKey)
}
