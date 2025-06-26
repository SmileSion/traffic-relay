package relay

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
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

// 定义一个跳过证书验证的 HTTP 客户端
var insecureHttpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
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

		// 如果原请求有查询参数，附加上去
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		var body io.Reader
		if method == http.MethodGet {
			// GET 请求一般没有 Body，直接置空
			body = nil
		} else {
			// 其他方法保持 Body
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

		// 执行转发（使用跳过证书验证的客户端）
		resp, err := insecureHttpClient.Do(req)
		if err != nil {
			http.Error(w, "请求后端失败", http.StatusBadGateway)
			logger.Logger.Printf("请求后端失败: %v", err)
			return
		}
		defer resp.Body.Close()

		// 拷贝响应头和响应体
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}
