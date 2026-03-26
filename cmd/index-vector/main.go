// Package main 提供向量化索引 CLI 工具
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.neolidy.top/neo/storybook/internal/config"
	"git.neolidy.top/neo/storybook/internal/database"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
)

func main() {
	// 命令行参数
	provider := flag.String("provider", "mock", "Embedding 提供商: mock, openai, ollama")
	apiKey := flag.String("api-key", "", "OpenAI API Key（provider=openai 时必需）")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama 服务地址")
	ollamaModel := flag.String("ollama-model", "nomic-embed-text", "Ollama 模型名称")
	ollamaDimension := flag.Int("ollama-dimension", 0, "Ollama Embedding 维度（未设置时按模型推断）")
	batchSize := flag.Int("batch", 10, "批处理大小")
	force := flag.Bool("force", false, "强制重新索引所有故事（包括已有向量的）")
	timeout := flag.Duration("timeout", 30*time.Minute, "超时时间（0表示无超时）")
	flag.Parse()

	log.Printf("🚀 开始向量化索引...")
	log.Printf("配置: provider=%s, batch=%d, force=%v", *provider, *batchSize, *force)

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// 连接数据库
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}

	// 创建可取消的context，支持超时和信号中断
	ctx, cancel := contextWithTimeoutOrSignal(*timeout)
	defer cancel()

	// 创建 Embedding Service
	var embeddingSvc service.EmbeddingService
	switch *provider {
	case "openai":
		if *apiKey == "" {
			*apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if *apiKey == "" {
			log.Fatal("❌ OpenAI API Key 未提供（请使用 -api-key 或设置 OPENAI_API_KEY 环境变量）")
		}
		embeddingSvc = service.NewOpenAIEmbedding(*apiKey)
		log.Printf("✅ 使用 OpenAI Embedding (text-embedding-3-small, 1536维)")

	case "ollama":
		if *ollamaDimension > 0 {
			embeddingSvc = service.NewOllamaEmbeddingWithDimension(*ollamaURL, *ollamaModel, *ollamaDimension)
		} else {
			embeddingSvc = service.NewOllamaEmbedding(*ollamaURL, *ollamaModel)
		}
		log.Printf("✅ 使用 Ollama Embedding (%s, %s, %d维)", *ollamaURL, *ollamaModel, embeddingSvc.GetDimension())

	case "mock":
		embeddingSvc = service.NewMockEmbedding()
		log.Printf("✅ 使用 Mock Embedding (1536维, 测试模式)")

	default:
		log.Fatalf("❌ 不支持的 provider: %s（支持: mock, openai, ollama）", *provider)
	}

	if err := service.ValidateStoryEmbeddingDimension(db, embeddingSvc); err != nil {
		log.Fatalf("❌ 向量 schema 校验失败: %v", err)
	}

	// 创建 Vector Service
	vectorSvc := service.NewVectorServiceWithBatchSize(db, embeddingSvc, *batchSize)

	// 查询需要索引的故事
	var stories []model.UserStory
	query := db.Model(&model.UserStory{}).Where("archived = false")

	if !*force {
		// 只索引没有向量的故事
		query = query.Where("embedding IS NULL")
	}

	if err := query.Find(&stories).Error; err != nil {
		log.Fatalf("❌ 查询故事失败: %v", err)
	}

	totalCount := len(stories)
	if totalCount == 0 {
		log.Println("✅ 没有需要索引的故事")
		return
	}

	log.Printf("📊 找到 %d 条需要索引的故事", totalCount)

	// 批量索引
	if err := vectorSvc.BatchIndexStories(ctx, stories); err != nil {
		log.Fatalf("❌ 批量索引失败: %v", err)
	}

	log.Printf("🎉 索引完成！成功索引 %d 条故事", totalCount)

	// 验证索引结果
	var indexedCount int64
	if *force {
		db.Model(&model.UserStory{}).Where("archived = false AND embedding IS NOT NULL").Count(&indexedCount)
	} else {
		db.Model(&model.UserStory{}).Where("archived = false AND embedding IS NOT NULL").Count(&indexedCount)
	}

	log.Printf("📈 数据库中共有 %d 条已索引的故事", indexedCount)
}

// contextWithTimeoutOrSignal 创建一个支持超时和系统信号的context
func contextWithTimeoutOrSignal(timeout time.Duration) (context.Context, context.CancelFunc) {
	// 基础context
	ctx := context.Background()

	// 如果指定了超时，使用 WithTimeout
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, timeout)

		// 同时监听系统信号
		signalCtx, signalCancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)

		// 返回组合的context和取消函数
		return signalCtx, func() {
			signalCancel()
			cancel()
		}
	}

	// 没有超时时，只监听信号
	return signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
}
