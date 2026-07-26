package main

import (
	"context"
	"log"
	"time"

	"rag/config"
	"rag/internal/repository"
	"rag/internal/router"
	"rag/internal/worker"
)

func Init() {
	config.Init()
	repository.InitPostgreSQL()
}

func main() {
	Init()

	// 启动文档处理 worker
	ctx := context.Background()
	w, err := worker.NewProcessorWorker(ctx, 10*time.Second)
	if err != nil {
		log.Fatalf("启动 worker 失败: %v", err)
	}
	w.Start()
	defer w.Stop()

	r := router.NewRouter()
	if err := r.Run(config.HttpPort); err != nil {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}
