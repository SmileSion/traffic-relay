package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		// 重要：重设 Body，使后续请求可用
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return b.String()
}

// MakeProxyHandler 返回一个代理函数，将指定路径请求转发至 backendURL
func MakeProxyHandler(backendURL string) http.HandlerFunc {
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

		// 日志记录：接收到的请求
		logger.Logger.Printf("收到请求：\n%s", dumpRequest(r))

		// 拼接目标 URL
		targetURL := strings.TrimRight(backendURL, "/") + r.URL.Path

		// 构造新请求转发
		req, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			http.Error(w, "构造请求失败", http.StatusInternalServerError)
			logger.Logger.Printf("构造请求失败: %v", err)
			return
		}
		req.Header = r.Header.Clone()

		// 日志记录：转发的请求
		logger.Logger.Printf("转发请求到：%s\n%s", targetURL, dumpRequest(req))

		// 执行请求
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "请求后端失败", http.StatusBadGateway)
			logger.Logger.Printf("请求后端失败: %v", err)
			return
		}
		defer resp.Body.Close()

		// 拷贝响应头和内容
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}
