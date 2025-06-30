package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"traffic-relay/config"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger       = logrus.New()
	asyncWriters []*AsyncWriter
	mu           sync.Mutex
)

type PrefixFormatter struct{}

func (f *PrefixFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05.000")
	level := strings.ToUpper(entry.Level.String())
	message := entry.Message
	return []byte(fmt.Sprintf("[traffic-relay] %s [%s] %s\n", timestamp, level, message)), nil
}


// InitLogger 初始化日志
func InitLogger() {
	logConf := config.Conf.Log

	Logger.SetFormatter(&PrefixFormatter{})

	level, err := logrus.ParseLevel(strings.ToLower(logConf.Level))
	if err != nil {
		level = logrus.InfoLevel
	}
	Logger.SetLevel(level)

	var writers []io.Writer
	mu.Lock()
	defer mu.Unlock()

	// 清空之前的异步写入器（如果有）
	asyncWriters = asyncWriters[:0]

	if logConf.EnableConsole {
		w := NewAsyncWriter(os.Stdout)
		asyncWriters = append(asyncWriters, w)
		writers = append(writers, w)
	}
	if logConf.Filepath != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   logConf.Filepath,
			MaxSize:    logConf.MaxSize,
			MaxBackups: logConf.MaxBackups,
			MaxAge:     logConf.MaxAge,
			Compress:   logConf.Compress,
		}
		w := NewAsyncWriter(fileWriter)
		asyncWriters = append(asyncWriters, w)
		writers = append(writers, w)
	}

	Logger.SetOutput(io.MultiWriter(writers...))
}

// CloseAsyncWriters 关闭所有异步写入器，优雅退出时调用
func CloseAsyncWriters() error {
	mu.Lock()
	defer mu.Unlock()

	var err error
	for _, w := range asyncWriters {
		if closeErr := w.Close(); closeErr != nil {
			err = closeErr // 记录最后一个错误，也可以用多错误处理
		}
	}
	asyncWriters = asyncWriters[:0]
	return err
}
