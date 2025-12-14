//go:build linux

package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/atticus6/echPlus/apps/desktop/logger"
)

// SetSOCKS5Proxy 设置 SOCKS5 系统代理 (Linux)
// 支持 GNOME (gsettings) 和环境变量方式
func (p *ProxyServerDesktop) SetSOCKS5Proxy(config ProxyConfig) error {
	// 尝试使用 gsettings (GNOME)
	if p.hasGSettings() {
		if err := p.setGnomeProxy(config); err != nil {
			logger.Info("GNOME 代理设置失败: %v，尝试其他方式\n", err)
		} else {
			logger.Info("✓ 已通过 GNOME 设置 SOCKS5 代理\n")
			return nil
		}
	}

	// 尝试使用 KDE 设置
	if p.hasKDE() {
		if err := p.setKDEProxy(config); err != nil {
			logger.Info("KDE 代理设置失败: %v\n", err)
		} else {
			logger.Info("✓ 已通过 KDE 设置 SOCKS5 代理\n")
			return nil
		}
	}

	// 设置环境变量（写入 profile）
	if err := p.setEnvProxy(config); err != nil {
		return fmt.Errorf("设置环境变量代理失败: %w", err)
	}

	logger.Info("✓ 已设置 Linux SOCKS5 代理: %s:%s\n", config.Host, config.Port)
	return nil
}

// DisableSOCKS5Proxy 禁用 SOCKS5 系统代理 (Linux)
func (p *ProxyServerDesktop) DisableSOCKS5Proxy() error {
	// 禁用 GNOME 代理
	if p.hasGSettings() {
		p.disableGnomeProxy()
	}

	// 禁用 KDE 代理
	if p.hasKDE() {
		p.disableKDEProxy()
	}

	// 清除环境变量
	p.clearEnvProxy()

	logger.Info("✓ 已禁用 Linux SOCKS5 代理\n")
	return nil
}

// hasGSettings 检查是否有 gsettings 命令
func (p *ProxyServerDesktop) hasGSettings() bool {
	_, err := exec.LookPath("gsettings")
	return err == nil
}

// hasKDE 检查是否是 KDE 环境
func (p *ProxyServerDesktop) hasKDE() bool {
	desktop := os.Getenv("XDG_CURRENT_DESKTOP")
	return strings.Contains(strings.ToLower(desktop), "kde")
}

// setGnomeProxy 设置 GNOME 代理
func (p *ProxyServerDesktop) setGnomeProxy(config ProxyConfig) error {
	commands := [][]string{
		{"gsettings", "set", "org.gnome.system.proxy", "mode", "manual"},
		{"gsettings", "set", "org.gnome.system.proxy.socks", "host", config.Host},
		{"gsettings", "set", "org.gnome.system.proxy.socks", "port", config.Port},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// disableGnomeProxy 禁用 GNOME 代理
func (p *ProxyServerDesktop) disableGnomeProxy() error {
	cmd := exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none")
	return cmd.Run()
}

// setKDEProxy 设置 KDE 代理
func (p *ProxyServerDesktop) setKDEProxy(config ProxyConfig) error {
	// KDE 使用 kwriteconfig5 或 kwriteconfig
	kwriteconfig := "kwriteconfig5"
	if _, err := exec.LookPath(kwriteconfig); err != nil {
		kwriteconfig = "kwriteconfig"
	}

	commands := [][]string{
		{kwriteconfig, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "1"},
		{kwriteconfig, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "socksProxy", fmt.Sprintf("socks://%s:%s", config.Host, config.Port)},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// disableKDEProxy 禁用 KDE 代理
func (p *ProxyServerDesktop) disableKDEProxy() error {
	kwriteconfig := "kwriteconfig5"
	if _, err := exec.LookPath(kwriteconfig); err != nil {
		kwriteconfig = "kwriteconfig"
	}

	cmd := exec.Command(kwriteconfig, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "0")
	return cmd.Run()
}

// setEnvProxy 设置环境变量代理
func (p *ProxyServerDesktop) setEnvProxy(config ProxyConfig) error {
	proxyURL := fmt.Sprintf("socks5://%s:%s", config.Host, config.Port)

	// 设置当前进程的环境变量
	os.Setenv("ALL_PROXY", proxyURL)
	os.Setenv("all_proxy", proxyURL)

	// 写入 ~/.profile 或 ~/.bashrc
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	profilePath := filepath.Join(homeDir, ".profile")
	content := fmt.Sprintf("\n# Proxy settings (added by EchPlus)\nexport ALL_PROXY=%s\nexport all_proxy=%s\n", proxyURL, proxyURL)

	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	return err
}

// clearEnvProxy 清除环境变量代理
func (p *ProxyServerDesktop) clearEnvProxy() {
	os.Unsetenv("ALL_PROXY")
	os.Unsetenv("all_proxy")
}

// GetNetworkServices Linux 返回桌面环境信息
func (p *ProxyServerDesktop) GetNetworkServices() ([]string, error) {
	var services []string

	if p.hasGSettings() {
		services = append(services, "GNOME")
	}
	if p.hasKDE() {
		services = append(services, "KDE")
	}
	services = append(services, "Environment Variables")

	return services, nil
}

// SetSOCKS5ForService Linux 不需要此方法
func (p *ProxyServerDesktop) SetSOCKS5ForService(service string, config ProxyConfig) error {
	return p.SetSOCKS5Proxy(config)
}

// DisableSOCKS5ForService Linux 不需要此方法
func (p *ProxyServerDesktop) DisableSOCKS5ForService(service string) error {
	return p.DisableSOCKS5Proxy()
}
