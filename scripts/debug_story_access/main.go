package main

import (
	"fmt"
	"log"

	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("storybook.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 查询故事 1
	var story model.UserStory
	if err := db.Preload("Project").First(&story, 1).Error; err != nil {
		log.Fatal("故事不存在:", err)
	}

	fmt.Printf("故事 ID: %d\n", story.ID)
	fmt.Printf("故事标题: %s\n", story.Title)
	fmt.Printf("项目 ID: %d\n", story.ProjectID)
	fmt.Printf("项目名称: %s\n", story.Project.Name)
	fmt.Printf("项目所有者 ID: %d\n", story.Project.OwnerID)
	fmt.Println()

	// 查询项目成员
	var members []model.ProjectMember
	if err := db.Preload("User").Where("project_id = ?", story.ProjectID).Find(&members).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Printf("项目成员列表 (共 %d 人):\n", len(members))
	for _, m := range members {
		fmt.Printf("  - 用户 ID: %d, 邮箱: %s, 角色: %s\n", m.UserID, m.User.Email, m.RoleInProject)
	}
}
