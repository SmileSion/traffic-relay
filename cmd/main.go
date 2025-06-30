package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"traffic-relay/config"
	"traffic-relay/logger"
	"traffic-relay/relay"
	"traffic-relay/utils"
)

func main() {
	configPath := flag.String("c", "config/config.toml", "配置文件路径")
	flag.Parse()

	config.InitConfig(*configPath)
	logger.InitLogger()
	utils.StartQPSMonitor()

	for _, route := range config.Conf.Routes {
		http.HandleFunc(route.ListenPath, relay.MakeProxyHandler(route))
		var backends string
		if len(route.BackendURLs) > 0 {
			backends = strings.Join(route.BackendURLs, ", ")
		} else {
			backends = route.BackendURL
		}
		log.Printf("注册代理: %s => %s (method_override=%s)", route.ListenPath, backends, route.MethodOverride)
	}

	log.Printf("启动代理服务器监听：%s", config.Conf.Relay.ListenAddr)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- http.ListenAndServe(config.Conf.Relay.ListenAddr, nil)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("收到退出信号 %v，准备关闭...", sig)
		if err := logger.CloseAsyncWriters(); err != nil {
			log.Printf("关闭日志写入器出错: %v", err)
		} else {
			log.Printf("日志写入器已关闭")
		}
	case err := <-serverErrCh:
		log.Fatalf("HTTP 服务器异常退出: %v", err)
	}
}
