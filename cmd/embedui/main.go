package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fail("定位仓库根目录失败: %v", err)
	}

	srcDir := filepath.Join(repoRoot, "storybook-page", "dist")
	dstDir := filepath.Join(repoRoot, "internal", "webui", "dist")

	if _, err := os.Stat(filepath.Join(srcDir, "index.html")); err != nil {
		fail("未找到前端构建产物，请先执行 `cd storybook-page && pnpm run build`")
	}

	if err := os.MkdirAll(dstDir, 0o750); err != nil {
		fail("创建目标目录失败: %v", err)
	}

	if err := cleanDestination(dstDir); err != nil {
		fail("清理目标目录失败: %v", err)
	}

	if err := copyDir(srcDir, dstDir); err != nil {
		fail("同步前端产物失败: %v", err)
	}

	fmt.Printf("frontend embed synced: %s -> %s\n", srcDir, dstDir)
}

func cleanDestination(dstDir string) error {
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		// 保留跟踪文件，避免每次同步都污染工作区。
		if name == ".gitignore" || name == "index.html" {
			continue
		}

		if err := os.RemoveAll(filepath.Join(dstDir, name)); err != nil {
			return err
		}
	}
	return nil
}

func copyDir(srcDir, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) (err error) {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dstDir, relPath)
		if relPath == "index.html" {
			targetPath = filepath.Join(dstDir, "index.generated.html")
		}

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o750)
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			closeErr := srcFile.Close()
			if err == nil && closeErr != nil {
				err = closeErr
			}
		}()

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
			return err
		}

		dstFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			_ = dstFile.Close()
			return err
		}

		if err := dstFile.Close(); err != nil {
			return err
		}

		return nil
	})
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := wd
	for {
		if looksLikeRepoRoot(dir) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("未找到包含 go.mod 的仓库根目录")
		}
		dir = parent
	}
}

func looksLikeRepoRoot(dir string) bool {
	if !fileExists(filepath.Join(dir, "go.mod")) {
		return false
	}
	if !dirExists(filepath.Join(dir, "storybook-page")) {
		return false
	}
	if !dirExists(filepath.Join(dir, "internal", "webui")) {
		return false
	}
	return true
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
