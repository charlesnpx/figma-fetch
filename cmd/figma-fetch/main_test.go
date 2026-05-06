package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseFigmaURL(t *testing.T) {
	ref, err := parseFigmaURL("https://www.figma.com/board/AbCdEf/Board?node-id=12-34")
	if err != nil {
		t.Fatal(err)
	}
	if ref.FileKey != "AbCdEf" || ref.NodeID != "12:34" || ref.Mode != "board" {
		t.Fatalf("ref = %+v", ref)
	}
}

func TestFetchWritesOutputAndUsesCache(t *testing.T) {
	tmp := t.TempDir()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/files/file123/nodes" || r.URL.Query().Get("ids") != "1:2" {
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"nodes": map[string]any{"1:2": map[string]any{"document": map[string]any{
			"id": "1:2", "name": "Frame", "type": "FRAME", "children": []any{map[string]any{"id": "3:4", "type": "TEXT", "characters": "Hello"}},
		}}}})
	}))
	defer server.Close()
	t.Setenv("FIGMA_API_BASE", server.URL)

	opts := fetchOptions{
		url:      "https://www.figma.com/design/file123/Name?node-id=1-2",
		outDir:   filepath.Join(tmp, "out"),
		cacheDir: filepath.Join(tmp, "cache"),
		token:    "token",
	}
	if err := fetchAndWrite(opts); err != nil {
		t.Fatal(err)
	}
	if err := fetchAndWrite(opts); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("api calls = %d, want 1", calls)
	}
	for _, rel := range []string{"content.md", "content.json", "raw/nodes.json"} {
		if _, err := os.Stat(filepath.Join(opts.outDir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestRenderWritesAsset(t *testing.T) {
	tmp := t.TempDir()
	var imageURL string
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("png"))
	}))
	defer imageServer.Close()
	imageURL = imageServer.URL + "/image.png"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files/file123/nodes":
			_, _ = w.Write([]byte(`{"nodes":{"1:2":{"document":{"id":"1:2","name":"Frame","type":"FRAME"}}}}`))
		case "/images/file123":
			_ = json.NewEncoder(w).Encode(map[string]any{"images": map[string]string{"1:2": imageURL}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("FIGMA_API_BASE", server.URL)

	out := filepath.Join(tmp, "out")
	err := fetchAndWrite(fetchOptions{
		url: "https://www.figma.com/design/file123/Name?node-id=1-2", outDir: out,
		cacheDir: filepath.Join(tmp, "cache"), token: "token", render: "png",
	})
	if err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(out, "assets", "1-2.png")); err != nil || string(body) != "png" {
		t.Fatalf("asset body = %q err=%v", string(body), err)
	}
}

func TestSplitArgsAcceptsURLBeforeFlags(t *testing.T) {
	flags, target, err := splitArgs([]string{"url", "--out", "ctx", "--no-cache"}, map[string]bool{"out": true})
	if err != nil {
		t.Fatal(err)
	}
	if target != "url" || len(flags) != 3 {
		t.Fatalf("target=%q flags=%v", target, flags)
	}
}
