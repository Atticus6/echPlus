package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	baseDir     string
	currentDate string
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	infoFile    *os.File
	errorFile   *os.File
	debugFile   *os.File
	mu          sync.Mutex
}

var defaultLogger *Logger

// Init 初始化日志系统
func Init(baseDir string) error {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	defaultLogger = &Logger{
		baseDir: baseDir,
	}

	return defaultLogger.rotateFiles()
}

// 按日期轮转日志文件
func (l *Logger) rotateFiles() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if l.currentDate == today {
		return nil
	}

	// 关闭旧文件
	if l.infoFile != nil {
		l.infoFile.Close()
	}
	if l.errorFile != nil {
		l.errorFile.Close()
	}
	if l.debugFile != nil {
		l.debugFile.Close()
	}

	l.currentDate = today
	flags := log.Ltime | log.Lshortfile

	// 创建 info 日志文件
	infoPath := filepath.Join(l.baseDir, fmt.Sprintf("info_%s.log", today))
	infoFile, err := os.OpenFile(infoPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	l.infoFile = infoFile
	l.infoLogger = log.New(infoFile, "[INFO] ", flags)

	// 创建 error 日志文件
	errorPath := filepath.Join(l.baseDir, fmt.Sprintf("error_%s.log", today))
	errorFile, err := os.OpenFile(errorPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	l.errorFile = errorFile
	l.errorLogger = log.New(errorFile, "[ERROR] ", flags)

	// 创建 debug 日志文件
	debugPath := filepath.Join(l.baseDir, fmt.Sprintf("debug_%s.log", today))
	debugFile, err := os.OpenFile(debugPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	l.debugFile = debugFile
	l.debugLogger = log.New(debugFile, "[DEBUG] ", flags)

	return nil
}

func (l *Logger) checkRotate() {
	today := time.Now().Format("2006-01-02")
	if l.currentDate != today {
		l.rotateFiles()
	}
}

func Info(format string, v ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.checkRotate()
	defaultLogger.infoLogger.Output(2, fmt.Sprintf(format, v...))
}

func Error(format string, v ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.checkRotate()
	defaultLogger.errorLogger.Output(2, fmt.Sprintf(format, v...))
}

func Debug(format string, v ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.checkRotate()
	defaultLogger.debugLogger.Output(2, fmt.Sprintf(format, v...))
}

// Fatal 记录错误并退出
func Fatal(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.checkRotate()
		defaultLogger.errorLogger.Output(2, fmt.Sprintf(format, v...))
	}
	os.Exit(1)
}

// Close 关闭所有日志文件
func Close() {
	if defaultLogger == nil {
		return
	}
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()

	if defaultLogger.infoFile != nil {
		defaultLogger.infoFile.Close()
	}
	if defaultLogger.errorFile != nil {
		defaultLogger.errorFile.Close()
	}
	if defaultLogger.debugFile != nil {
		defaultLogger.debugFile.Close()
	}
}
