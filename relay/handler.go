package relay

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
	"traffic-relay/config"
	"traffic-relay/logger"
	"traffic-relay/relay/internal"
)

func MakeProxyHandler(route config.Route) http.HandlerFunc {
	targets := route.BackendURLs
	if len(targets) == 0 && route.BackendURL != "" {
		targets = []string{route.BackendURL}
	}
	balancer := NewRoundRobinBalancer(targets)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			internal.HeaderCORS(w)
			w.WriteHeader(http.StatusOK)
			return
		}
		internal.HeaderCORS(w)

		bodyBytes := internal.ReadAndRestoreBody(r)
		logger.Logger.Printf("收到请求：\n%s", internal.DumpRequest(r, bodyBytes))

		method := internal.GetMethod(r, route.MethodOverride)
		targetBackend := balancer.Next()
		if targetBackend == "" {
			http.Error(w, "无可用后端地址", http.StatusServiceUnavailable)
			return
		}
		targetURL := internal.BuildTargetURL(targetBackend, r)

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(bodyBytes))
		if err != nil {
			http.Error(w, "构造请求失败", http.StatusInternalServerError)
			logger.Logger.Printf("构造请求失败: %v", err)
			return
		}
		req.Header = r.Header.Clone()

		logger.Logger.Printf("转发请求到：%s\n%s", targetURL, internal.DumpRequest(req, bodyBytes))

		resp, err := insecureHttpClient.Do(req)
		if err != nil {
			http.Error(w, "请求后端失败", http.StatusBadGateway)
			logger.Logger.Printf("请求后端失败: %v", err)
			return
		}
		defer resp.Body.Close()
		logger.Logger.Printf("后端响应状态码: %d", resp.StatusCode)

		internal.CopyHeaders(w, resp.Header)
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}
