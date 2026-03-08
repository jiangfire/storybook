package fileutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrPathOutsideWorkspace = errors.New("path outside workspace")
	ErrPathEmpty            = errors.New("path is empty")
)

func ReadWorkspaceFile(workspaceRoot, inputPath string) (string, []byte, error) {
	absPath, err := ResolveWorkspacePath(workspaceRoot, inputPath)
	if err != nil {
		return "", nil, err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", nil, err
	}
	return absPath, content, nil
}

func ResolveWorkspacePath(workspaceRoot, inputPath string) (string, error) {
	path := strings.TrimSpace(inputPath)
	if path == "" {
		return "", ErrPathEmpty
	}

	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		root = cwd
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	cleaned := filepath.Clean(path)
	absPath := cleaned
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(rootAbs, cleaned)
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(rootAbs, absPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathOutsideWorkspace
	}

	return absPath, nil
}
