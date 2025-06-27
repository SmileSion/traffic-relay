package logger

import (
	"log"

	"gopkg.in/natefinch/lumberjack.v2"
	"traffic-relay/config"
)

var Logger *log.Logger

func InitLogger() {
	Logger = log.New(&lumberjack.Logger{
		Filename:   config.Conf.Log.Filepath,
		MaxSize:    config.Conf.Log.MaxSize,
		MaxBackups: config.Conf.Log.MaxBackups,
		MaxAge:     config.Conf.Log.MaxAge,
		Compress:   config.Conf.Log.Compress,
	}, "[Traffic-Relay] ", log.LstdFlags)
}
