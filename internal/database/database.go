package database

import (
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/config"
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	var (
		db  *gorm.DB
		err error
	)

	switch cfg.DBDriver {
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DBDSN), &gorm.Config{})
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.DBDSN), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER: %s", cfg.DBDriver)
	}

	if err != nil {
		return nil, err
	}

	if err := configurePool(db, cfg); err != nil {
		return nil, err
	}

	if cfg.DBAutoMigrate {
		if err := autoMigrate(db); err != nil {
			return nil, err
		}
	}

	return db, nil
}

// configurePool 设置底层 *sql.DB 的连接池参数。
func configurePool(db *gorm.DB, cfg *config.Config) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}
	if cfg.DBMaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	}
	if cfg.DBConnMaxLifetimeMinutes > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeMinutes) * time.Minute)
	}
	return nil
}

// autoMigrate 集中迁移所有业务模型。生产环境建议关闭并使用独立迁移工具（如 golang-migrate）。
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.ProjectMember{},
		&model.ProjectTechLead{},
		&model.BoardColumn{},
		&model.Sprint{},
		&model.UserStory{},
		&model.BugReport{},
		&model.BugComment{},
		&model.Task{},
		&model.ActivityLog{},
		&model.TestCase{},
		&model.AIConfig{},
		&model.Notification{},
	)
}
