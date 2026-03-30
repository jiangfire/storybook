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

	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Println("所有用户:")
	for _, u := range users {
		fmt.Printf("  ID: %d, 邮箱: %s, 角色: %s\n", u.ID, u.Email, u.Role)
	}
}
