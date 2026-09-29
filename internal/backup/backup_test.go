package backup

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

func seedUser(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user := model.User{Username: "u1", Email: "u1@test.dev", HashedPassword: "x", Role: model.RoleAdmin}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// closeDB 注册测试结束时释放 SQLite 文件句柄（Windows 上不释放会导致 TempDir 清理失败）。
func closeDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestRunSQLiteProducesRestorableSnapshot(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.db")

	db, err := gorm.Open(sqlite.Open(src), &gorm.Config{})
	if err != nil {
		t.Fatalf("open source db: %v", err)
	}
	closeDB(t, db)
	seedUser(t, db)

	dest := filepath.Join(dir, "backup.db")
	if err := Run(db, "sqlite", src, dest); err != nil {
		t.Fatalf("backup: %v", err)
	}

	// 恢复验证：备份文件必须能作为独立库打开并查到数据
	restored, err := gorm.Open(sqlite.Open(dest), &gorm.Config{})
	if err != nil {
		t.Fatalf("open restored db: %v", err)
	}
	var user model.User
	if err := restored.First(&user).Error; err != nil {
		t.Fatalf("query restored user: %v", err)
	}
	if user.Email != "u1@test.dev" {
		t.Fatalf("unexpected restored email: %s", user.Email)
	}
	closeDB(t, restored)
}

func TestRunSQLiteRefusesToOverwriteExistingFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.db")

	db, err := gorm.Open(sqlite.Open(src), &gorm.Config{})
	if err != nil {
		t.Fatalf("open source db: %v", err)
	}
	closeDB(t, db)
	seedUser(t, db)

	dest := filepath.Join(dir, "backup.db")
	if err := Run(db, "sqlite", src, dest); err != nil {
		t.Fatalf("first backup: %v", err)
	}
	err = Run(db, "sqlite", src, dest)
	if err == nil {
		t.Fatal("expected error when destination already exists")
	}
	if !strings.Contains(err.Error(), "已存在") {
		t.Fatalf("expected explicit overwrite refusal, got: %v", err)
	}
}

func TestRunPostgresWithoutPgDumpFailsWithClearError(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "backup.sql")

	// db 句柄对 postgres 分支不参与（走 pg_dump 子进程），传 nil 即可
	err := Run(nil, "postgres", "postgres://127.0.0.1:1/none", dest)
	if err == nil {
		t.Fatal("expected error for postgres backup without reachable pg_dump")
	}
	if !strings.Contains(err.Error(), "pg_dump") {
		t.Fatalf("error should mention pg_dump, got: %v", err)
	}
}
