//go:build darwin

package services

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/atticus6/echPlus/apps/desktop/logger"
)

// GetNetworkServices 获取所有网络服务 (macOS)
func (p *ProxyServerDesktop) GetNetworkServices() ([]string, error) {
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var services []string

	// 跳过第一行（标题）和空行
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if i == 0 || line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		services = append(services, line)
	}

	return services, nil
}

// SetSOCKS5ForService 为指定网络服务设置 SOCKS5 代理 (macOS)
func (p *ProxyServerDesktop) SetSOCKS5ForService(service string, config ProxyConfig) error {
	// 设置 SOCKS5 代理服务器
	cmd := exec.Command("networksetup", "-setsocksfirewallproxy", service, config.Host, config.Port)
	if err := cmd.Run(); err != nil {
		return err
	}

	// 启用 SOCKS5 代理
	cmd = exec.Command("networksetup", "-setsocksfirewallproxystate", service, "on")
	return cmd.Run()
}

// SetSOCKS5Proxy 设置 SOCKS5 系统代理 (macOS)
func (p *ProxyServerDesktop) SetSOCKS5Proxy(config ProxyConfig) error {
	services, err := p.GetNetworkServices()
	if err != nil {
		return fmt.Errorf("获取网络服务失败: %w", err)
	}

	for _, service := range services {
		if err := p.SetSOCKS5ForService(service, config); err != nil {
			logger.Info("为 %s 设置代理失败: %v\n", service, err)
			continue
		}
		logger.Info("✓ 已为 %s 设置 SOCKS5 代理\n", service)
	}

	return nil
}

// DisableSOCKS5Proxy 禁用 SOCKS5 系统代理 (macOS)
func (p *ProxyServerDesktop) DisableSOCKS5Proxy() error {
	services, err := p.GetNetworkServices()
	if err != nil {
		return fmt.Errorf("获取网络服务失败: %w", err)
	}

	for _, service := range services {
		if err := p.DisableSOCKS5ForService(service); err != nil {
			fmt.Printf("为 %s 禁用代理失败: %v\n", service, err)
			continue
		}
		fmt.Printf("✓ 已为 %s 禁用 SOCKS5 代理\n", service)
	}

	return nil
}

// DisableSOCKS5ForService 为指定网络服务禁用 SOCKS5 代理 (macOS)
func (p *ProxyServerDesktop) DisableSOCKS5ForService(service string) error {
	cmd := exec.Command("networksetup", "-setsocksfirewallproxystate", service, "off")
	return cmd.Run()
}
