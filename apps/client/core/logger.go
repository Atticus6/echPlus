package core

import (
	"fmt"
	"log"
)

// LogHandler 日志处理接口，由外部实现
type LogHandler interface {
	Info(msg string)
	Error(msg string)
	Debug(msg string)
}

var logHandler LogHandler

// SetLogHandler 设置日志处理器
func SetLogHandler(handler LogHandler) {
	logHandler = handler
}

// LogInfo 记录 info 日志
func LogInfo(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if logHandler != nil {
		logHandler.Info(msg)
	} else {
		log.Printf("[INFO] %s", msg)
	}
}

// LogError 记录 error 日志
func LogError(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if logHandler != nil {
		logHandler.Error(msg)
	} else {
		log.Printf("[ERROR] %s", msg)
	}
}

// LogDebug 记录 debug 日志
func LogDebug(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if logHandler != nil {
		logHandler.Debug(msg)
	} else {
		log.Printf("[DEBUG] %s", msg)
	}
}
