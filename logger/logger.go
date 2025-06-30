package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"traffic-relay/config"
)

var Logger = logrus.New()

func InitLogger() {
	logConf := config.Conf.Log

	// 设置日志格式
	Logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: time.DateTime,
		FullTimestamp:   true,
		ForceColors:     false,
		DisableColors:   true,
	})

	// 设置日志级别（支持 debug/info/warn/error）
	level, err := logrus.ParseLevel(strings.ToLower(logConf.Level))
	if err != nil {
		level = logrus.InfoLevel
	}
	Logger.SetLevel(level)

	// 输出目标
	var writers []io.Writer
	if logConf.EnableConsole {
		writers = append(writers, os.Stdout)
	}
	if logConf.Filepath != "" {
		writers = append(writers, &lumberjack.Logger{
			Filename:   logConf.Filepath,
			MaxSize:    logConf.MaxSize,
			MaxBackups: logConf.MaxBackups,
			MaxAge:     logConf.MaxAge,
			Compress:   logConf.Compress,
		})
	}

	// 多路输出
	Logger.SetOutput(io.MultiWriter(writers...))
}
