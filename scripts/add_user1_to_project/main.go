package main

import (
	"fmt"
	"log"

	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("storybook.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 将用户 1 添加到项目 2
	member := model.ProjectMember{
		ProjectID:     2,
		UserID:        1,
		RoleInProject: "product",
	}

	if err := db.Create(&member).Error; err != nil {
		log.Fatal("添加成员失败:", err)
	}

	fmt.Println("✓ 已将 test@test.com 添加为项目 2 的成员")
}
