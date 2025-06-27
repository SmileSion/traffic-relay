package relay

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"traffic-relay/config"
	"traffic-relay/logger"
)

const maxLogSize = 1024

// dumpRequest 打印请求（用于日志记录）
func dumpRequest(r *http.Request, body []byte) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s %s %s\n", r.Method, r.URL.RequestURI(), r.Proto)
	for k, v := range r.Header {
		fmt.Fprintf(&b, "%s: %s\n", k, v)
	}

	//截断日志，避免过长攻击
	if len(body) > maxLogSize {
		fmt.Fprintf(&b, "\n%s\n[Body已截断，原始长度 %d 字节]", string(body[:maxLogSize]), len(body))
	} else {
		fmt.Fprintf(&b, "\n%s\n", string(body))
	}
	return b.String()
}

// insecureHttpClient 是跳过证书验证的客户端
var insecureHttpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        100,              //客户端维护的最大空闲（保持活动的）TCP 连接数
		MaxIdleConnsPerHost: 100,              //每个目标主机（host）允许保持的最大空闲连接数
		IdleConnTimeout:     90 * time.Second, //空闲连接的超时时间，超过该时间未被使用的空闲连接会被关闭
	},
	Timeout: 30 * time.Second,
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

		//读取、缓存请求体
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			// 关闭旧的 Body，防止资源泄漏
			r.Body.Close()
			// 重新设置 Body，保证后续可以正常读取
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		logger.Logger.Printf("收到请求：\n%s", dumpRequest(r, bodyBytes))

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
			body = bytes.NewReader(bodyBytes)
		}

		//精细超时控制
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
		if err != nil {
			http.Error(w, "构造请求失败", http.StatusInternalServerError)
			logger.Logger.Printf("构造请求失败: %v", err)
			return
		}
		req.Header = r.Header.Clone()

		logger.Logger.Printf("转发请求到：%s\n%s", targetURL, dumpRequest(req, bodyBytes))

		resp, err := insecureHttpClient.Do(req)
		if err != nil {
			http.Error(w, "请求后端失败", http.StatusBadGateway)
			logger.Logger.Printf("请求后端失败: %v", err)
			return
		}
		defer resp.Body.Close()
		logger.Logger.Printf("后端响应状态码: %d", resp.StatusCode)

		for k, v := range resp.Header {
			//过滤响应字段，如需使用打开即可
			// lowerK := strings.ToLower(k)
			// if lowerK == "content-length" || lowerK == "transfer-encoding" || lowerK == "connection" {
			// 	continue
			// }
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}
