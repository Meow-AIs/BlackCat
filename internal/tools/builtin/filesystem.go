package builtin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/meowai/blackcat/internal/tools"
)

// sensitivePathPrefixes are directory/file prefixes that should never be read
// and sent to an LLM. Populated by init() from the user's home directory.
var sensitivePathPrefixes []string

// sensitiveFileNames are exact filenames (no extension) that are always blocked.
var sensitiveFileNames = []string{
	"id_rsa",
	"id_ed25519",
	"id_ecdsa",
	"id_dsa",
}

// sensitiveExtensions are file extensions whose content must never be forwarded.
var sensitiveExtensions = []string{
	".pem",
	".key",
	".pfx",
	".p12",
}

func init() {
	home, _ := os.UserHomeDir()
	if home != "" {
		sensitivePathPrefixes = []string{
			filepath.Join(home, ".ssh"),
			filepath.Join(home, ".aws", "credentials"),
			filepath.Join(home, ".aws", "config"),
			filepath.Join(home, ".config", "gcloud"),
			filepath.Join(home, ".azure"),
			filepath.Join(home, ".docker", "config.json"),
			filepath.Join(home, ".kube", "config"),
			filepath.Join(home, ".gnupg"),
			filepath.Join(home, ".npmrc"),
			filepath.Join(home, ".pypirc"),
			filepath.Join(home, ".git-credentials"),
			filepath.Join(home, ".netrc"),
			filepath.Join(home, ".blackcat", "config.yaml"),
		}
	}
}

// isSensitivePath returns true when path resolves to a location that should
// never be forwarded to an LLM (credential files, private keys, config with
// secrets, etc.).  The path is cleaned/absolutised before checking so that
// traversal sequences like ../../.ssh/id_rsa are caught correctly.
func isSensitivePath(path string) bool {
	// Resolve to an absolute, clean path so traversal tricks don't bypass checks.
	abs, err := filepath.Abs(path)
	if err != nil {
		// If we cannot resolve it, err on the safe side.
		return true
	}
	cleaned := filepath.Clean(abs)

	// Check prefix denylist (covers entire subtrees, e.g. ~/.ssh/).
	for _, prefix := range sensitivePathPrefixes {
		// A path is sensitive if it equals the prefix exactly OR is nested inside it.
		if cleaned == prefix || strings.HasPrefix(cleaned, prefix+string(filepath.Separator)) {
			return true
		}
	}

	// Check exact filenames (basename without extension).
	base := filepath.Base(cleaned)
	nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
	for _, name := range sensitiveFileNames {
		if strings.EqualFold(nameWithoutExt, name) {
			return true
		}
	}

	// Check by extension.
	ext := strings.ToLower(filepath.Ext(cleaned))
	for _, sensitiveExt := range sensitiveExtensions {
		if ext == sensitiveExt {
			return true
		}
	}

	return false
}

func requireStringArg(args map[string]any, key string) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", fmt.Errorf("missing required argument %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("argument %q must be a string", key)
	}
	return s, nil
}

// --- ReadFileTool ---

type ReadFileTool struct{}

func NewReadFileTool() *ReadFileTool { return &ReadFileTool{} }

func (t *ReadFileTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "read_file",
		Description: "Read the contents of a file",
		Category:    "filesystem",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "File path to read", Required: true},
		},
	}
}

func (t *ReadFileTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	if isSensitivePath(path) {
		return tools.Result{
			Error:    fmt.Sprintf("access denied: %q is a sensitive path and may not be read", path),
			ExitCode: 1,
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: 1}, nil
	}
	return tools.Result{Output: string(data), ExitCode: 0}, nil
}

// --- WriteFileTool ---

type WriteFileTool struct{}

func NewWriteFileTool() *WriteFileTool { return &WriteFileTool{} }

func (t *WriteFileTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "write_file",
		Description: "Write content to a file, creating parent directories if needed",
		Category:    "filesystem",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "File path to write", Required: true},
			{Name: "content", Type: "string", Description: "Content to write", Required: true},
		},
	}
}

func (t *WriteFileTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}
	content, err := requireStringArg(args, "content")
	if err != nil {
		return tools.Result{}, err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return tools.Result{Error: err.Error(), ExitCode: 1}, nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return tools.Result{Error: err.Error(), ExitCode: 1}, nil
	}
	return tools.Result{Output: fmt.Sprintf("wrote %d bytes to %s", len(content), path), ExitCode: 0}, nil
}

// --- ListDirTool ---

type ListDirTool struct{}

func NewListDirTool() *ListDirTool { return &ListDirTool{} }

func (t *ListDirTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "list_dir",
		Description: "List files and directories in a path",
		Category:    "filesystem",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Directory path", Required: true},
		},
	}
}

func (t *ListDirTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: 1}, nil
	}

	var sb strings.Builder
	for _, e := range entries {
		kind := "file"
		if e.IsDir() {
			kind = "dir "
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		fmt.Fprintf(&sb, "%s  %8d  %s\n", kind, size, e.Name())
	}
	return tools.Result{Output: sb.String(), ExitCode: 0}, nil
}

// --- SearchFilesTool ---

type SearchFilesTool struct{}

func NewSearchFilesTool() *SearchFilesTool { return &SearchFilesTool{} }

func (t *SearchFilesTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "search_files",
		Description: "Search for files matching a glob pattern",
		Category:    "filesystem",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Root directory to search", Required: true},
			{Name: "pattern", Type: "string", Description: "Glob pattern (e.g., *.go)", Required: true},
		},
	}
}

func (t *SearchFilesTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	root, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}
	pattern, err := requireStringArg(args, "pattern")
	if err != nil {
		return tools.Result{}, err
	}

	var matches []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		matched, _ := filepath.Match(pattern, info.Name())
		if matched {
			rel, _ := filepath.Rel(root, path)
			matches = append(matches, rel)
		}
		return nil
	})

	return tools.Result{Output: strings.Join(matches, "\n"), ExitCode: 0}, nil
}

// --- SearchContentTool ---

type SearchContentTool struct{}

func NewSearchContentTool() *SearchContentTool { return &SearchContentTool{} }

func (t *SearchContentTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "search_content",
		Description: "Search file contents for a text pattern",
		Category:    "filesystem",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Root directory to search", Required: true},
			{Name: "pattern", Type: "string", Description: "Text pattern to search for", Required: true},
		},
	}
}

func (t *SearchContentTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	root, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}
	pattern, err := requireStringArg(args, "pattern")
	if err != nil {
		return tools.Result{}, err
	}

	var results []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if strings.Contains(line, pattern) {
				rel, _ := filepath.Rel(root, path)
				results = append(results, fmt.Sprintf("%s:%d: %s", rel, lineNum, line))
			}
		}
		return nil
	})

	if len(results) == 0 {
		return tools.Result{Output: "no matches found", ExitCode: 0}, nil
	}
	return tools.Result{Output: strings.Join(results, "\n"), ExitCode: 0}, nil
}
