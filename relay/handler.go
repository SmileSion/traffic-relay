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
	"traffic-relay/utils"
)

func MakeProxyHandler(route config.Route) http.HandlerFunc {
	targets := route.BackendURLs
	if len(targets) == 0 && route.BackendURL != "" {
		targets = []string{route.BackendURL}
	}
	balancer := NewRoundRobinBalancer(targets)
	return func(w http.ResponseWriter, r *http.Request) {
		utils.Inc()
		requestID := fmt.Sprintf("%d", time.Now().UnixNano())

		if r.Method == http.MethodOptions {
			internal.HeaderCORS(w)
			w.WriteHeader(http.StatusOK)
			return
		}
		internal.HeaderCORS(w)

		bodyBytes := internal.ReadAndRestoreBody(r)
		logger.Logger.Debugf("[ID:%s] 收到请求：\n%s", requestID, internal.DumpRequest(r, bodyBytes))

		method := internal.GetMethod(r, route.MethodOverride)
		targetBackend := balancer.Next()
		if targetBackend == "" {
			http.Error(w, "无可用后端地址", http.StatusServiceUnavailable)
			logger.Logger.Warnf("[ID:%s] 无可用后端地址", requestID)
			return
		}
		targetURL := internal.BuildTargetURL(targetBackend, r)

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(bodyBytes))
		if err != nil {
			http.Error(w, "构造请求失败", http.StatusInternalServerError)
			logger.Logger.Errorf("[ID:%s] 构造请求失败: %v", requestID, err)
			return
		}
		req.Header = r.Header.Clone()

		logger.Logger.Infof("[ID:%s] 转发请求到：%s", requestID, targetURL)
		logger.Logger.Debugf("[ID:%s] 转发请求内容：\n%s", requestID, internal.DumpRequest(req, bodyBytes))

		resp, err := insecureHttpClient.Do(req)
		if err != nil {
			http.Error(w, "请求后端失败", http.StatusBadGateway)
			logger.Logger.Errorf("[ID:%s] 请求后端失败: %v", requestID, err)
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "读取后端响应失败", http.StatusInternalServerError)
			logger.Logger.Errorf("[ID:%s] 读取后端响应失败: %v", requestID, err)
			return
		}

		logger.Logger.Debugf("[ID:%s] 后端响应内容：\n%s", requestID, internal.DumpResponse(resp))

		internal.CopyHeaders(w, resp.Header)
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(respBody)
	}
}
