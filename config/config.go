package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

type LogConfig struct {
	Filepath      string `toml:"filepath"`
	MaxSize       int    `toml:"max_size"`
	MaxBackups    int    `toml:"max_backups"`
	MaxAge        int    `toml:"max_age"`
	Level         string `toml:"level"`
	Compress      bool   `toml:"compress"`
	EnableConsole bool   `toml:"enable_console"`
}

type Route struct {
	ListenPath     string   `toml:"listen_path"`
	BackendURL     string   `toml:"backend_url"`  // 兼容旧单地址配置
	BackendURLs    []string `toml:"backend_urls"` // 新支持多个地址
	MethodOverride string   `toml:"method_override"`
}

type RelayConfig struct {
	ListenAddr string `toml:"listen_addr"`
}

type Config struct {
	Log    LogConfig   `toml:"log"`
	Relay  RelayConfig `toml:"relay"`
	Routes []Route     `toml:"routes"`
}

var Conf Config

// InitConfig 使用指定路径加载配置文件
func InitConfig(path string) {
	if _, err := toml.DecodeFile(path, &Conf); err != nil {
		panic(err)
	}
	log.Printf("配置文件加载成功，监听地址：%s", Conf.Relay.ListenAddr)
	for _, r := range Conf.Routes {
		log.Printf("路由配置: %s => %v (method_override=%s)", r.ListenPath, r.BackendURLsOrSingle(), r.MethodOverride)
	}
}

// 返回路由使用的后端地址列表（兼容单个地址）
func (r *Route) BackendURLsOrSingle() []string {
	if len(r.BackendURLs) > 0 {
		return r.BackendURLs
	}
	if r.BackendURL != "" {
		return []string{r.BackendURL}
	}
	return nil
}
