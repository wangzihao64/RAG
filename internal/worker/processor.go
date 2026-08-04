// Package worker 提供后台任务处理能力。
package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"rag/config"
	"rag/internal/model"
	"rag/internal/repository"
	"rag/internal/service"
)

// ProcessorWorker 后台轮询待处理文档并执行处理流水线。
type ProcessorWorker struct {
	pipeline *service.Pipeline
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewProcessorWorker 构造一个文档处理 worker。
func NewProcessorWorker(ctx context.Context, pollInterval time.Duration) (*ProcessorWorker, error) {
	pipeline, err := service.NewPipeline(ctx)
	if err != nil {
		return nil, err
	}
	if pollInterval <= 0 {
		pollInterval = 10 * time.Second
	}
	return &ProcessorWorker{
		pipeline: pipeline,
		interval: pollInterval,
		stopCh:   make(chan struct{}),
	}, nil
}

// Start 启动后台轮询协程。
func (w *ProcessorWorker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run()
	}()
	log.Printf("文档处理 worker 已启动，轮询间隔 %v", w.interval)
}

// Stop 优雅停止 worker。
func (w *ProcessorWorker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	if w.pipeline != nil {
		_ = w.pipeline.Close()
	}
	log.Println("文档处理 worker 已停止")
}

// run 是后台轮询的主循环。
func (w *ProcessorWorker) run() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// 启动时立即执行一次
	w.pollAndProcess()

	for {
		select {
		case <-ticker.C:
			w.pollAndProcess()
		case <-w.stopCh:
			return
		}
	}
}

// pollAndProcess 原子领取待处理文档并逐批处理。
func (w *ProcessorWorker) pollAndProcess() {
	ctx := context.Background()
	dao := repository.NewDocumentDao(ctx)

	workerCount := config.WorkerCount
	if workerCount < 1 {
		workerCount = 1
	}

	for {
		docs, err := dao.ClaimPendingDocuments(workerCount)
		if err != nil {
			log.Printf("领取待处理文档失败: %v", err)
			return
		}
		if len(docs) == 0 {
			return
		}

		log.Printf("领取到 %d 个待处理文档", len(docs))
		jobs := make(chan model.Document)
		var jobsWg sync.WaitGroup
		jobsWg.Add(len(docs))

		for i := 0; i < len(docs); i++ {
			go func(workerID int) {
				defer jobsWg.Done()
				for doc := range jobs {
					if err := w.pipeline.ProcessDocument(ctx, doc.ID); err != nil {
						log.Printf("worker-%d 处理文档 %d 失败: %v", workerID, doc.ID, err)
					} else {
						log.Printf("worker-%d 处理文档 %d 完成", workerID, doc.ID)
					}
				}
			}(i + 1)
		}

		for _, doc := range docs {
			jobs <- doc
		}
		close(jobs)
		jobsWg.Wait()
	}
}
