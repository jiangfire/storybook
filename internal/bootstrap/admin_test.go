package bootstrap

import (
	"testing"

	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestEnsureAdminCreatesNewUser(t *testing.T) {
	db := openBootstrapTestDB(t)

	result, err := EnsureAdmin(db, EnsureAdminParams{
		Email:    "admin@example.com",
		Password: "Admin1234",
	})
	if err != nil {
		t.Fatalf("EnsureAdmin returned error: %v", err)
	}
	if !result.Created {
		t.Fatalf("expected Created=true")
	}

	var user model.User
	if err := db.Where("email = ?", "admin@example.com").First(&user).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Role != model.RoleAdmin {
		t.Fatalf("expected role admin, got %s", user.Role)
	}
	if user.Username != "admin" {
		t.Fatalf("expected generated username admin, got %s", user.Username)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte("Admin1234")); err != nil {
		t.Fatalf("password was not hashed correctly: %v", err)
	}
}

func TestEnsureAdminPromotesExistingUser(t *testing.T) {
	db := openBootstrapTestDB(t)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("User1234"), 10)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := model.User{
		Username:       "tester",
		Email:          "tester@example.com",
		HashedPassword: string(hashedPassword),
		Role:           model.RoleDeveloper,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	result, err := EnsureAdmin(db, EnsureAdminParams{
		Email:    "tester@example.com",
		Password: "Admin5678",
	})
	if err != nil {
		t.Fatalf("EnsureAdmin returned error: %v", err)
	}
	if result.Created {
		t.Fatalf("expected Created=false")
	}
	if !result.RoleChanged {
		t.Fatalf("expected RoleChanged=true")
	}
	if !result.PasswordChanged {
		t.Fatalf("expected PasswordChanged=true")
	}

	var refreshed model.User
	if err := db.First(&refreshed, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if refreshed.Role != model.RoleAdmin {
		t.Fatalf("expected role admin, got %s", refreshed.Role)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(refreshed.HashedPassword), []byte("Admin5678")); err != nil {
		t.Fatalf("password was not updated: %v", err)
	}
}

func openBootstrapTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:bootstrap_admin_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}
