package model

import "time"

// AIConfig AI 模型配置。
type AIConfig struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	APIKeyEncrypted string    `gorm:"type:text;not null" json:"-"`
	Model           string    `gorm:"size:100;not null" json:"model"`
	Temperature     float64   `gorm:"not null;default:0.7" json:"temperature"`
	MaxTokens       int       `gorm:"not null;default:4096" json:"max_tokens"`
	Enabled         bool      `gorm:"not null;default:true;index" json:"enabled"`
	UpdatedBy       uint      `gorm:"not null;index" json:"updated_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
