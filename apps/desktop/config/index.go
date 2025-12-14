package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/atticus6/echPlus/apps/client/core"
)

type ConfigType struct {
	ListenAddr   string
	ListenPort   int64
	DNSServer    string
	ECHDomain    string
	RoutingMode  core.RoutingMode
	SelectNodeId int64
}

var StoreDir string
var configPath string
var ConfigState ConfigType

var defaultConfig = ConfigType{
	ListenAddr:  "0.0.0.0",
	ListenPort:  33255,
	DNSServer:   "dns.alidns.com/dns-query",
	ECHDomain:   "cloudflare-ech.com",
	RoutingMode: core.RoutingModeGlobal,
}

func init() {
	homeDir, err2 := os.UserHomeDir()
	if err2 != nil {
		log.Fatal("无法获取用户目录:", err2)
	}
	StoreDir = filepath.Join(homeDir, ".echplus")

	configPath = filepath.Join(StoreDir, "config.json")

	data, err := os.ReadFile(configPath)

	if err != nil {
		log.Printf("配置文件不存在，使用默认配置: %s", configPath)

		ConfigState = defaultConfig
	} else {
		if err := json.Unmarshal(data, &ConfigState); err != nil {
			log.Printf("解析配置文件失败: %v，使用默认配置", err)
			ConfigState = defaultConfig
		}
	}
	log.Printf("已从文件加载配置: %s", configPath)
}

func (d *ConfigType) GetproxyConfig() core.Config {
	return core.Config{
		ListenAddr:  fmt.Sprintf("%s:%d", d.ListenAddr, d.ListenPort),
		DNSServer:   d.DNSServer,
		RoutingMode: d.RoutingMode,
		ECHDomain:   d.ECHDomain,
		StoreDir:    StoreDir,
	}
}

func (d *ConfigType) SaveConfig() (err error) {
	data, err2 := json.MarshalIndent(d, "", "  ")
	if err2 != nil {
		err = err2
		return
	}
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}
	return
}

func (d *ConfigType) GetValue() ConfigType {
	return *d
}
