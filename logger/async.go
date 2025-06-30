package logger

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// AsyncWriter 实现异步写入的结构体
type AsyncWriter struct {
	ch     chan []byte
	writer io.Writer
	wg     sync.WaitGroup
	closed bool
	mu     sync.Mutex
}

// NewAsyncWriter 创建一个异步写入器
func NewAsyncWriter(w io.Writer) *AsyncWriter {
	aw := &AsyncWriter{
		ch:     make(chan []byte, 1000), // 可调缓冲大小
		writer: w,
	}
	aw.wg.Add(1)
	go aw.run()
	return aw
}

// Write 实现 io.Writer 接口（写满时阻塞）
func (aw *AsyncWriter) Write(p []byte) (int, error) {
	aw.mu.Lock()
	if aw.closed {
		aw.mu.Unlock()
		return 0, fmt.Errorf("async writer is closed")
	}
	aw.mu.Unlock()

	data := append([]byte(nil), p...) // 拷贝切片防止变更
	aw.ch <- data                     // 阻塞直到通道有空间
	return len(p), nil
}

// run 后台写入协程，写失败时重试
func (aw *AsyncWriter) run() {
	defer aw.wg.Done()
	for msg := range aw.ch {
		for {
			_, err := aw.writer.Write(msg)
			if err == nil {
				break
			}
			// 简单重试，等待100ms后再写
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Close 优雅关闭，等待所有日志写完
func (aw *AsyncWriter) Close() error {
	aw.mu.Lock()
	if aw.closed {
		aw.mu.Unlock()
		return fmt.Errorf("async writer already closed")
	}
	aw.closed = true
	aw.mu.Unlock()

	close(aw.ch)
	aw.wg.Wait()
	return nil
}
