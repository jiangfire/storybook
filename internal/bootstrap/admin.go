package bootstrap

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/jiangfire/storybook/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type EnsureAdminParams struct {
	Email    string
	Username string
	Password string
}

type EnsureAdminResult struct {
	UserID          uint
	Email           string
	Username        string
	Created         bool
	RoleChanged     bool
	PasswordChanged bool
}

func EnsureAdmin(db *gorm.DB, params EnsureAdminParams) (*EnsureAdminResult, error) {
	email := strings.ToLower(strings.TrimSpace(params.Email))
	if email == "" {
		return nil, fmt.Errorf("email 不能为空")
	}

	username := strings.TrimSpace(params.Username)
	password := strings.TrimSpace(params.Password)

	var user model.User
	err := db.Where("email = ?", email).First(&user).Error
	switch {
	case err == nil:
		return ensureExistingAdmin(db, &user, username, password)
	case err != nil && err != gorm.ErrRecordNotFound:
		return nil, err
	}

	if password == "" {
		return nil, fmt.Errorf("创建新管理员时 password 必填")
	}
	if !isStrongPassword(password) {
		return nil, fmt.Errorf("password 至少 8 位，且必须包含字母和数字")
	}

	finalUsername, err := resolveUsername(db, username, email, 0)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, err
	}

	user = model.User{
		Username:       finalUsername,
		Email:          email,
		HashedPassword: string(hashedPassword),
		Role:           model.RoleAdmin,
	}
	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &EnsureAdminResult{
		UserID:          user.ID,
		Email:           user.Email,
		Username:        user.Username,
		Created:         true,
		RoleChanged:     true,
		PasswordChanged: true,
	}, nil
}

func ensureExistingAdmin(db *gorm.DB, user *model.User, username, password string) (*EnsureAdminResult, error) {
	updates := map[string]any{}
	result := &EnsureAdminResult{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
	}

	if user.Role != model.RoleAdmin {
		updates["role"] = model.RoleAdmin
		result.RoleChanged = true
	}

	if username != "" {
		finalUsername, err := resolveUsername(db, username, user.Email, user.ID)
		if err != nil {
			return nil, err
		}
		if finalUsername != user.Username {
			updates["username"] = finalUsername
			result.Username = finalUsername
		}
	}

	if password != "" {
		if !isStrongPassword(password) {
			return nil, fmt.Errorf("password 至少 8 位，且必须包含字母和数字")
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			return nil, err
		}
		updates["hashed_password"] = string(hashedPassword)
		result.PasswordChanged = true
	}

	if len(updates) > 0 {
		if err := db.Model(user).Updates(updates).Error; err != nil {
			return nil, err
		}
		if role, ok := updates["role"].(string); ok {
			user.Role = role
		}
		if newUsername, ok := updates["username"].(string); ok {
			user.Username = newUsername
			result.Username = newUsername
		}
	}

	return result, nil
}

func resolveUsername(db *gorm.DB, requestedUsername, email string, excludeUserID uint) (string, error) {
	base := strings.TrimSpace(requestedUsername)
	if base == "" {
		base = usernameFromEmail(email)
	}
	if base == "" {
		return "", fmt.Errorf("无法生成用户名")
	}

	candidate := base
	for i := 0; i < 1000; i++ {
		var exists int64
		query := db.Model(&model.User{}).Where("username = ?", candidate)
		if excludeUserID > 0 {
			query = query.Where("id <> ?", excludeUserID)
		}
		if err := query.Count(&exists).Error; err != nil {
			return "", err
		}
		if exists == 0 {
			return candidate, nil
		}
		if requestedUsername != "" {
			return "", fmt.Errorf("username 已被占用")
		}
		candidate = fmt.Sprintf("%s_%d", base, i+2)
	}

	return "", fmt.Errorf("无法找到可用用户名")
}

func usernameFromEmail(email string) string {
	local := strings.TrimSpace(strings.Split(email, "@")[0])
	if local == "" {
		return ""
	}

	var builder strings.Builder
	for _, r := range local {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
		case r == '.', r == '-', r == '_':
			builder.WriteRune(r)
		}
	}

	username := strings.Trim(builder.String(), "._-")
	if username == "" {
		return "admin"
	}
	return username
}

func isStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasLetter := false
	hasDigit := false
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}
