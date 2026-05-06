package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("missing argument")
	}
	switch args[0] {
	case "install-skill":
		return runInstaller(args[1:])
	case "--version", "-version", "version":
		fmt.Println(version)
		return nil
	case "-h", "--help", "help":
		usage()
		return nil
	}

	fs := flag.NewFlagSet("figma-fetch", flag.ContinueOnError)
	nodeID := fs.String("node", "", "Override node ID")
	outDir := fs.String("out", "", "Output directory")
	cacheDir := fs.String("cache-dir", "", "Cache directory")
	noCache := fs.Bool("no-cache", false, "Bypass cache")
	render := fs.String("render", "", "Render format (png|svg|pdf|jpg)")
	token := fs.String("token", "", "Figma PAT")
	flagArgs, target, err := splitArgs(args, map[string]bool{
		"node": true, "out": true, "cache-dir": true, "render": true, "token": true,
	})
	if err != nil {
		return err
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if target == "" {
		return errors.New("expected exactly one URL argument")
	}
	return fetchAndWrite(fetchOptions{
		url: target, nodeID: *nodeID, outDir: *outDir, cacheDir: *cacheDir,
		noCache: *noCache, render: *render, token: *token,
	})
}

func runInstaller(args []string) error {
	script, err := installerPath()
	if err != nil {
		return err
	}
	cmd := exec.Command(script, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func installerPath() (string, error) {
	candidates := []string{}
	if _, file, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(file), "..", "..", "install-skill.sh"))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "install-skill.sh"))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "install-skill.sh"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("install-skill.sh not found next to source or binary")
}

func splitArgs(args []string, valueFlags map[string]bool) ([]string, string, error) {
	flags := []string{}
	target := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if i+2 != len(args) || target != "" {
				return nil, "", errors.New("expected exactly one URL argument")
			}
			target = args[i+1]
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flags = append(flags, arg)
			name := strings.TrimLeft(arg, "-")
			name, _, hasInline := strings.Cut(name, "=")
			if valueFlags[name] && !hasInline {
				if i+1 >= len(args) {
					return nil, "", fmt.Errorf("flag needs an argument: %s", arg)
				}
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		if target != "" {
			return nil, "", errors.New("expected exactly one URL argument")
		}
		target = arg
	}
	return flags, target, nil
}

func usage() {
	fmt.Println(`Usage:
  figma-fetch <url> [--node <id>] [--out <dir>] [--cache-dir <dir>] [--no-cache] [--render png|svg|pdf|jpg] [--token <pat>]
  figma-fetch install-skill [--plan|--install|--uninstall] [--target all|claude|codex|tools] [--json] [--install-root <dir>]
  figma-fetch --version | --help`)
}

func safeID(value string) string {
	return strings.NewReplacer(":", "-", ";", "-", "/", "-", "\\", "-", "=", "-").Replace(value)
}

func sortedMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func anySlice(value any) []any {
	if values, ok := value.([]any); ok {
		return values
	}
	return nil
}

func firstPresent(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
