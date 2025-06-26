package relay

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

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

// insecureHttpClient 是跳过证书验证的客户端
var insecureHttpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// RoundRobinBalancer 轮询负载均衡器
type RoundRobinBalancer struct {
	counter uint64
	targets []string
}

func NewRoundRobinBalancer(targets []string) *RoundRobinBalancer {
	return &RoundRobinBalancer{targets: targets}
}

func (rr *RoundRobinBalancer) Next() string {
	if len(rr.targets) == 0 {
		return ""
	}
	idx := atomic.AddUint64(&rr.counter, 1)
	return rr.targets[int(idx-1)%len(rr.targets)]
}

// MakeProxyHandler 返回一个代理处理函数，支持多个后端地址轮询转发和方法重写
func MakeProxyHandler(route config.Route) http.HandlerFunc {
	targets := route.BackendURLs
	if len(targets) == 0 && route.BackendURL != "" {
		targets = []string{route.BackendURL}
	}
	balancer := NewRoundRobinBalancer(targets)

	return func(w http.ResponseWriter, r *http.Request) {
		// 处理 OPTIONS 预检请求
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")

		logger.Logger.Printf("收到请求：\n%s", dumpRequest(r))

		method := r.Method
		if route.MethodOverride != "" && strings.ToUpper(route.MethodOverride) != r.Method {
			method = strings.ToUpper(route.MethodOverride)
			logger.Logger.Printf("方法重写: %s => %s", r.Method, method)
		}

		targetBackend := balancer.Next()
		if targetBackend == "" {
			http.Error(w, "无可用后端地址", http.StatusServiceUnavailable)
			return
		}

		targetURL := strings.TrimRight(targetBackend, "/") + r.URL.Path
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		var body io.Reader
		if method == http.MethodGet {
			body = nil
		} else {
			body = r.Body
		}

		req, err := http.NewRequest(method, targetURL, body)
		if err != nil {
			http.Error(w, "构造请求失败", http.StatusInternalServerError)
			logger.Logger.Printf("构造请求失败: %v", err)
			return
		}
		req.Header = r.Header.Clone()

		logger.Logger.Printf("转发请求到：%s\n%s", targetURL, dumpRequest(req))

		resp, err := insecureHttpClient.Do(req)
		if err != nil {
			http.Error(w, "请求后端失败", http.StatusBadGateway)
			logger.Logger.Printf("请求后端失败: %v", err)
			return
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}
