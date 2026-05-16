package service

import (
	"sync"
	"time"

	"git.neolidy.top/neo/storybook/internal/metrics"
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

const aiServiceCacheLabel = "ai_service"

// AIService construction is hot-path: NewAIService is called per request. The
// previous implementation queried the DB and ran AES-256-GCM key decryption on
// every call. This file caches the constructed service keyed by AIConfig
// (id, updated_at). The next NewAIService call still issues a 1-row probe but
// skips decryption + client construction when the config has not changed.

type aiCacheEntry struct {
	svc       AIService
	configID  uint
	updatedAt time.Time
}

var (
	aiCacheMu sync.RWMutex
	aiCache   *aiCacheEntry
)

// InvalidateAIServiceCache clears the cached AIService. Call after upserting
// AIConfig — though strictly redundant since NewAIService re-validates against
// updated_at, this gives an immediate guarantee for callers that want it.
func InvalidateAIServiceCache() {
	aiCacheMu.Lock()
	aiCache = nil
	aiCacheMu.Unlock()
}

// cachedAIServiceFor returns the cached *openAIService when (id, updated_at)
// matches and a fresh one otherwise. Returns nil when the cache is cold so the
// caller knows to perform the (expensive) build.
func cachedAIServiceFor(configID uint, updatedAt time.Time) AIService {
	aiCacheMu.RLock()
	defer aiCacheMu.RUnlock()
	if aiCache != nil && aiCache.configID == configID && aiCache.updatedAt.Equal(updatedAt) {
		metrics.CacheHits.WithLabelValues(aiServiceCacheLabel).Inc()
		return aiCache.svc
	}
	metrics.CacheMisses.WithLabelValues(aiServiceCacheLabel).Inc()
	return nil
}

// storeAIServiceCache replaces the cache entry. Double-checks under write lock
// to avoid two concurrent NewAIService calls building the same service twice.
func storeAIServiceCache(svc AIService, configID uint, updatedAt time.Time) AIService {
	aiCacheMu.Lock()
	defer aiCacheMu.Unlock()
	if aiCache != nil && aiCache.configID == configID && aiCache.updatedAt.Equal(updatedAt) {
		return aiCache.svc
	}
	aiCache = &aiCacheEntry{svc: svc, configID: configID, updatedAt: updatedAt}
	return svc
}

// loadActiveAIConfig fetches the most recent enabled AIConfig.
// Returns (nil, nil) when no config exists; that is not an error condition.
func loadActiveAIConfig(db *gorm.DB) (*model.AIConfig, error) {
	var cfg model.AIConfig
	err := db.Where("enabled = ?", true).Order("id DESC").Limit(1).Find(&cfg).Error
	if err != nil {
		return nil, err
	}
	if cfg.ID == 0 {
		return nil, nil
	}
	return &cfg, nil
}
