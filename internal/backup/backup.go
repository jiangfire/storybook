// Package backup 提供数据库备份能力：SQLite 走在线安全的 VACUUM INTO 快照，
// PostgreSQL 委托 pg_dump 子进程。两者都不允许覆盖已存在的目标文件，
// 避免误写坏既有备份。
package backup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Run 按驱动分发备份：driver 为 "sqlite" 或 "postgres"。
// sqlite 分支使用 db 连接执行快照；postgres 分支只依赖 dsn（pg_dump 子进程）。
func Run(db *gorm.DB, driver, dsn, dest string) error {
	if strings.TrimSpace(dest) == "" {
		return fmt.Errorf("目标文件路径不能为空")
	}
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("目标文件已存在，拒绝覆盖: %s", dest)
	}

	switch driver {
	case "sqlite":
		return runSQLite(db, dest)
	case "postgres":
		return runPostgres(dsn, dest)
	default:
		return fmt.Errorf("不支持的数据库驱动: %s（支持 sqlite/postgres）", driver)
	}
}

// DefaultDest 返回默认备份文件名（含时间戳，避免覆盖）。
func DefaultDest(driver string) string {
	stamp := time.Now().Format("20060102-150405")
	if driver == "postgres" {
		return fmt.Sprintf("storybook-backup-%s.sql", stamp)
	}
	return fmt.Sprintf("storybook-backup-%s.db", stamp)
}

func runSQLite(db *gorm.DB, dest string) error {
	if db == nil {
		return fmt.Errorf("数据库连接为空")
	}
	// VACUUM INTO 生成在线一致性快照（服务运行中也可备份），
	// 路径为 SQL 字面量，单引号需翻倍转义。
	escaped := strings.ReplaceAll(dest, "'", "''")
	if err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", escaped)).Error; err != nil {
		return fmt.Errorf("sqlite 快照失败: %w", err)
	}
	return nil
}

func runPostgres(dsn, dest string) error {
	pgDump, err := exec.LookPath("pg_dump")
	if err != nil {
		return fmt.Errorf("未找到 pg_dump，请安装 PostgreSQL 客户端工具后重试")
	}
	cmd := exec.Command(pgDump, "--dbname="+dsn, "--file="+dest)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pg_dump 执行失败: %v（输出: %s）", err, strings.TrimSpace(string(output)))
	}
	if _, err := os.Stat(dest); err != nil {
		return fmt.Errorf("pg_dump 未生成目标文件: %s", dest)
	}
	return nil
}
