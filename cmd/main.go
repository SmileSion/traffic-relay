package main

import (
	"log"
	"net/http"
	"strings"
	"traffic-relay/config"
	"traffic-relay/logger"
	"traffic-relay/relay"
)

func main() {
	config.InitConfig()
	logger.InitLogger()

	for _, route := range config.Conf.Routes {
		http.HandleFunc(route.ListenPath, relay.MakeProxyHandler(route))
		// 打印所有后端地址（轮询列表）或单个地址
		var backends string
		if len(route.BackendURLs) > 0 {
			backends = strings.Join(route.BackendURLs, ", ")
		} else {
			backends = route.BackendURL
		}
		log.Printf("注册代理: %s => %s (method_override=%s)", route.ListenPath, backends, route.MethodOverride)
	}

	log.Printf("启动代理服务器监听：%s", config.Conf.Relay.ListenAddr)
	if err := http.ListenAndServe(config.Conf.Relay.ListenAddr, nil); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
