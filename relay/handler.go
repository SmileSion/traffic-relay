package relay

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
	"traffic-relay/config"
	"traffic-relay/logger"
	"traffic-relay/relay/internal"
)

func MakeProxyHandler(route config.Route) http.HandlerFunc {
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
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
		logger.Logger.Printf("[ID:%s] 收到请求：\n%s", requestID, internal.DumpRequest(r, bodyBytes))

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

		logger.Logger.Printf("[ID:%s] 转发请求到：%s\n%s", requestID, targetURL, internal.DumpRequest(req, bodyBytes))

		resp, err := insecureHttpClient.Do(req)
		if err != nil {
			http.Error(w, "请求后端失败", http.StatusBadGateway)
			logger.Logger.Printf("请求后端失败: %v", err)
			return
		}
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "读取后端响应失败", http.StatusInternalServerError)
			logger.Logger.Printf("读取后端响应失败: %v", err)
			return
		}
		logger.Logger.Printf("[ID:%s] 后端响应：\n%s", requestID, internal.DumpResponse(resp))

		internal.CopyHeaders(w, resp.Header)
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(respBody)
	}
}
