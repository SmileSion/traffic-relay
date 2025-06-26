package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"traffic-relay/config"
	"traffic-relay/logger"
)

// dumpRequest 打印请求（用于日志记录）
func dumpRequest(r *http.Request) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s %s %s\n", r.Method, r.URL.RequestURI(), r.Proto)
	for k, v := range r.Header {
		fmt.Fprintf(&b, "%s: %s\n", k, v)
	}
	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		fmt.Fprintf(&b, "\n%s\n", string(body))
		// 重置 Body，确保后续仍可读
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return b.String()
}

// MakeProxyHandler 返回一个代理函数，将请求转发至 backendURL，并根据配置覆盖 method
func MakeProxyHandler(route config.Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 处理 OPTIONS 预检请求
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.WriteHeader(http.StatusOK)
			return
		}

		// 设置跨域响应头
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// 日志记录：接收请求
		logger.Logger.Printf("收到请求：\n%s", dumpRequest(r))

		// 判断是否需要 method 重写
		method := r.Method
		if route.MethodOverride != "" && strings.ToUpper(route.MethodOverride) != r.Method {
			logger.Logger.Printf("方法重写: %s => %s", r.Method, route.MethodOverride)
			method = strings.ToUpper(route.MethodOverride)
		}

		// 目标 URL
		targetURL := strings.TrimRight(route.BackendURL, "/") + r.URL.Path

		// 处理请求体
		var body io.Reader
		var queryStr string

		if method == http.MethodGet {
			// POST → GET：将 Body 转为 query string 附在 URL 后
			if r.Body != nil {
				bodyBytes, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // 重设
				queryStr = string(bodyBytes)
				targetURL += "?" + url.QueryEscape(queryStr)
			}
			body = nil
		} else {
			// GET → POST 或保持 POST
			body = r.Body
		}

		// 构造转发请求
		req, err := http.NewRequest(method, targetURL, body)
		if err != nil {
			http.Error(w, "构造请求失败", http.StatusInternalServerError)
			logger.Logger.Printf("构造请求失败: %v", err)
			return
		}
		req.Header = r.Header.Clone()

		// 日志：转发请求
		logger.Logger.Printf("转发请求到：%s\n%s", targetURL, dumpRequest(req))

		// 执行转发
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "请求后端失败", http.StatusBadGateway)
			logger.Logger.Printf("请求后端失败: %v", err)
			return
		}
		defer resp.Body.Close()

		// 响应头和响应体
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}
