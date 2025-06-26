package main

import (
	"log"
	"net/http"
	"traffic-relay/config"
	"traffic-relay/logger"
	"traffic-relay/relay"
)

func main() {
	config.InitConfig()
	logger.InitLogger()

	for _, route := range config.Conf.Routes {
		http.HandleFunc(route.ListenPath, relay.MakeProxyHandler(route))
		log.Printf("注册代理: %s => %s (method_override=%s)", route.ListenPath, route.BackendURL, route.MethodOverride)
	}

	log.Printf("启动代理服务器监听：%s", config.Conf.Relay.ListenAddr)
	if err := http.ListenAndServe(config.Conf.Relay.ListenAddr, nil); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
