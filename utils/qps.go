package utils

import (
	"sync/atomic"
	"time"
	"traffic-relay/logger"
)

var counter int64

// Inc 增加请求计数
func Inc() {
	atomic.AddInt64(&counter, 1)
}

// StartQPSMonitor 启动 QPS 统计协程
func StartQPSMonitor() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			c := atomic.SwapInt64(&counter, 0)
			if c >= 100 {
				logger.Logger.Infof("[QPS] 当前并发高，每秒请求数：%d\n", c)
			}
		}
	}()
}
