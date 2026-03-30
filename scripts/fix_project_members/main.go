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

	var projects []model.Project
	if err := db.Find(&projects).Error; err != nil {
		log.Fatal(err)
	}

	fixed := 0
	for _, project := range projects {
		var count int64
		db.Model(&model.ProjectMember{}).
			Where("project_id = ? AND user_id = ?", project.ID, project.OwnerID).
			Count(&count)

		if count == 0 {
			var user model.User
			if err := db.First(&user, project.OwnerID).Error; err != nil {
				log.Printf("项目 %d 的所有者 %d 不存在，跳过\n", project.ID, project.OwnerID)
				continue
			}

			member := model.ProjectMember{
				ProjectID:     project.ID,
				UserID:        project.OwnerID,
				RoleInProject: user.Role,
			}

			if err := db.Create(&member).Error; err != nil {
				log.Printf("添加成员失败: %v\n", err)
				continue
			}

			fmt.Printf("✓ 已将用户 %s 添加为项目 '%s' 的成员\n", user.Email, project.Name)
			fixed++
		}
	}

	fmt.Printf("\n修复完成！共修复 %d 个项目\n", fixed)
}
