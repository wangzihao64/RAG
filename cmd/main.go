package main

import (
	"log"

	"rag/config"
	"rag/internal/dao"
	"rag/internal/router"
)

func Init() {
	config.Init()
	dao.InitPostgreSQL()
}

func main() {
	Init()

	r := router.NewRouter()
	if err := r.Run(config.HttpPort); err != nil {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}
