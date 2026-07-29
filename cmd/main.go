package main

import (
	"context"
	"log"
	"time"

	"rag/config"
	"rag/internal/repository"
	"rag/internal/router"
	"rag/internal/service"
	"rag/internal/worker"
)

func Init() {
	config.Init()
	repository.InitPostgreSQL()
}

// milvusInitTimeout 限制启动阶段连接 Milvus 的握手时长。
// Milvus 不可用时，client.NewClient 会无限重试阻塞；没有这个超时，
// HTTP 服务将永远无法启动（连不依赖 Milvus 的登录接口也会拒绝连接）。
const milvusInitTimeout = 10 * time.Second

func main() {
	Init()

	ctx := context.Background()

	// Milvus 为软依赖：初始化失败只告警、不退出，保证 HTTP 服务（登录、注册等
	// 不依赖 Milvus 的接口）始终可用。文档处理与在线查询在 Milvus 不可用时降级：
	// worker 不启动、query/chat 接口返回 503。Milvus 恢复后需重启进程使其生效。

	// 文档处理 worker
	workerCtx, cancelWorker := context.WithTimeout(ctx, milvusInitTimeout)
	w, err := worker.NewProcessorWorker(workerCtx, 10*time.Second)
	cancelWorker()
	if err != nil {
		log.Printf("[WARN] 文档处理 worker 初始化失败，文档处理暂不可用（Milvus 恢复后重启生效）: %v", err)
	} else {
		w.Start()
		defer w.Stop()
	}

	// 在线查询资源（Milvus + embedding + llm）
	queryCtx, cancelQuery := context.WithTimeout(ctx, milvusInitTimeout)
	err = service.InitQuery(queryCtx)
	cancelQuery()
	if err != nil {
		log.Printf("[WARN] 在线查询初始化失败，检索/问答接口将返回 503（Milvus 恢复后重启生效）: %v", err)
	} else {
		defer service.CloseQuery()
	}

	r := router.NewRouter()
	if err := r.Run(config.HttpPort); err != nil {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}
