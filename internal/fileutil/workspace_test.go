package fileutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWorkspacePath_Empty(t *testing.T) {
	root := t.TempDir()
	_, err := ResolveWorkspacePath(root, "")
	if !errors.Is(err, ErrPathEmpty) {
		t.Fatalf("expected ErrPathEmpty, got %v", err)
	}
	_, err = ResolveWorkspacePath(root, "   ")
	if !errors.Is(err, ErrPathEmpty) {
		t.Fatalf("expected ErrPathEmpty for whitespace, got %v", err)
	}
}

func TestResolveWorkspacePath_InsideRoot(t *testing.T) {
	root := t.TempDir()
	abs, err := ResolveWorkspacePath(root, "sub/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rootAbs, _ := filepath.Abs(root)
	want := filepath.Join(rootAbs, "sub", "file.txt")
	if abs != want {
		t.Errorf("expected %q, got %q", want, abs)
	}
}

func TestResolveWorkspacePath_BackslashNormalized(t *testing.T) {
	root := t.TempDir()
	abs, err := ResolveWorkspacePath(root, `sub\nested\file.txt`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rootAbs, _ := filepath.Abs(root)
	want := filepath.Join(rootAbs, "sub", "nested", "file.txt")
	if abs != want {
		t.Errorf("expected %q, got %q", want, abs)
	}
}

func TestResolveWorkspacePath_ParentEscape(t *testing.T) {
	root := t.TempDir()
	cases := []string{
		"../escape.txt",
		"../../escape.txt",
		"sub/../../escape.txt",
		"sub/../../../etc/passwd",
		"./../../escape.txt",
	}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			_, err := ResolveWorkspacePath(root, p)
			if !errors.Is(err, ErrPathOutsideWorkspace) {
				t.Fatalf("path %q should be rejected, got err=%v", p, err)
			}
		})
	}
}

func TestResolveWorkspacePath_AbsoluteOutsideRoot(t *testing.T) {
	root := t.TempDir()
	// Pick an absolute path guaranteed not under root.
	other := t.TempDir()
	outside := filepath.Join(other, "leak.txt")
	_, err := ResolveWorkspacePath(root, outside)
	if !errors.Is(err, ErrPathOutsideWorkspace) {
		t.Fatalf("absolute path outside root should be rejected, got err=%v", err)
	}
}

func TestResolveWorkspacePath_DefaultsToCwd(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	abs, err := ResolveWorkspacePath("", "file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(cwd, "file.txt")
	if abs != want {
		t.Errorf("expected default to cwd: want %q got %q", want, abs)
	}
}

func TestResolveWorkspacePath_DotInsideRoot(t *testing.T) {
	root := t.TempDir()
	// "sub/./file.txt" stays inside root and should resolve normally.
	abs, err := ResolveWorkspacePath(root, "sub/./file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rootAbs, _ := filepath.Abs(root)
	if !strings.HasPrefix(abs, rootAbs) {
		t.Errorf("resolved path %q should remain under root %q", abs, rootAbs)
	}
}

func TestReadWorkspaceFile_HappyAndEscape(t *testing.T) {
	root := t.TempDir()
	// Create a real file inside root.
	target := filepath.Join(root, "data.txt")
	if err := os.WriteFile(target, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	abs, content, err := ReadWorkspaceFile(root, "data.txt")
	if err != nil {
		t.Fatalf("read inside root failed: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("expected hello, got %q", content)
	}
	if !strings.HasPrefix(abs, root) {
		t.Errorf("expected resolved path under root, got %q", abs)
	}

	// Escape attempt should fail at resolution, never reaching ReadFile.
	_, _, err = ReadWorkspaceFile(root, "../escape.txt")
	if !errors.Is(err, ErrPathOutsideWorkspace) {
		t.Errorf("expected ErrPathOutsideWorkspace, got %v", err)
	}
}
