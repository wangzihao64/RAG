package main

import (
	"log"
	"rag/internal/repository"

	"rag/config"
	"rag/internal/router"
)

func Init() {
	config.Init()
	repository.InitPostgreSQL()
}

func main() {
	Init()

	r := router.NewRouter()
	if err := r.Run(config.HttpPort); err != nil {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}
