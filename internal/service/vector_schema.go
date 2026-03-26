package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

var vectorTypePattern = regexp.MustCompile(`^vector\((\d+)\)$`)

func DetectStoryEmbeddingDimension(db *gorm.DB) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database is required")
	}

	var formatType string
	err := db.Raw(`
		SELECT pg_catalog.format_type(a.atttypid, a.atttypmod)
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
		  AND c.relname = 'user_stories'
		  AND a.attname = 'embedding'
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		LIMIT 1
	`).Scan(&formatType).Error
	if err != nil {
		return 0, err
	}

	formatType = strings.TrimSpace(formatType)
	if formatType == "" {
		return 0, fmt.Errorf("user_stories.embedding column is missing")
	}

	matches := vectorTypePattern.FindStringSubmatch(formatType)
	if len(matches) != 2 {
		return 0, fmt.Errorf("user_stories.embedding has unsupported type %q", formatType)
	}

	dimension, err := strconv.Atoi(matches[1])
	if err != nil || dimension <= 0 {
		return 0, fmt.Errorf("invalid vector dimension %q", matches[1])
	}

	return dimension, nil
}

func ValidateStoryEmbeddingDimension(db *gorm.DB, embeddingSvc EmbeddingService) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}
	if embeddingSvc == nil {
		return fmt.Errorf("embedding service is required")
	}

	dbDimension, err := DetectStoryEmbeddingDimension(db)
	if err != nil {
		return err
	}

	serviceDimension := embeddingSvc.GetDimension()
	if serviceDimension <= 0 {
		return fmt.Errorf("embedding service returned invalid dimension %d", serviceDimension)
	}
	if dbDimension != serviceDimension {
		return fmt.Errorf("embedding dimension mismatch: database=%d service=%d", dbDimension, serviceDimension)
	}

	return nil
}
