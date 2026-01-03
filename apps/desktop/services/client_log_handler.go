package services

import (
	"github.com/atticus6/echPlus/apps/desktop/logger"
)

// ClientLogHandler 实现 client 的 LogHandler 接口
type ClientLogHandler struct{}

func (h *ClientLogHandler) Info(msg string) {
	logger.Info("[Client] %s", msg)
}

func (h *ClientLogHandler) Error(msg string) {
	logger.Error("[Client] %s", msg)
}

func (h *ClientLogHandler) Debug(msg string) {
	logger.Debug("[Client] %s", msg)
}
