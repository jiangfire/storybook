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

	// 检查用户 1 的项目成员关系
	var members []model.ProjectMember
	if err := db.Preload("Project").Where("user_id = ?", 1).Find(&members).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Printf("用户 1 (test@test.com) 是以下项目的成员:\n")
	if len(members) == 0 {
		fmt.Println("  (无)")
	}
	for _, m := range members {
		fmt.Printf("  - 项目 ID: %d, 名称: %s\n", m.ProjectID, m.Project.Name)
	}
	fmt.Println()

	// 检查所有项目
	var projects []model.Project
	if err := db.Find(&projects).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Println("所有项目:")
	for _, p := range projects {
		fmt.Printf("  - ID: %d, 名称: %s, 所有者ID: %d\n", p.ID, p.Name, p.OwnerID)
	}
}
