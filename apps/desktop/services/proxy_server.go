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
	// 设置 client 日志处理器，将日志输出到 desktop
	core.SetLogHandler(&ClientLogHandler{})
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

// GetTrafficStats 获取流量统计
func (p *ProxyServerDesktop) GetTrafficStats() *TrafficStatsResponse {
	stats := s.GetTrafficStats()
	if stats == nil {
		return &TrafficStatsResponse{}
	}
	
	upload, download := stats.GetTotalStats()
	uploadSpeed, downloadSpeed := stats.GetSpeed()
	topSites := stats.GetTopSites(10)
	
	sites := make([]SiteStatsResponse, 0, len(topSites))
	for _, site := range topSites {
		sites = append(sites, SiteStatsResponse{
			Host:        site.Host,
			Upload:      site.Upload,
			Download:    site.Download,
			Connections: site.Connections,
		})
	}
	
	return &TrafficStatsResponse{
		TotalUpload:   upload,
		TotalDownload: download,
		UploadSpeed:   uploadSpeed,
		DownloadSpeed: downloadSpeed,
		Sites:         sites,
	}
}

// TrafficStatsResponse 流量统计响应
type TrafficStatsResponse struct {
	TotalUpload   int64               `json:"totalUpload"`
	TotalDownload int64               `json:"totalDownload"`
	UploadSpeed   int64               `json:"uploadSpeed"`   // bytes/s
	DownloadSpeed int64               `json:"downloadSpeed"` // bytes/s
	Sites         []SiteStatsResponse `json:"sites"`
}

// SiteStatsResponse 站点统计响应
type SiteStatsResponse struct {
	Host        string `json:"host"`
	Upload      int64  `json:"upload"`
	Download    int64  `json:"download"`
	Connections int64  `json:"connections"`
}

var ProxyServerInstance = ProxyServerDesktop{}
