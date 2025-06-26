#!/bin/bash

APP_NAME="Traffic-relay-linux-amd64"                  # 可执行文件名称（需替换为你的程序）
LOG_FILE="./run.log"            # 日志文件路径
PID_FILE="./run.pid"            # 保存进程 ID 的文件

start() {
    if [ -f $PID_FILE ] && kill -0 $(cat $PID_FILE) 2>/dev/null; then
        echo "$APP_NAME is already running (PID=$(cat $PID_FILE))"
    else
        echo "Starting $APP_NAME..."
        nohup ./$APP_NAME > $LOG_FILE 2>&1 &
        echo $! > $PID_FILE
        echo "$APP_NAME started with PID $(cat $PID_FILE)"
    fi
}

stop() {
    if [ -f $PID_FILE ]; then
        PID=$(cat $PID_FILE)
        echo "Stopping $APP_NAME (PID=$PID)..."
        kill $PID
        rm -f $PID_FILE
        echo "$APP_NAME stopped."
    else
        echo "$APP_NAME is not running."
    fi
}

status() {
    if [ -f $PID_FILE ] && kill -0 $(cat $PID_FILE) 2>/dev/null; then
        echo "$APP_NAME is running (PID=$(cat $PID_FILE))"
    else
        echo "$APP_NAME is not running."
    fi
}

restart() {
    echo "Restarting $APP_NAME..."
    stop
    sleep 1
    start
}

case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
esac
