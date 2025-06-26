package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

// 日志配置结构体
type LogConfig struct {
	Filepath   string `toml:"filepath"`
	MaxSize    int    `toml:"max_size"`
	MaxBackups int    `toml:"max_backups"`
	MaxAge     int    `toml:"max_age"`
	Compress   bool   `toml:"compress"`
}

// 转发配置结构体
type RelayConfig struct {
	ListenAddr string `toml:"listen_addr"`
}

// 单个路由规则
type Route struct {
	ListenPath string `toml:"listen_path"`
	BackendURL string `toml:"backend_url"`
	MethodOverride string `toml:"method_override"`
}

// 总配置结构体
type Config struct {
	Log    LogConfig   `toml:"log"`
	Relay  RelayConfig `toml:"relay"`
	Routes []Route     `toml:"routes"`
}

var Conf Config

// 初始化配置
func InitConfig() {
	if _, err := toml.DecodeFile("config/config.toml", &Conf); err != nil {
		panic(err)
	}

	log.Println("配置文件加载成功：")
	log.Printf("  ListenAddr: %s\n", Conf.Relay.ListenAddr)
	for _, route := range Conf.Routes {
		log.Printf("  路由: %s => %s\n", route.ListenPath, route.BackendURL)
	}
}
