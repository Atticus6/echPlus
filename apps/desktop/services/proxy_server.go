package services

import (
	"fmt"

	"github.com/atticus6/echPlus/apps/client/core"
	"github.com/atticus6/echPlus/apps/desktop/config"
	"github.com/atticus6/echPlus/apps/desktop/database"
	"github.com/atticus6/echPlus/apps/desktop/logger"
	"github.com/atticus6/echPlus/apps/desktop/models"
)

var s *core.ProxyServer

func init() {
	s = core.NewProxyServer(config.ConfigState.GetproxyConfig())
}

type ProxyServerDesktop struct {
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Host string
	Port string
}

func (p *ProxyServerDesktop) Start() (err error) {

	if s.GetConfig().ServerAddr == "" {
		p.SwitchNode(config.ConfigState.SelectNodeId)
	}

	err = s.Start()
	if err != nil {
		logger.Error("%s", err)
		return
	}

	err = p.SetSOCKS5Proxy(ProxyConfig{
		Host: config.ConfigState.ListenAddr,
		Port: fmt.Sprint(config.ConfigState.ListenPort),
	})
	if err != nil {
		logger.Error("%s", err)
	}
	return
}

func (p *ProxyServerDesktop) Stop() (err error) {
	err = s.Stop()
	if err != nil {
		logger.Error("%s", err.Error())
	}
	err = p.DisableSOCKS5Proxy()
	return
}

func (p *ProxyServerDesktop) SwitchNode(nodeId int64) {
	if nodeId == 0 {
		return
	}
	var node models.Node
	if err := database.GetDB().Find(&node, nodeId).Error; err != nil {
		logger.Error("节点不存在")
		return
	}
	config.ConfigState.SelectNodeId = nodeId
	orgionConfig := s.GetConfig()
	orgionConfig.Token = node.Token
	orgionConfig.ServerAddr = fmt.Sprintf("%s:%d", node.Address, node.Port)
	orgionConfig.ServerIP = node.ServerIP
	err := s.UpdateConfig(orgionConfig)
	if err != nil {
		logger.Error("%s", err.Error())
	}

}

func (p *ProxyServerDesktop) IsRunning() bool {
	return s.IsRunning()
}

var ProxyServerInstance = ProxyServerDesktop{}
