//go:build windows

package services

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/atticus6/echPlus/apps/desktop/logger"
)

const (
	regPath = `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
)

// SetSOCKS5Proxy 设置 SOCKS5 系统代理 (Windows)
// Windows 原生不直接支持 SOCKS5 系统代理，这里通过注册表设置代理
// 注意：Windows IE/系统代理主要支持 HTTP 代理，SOCKS5 需要应用程序单独支持
func (p *ProxyServerDesktop) SetSOCKS5Proxy(config ProxyConfig) error {
	proxyAddr := fmt.Sprintf("socks=%s:%s", config.Host, config.Port)

	// 启用代理
	cmd := exec.Command("reg", "add", regPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启用代理失败: %w", err)
	}

	// 设置代理服务器地址
	cmd = exec.Command("reg", "add", regPath, "/v", "ProxyServer", "/t", "REG_SZ", "/d", proxyAddr, "/f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("设置代理服务器失败: %w", err)
	}

	// 刷新系统代理设置
	p.refreshProxySettings()

	logger.Info("✓ 已设置 Windows SOCKS5 代理: %s:%s\n", config.Host, config.Port)
	return nil
}

// DisableSOCKS5Proxy 禁用 SOCKS5 系统代理 (Windows)
func (p *ProxyServerDesktop) DisableSOCKS5Proxy() error {
	// 禁用代理
	cmd := exec.Command("reg", "add", regPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("禁用代理失败: %w", err)
	}

	// 刷新系统代理设置
	p.refreshProxySettings()

	logger.Info("✓ 已禁用 Windows SOCKS5 代理\n")
	return nil
}

// refreshProxySettings 刷新系统代理设置，使更改立即生效
func (p *ProxyServerDesktop) refreshProxySettings() {
	// 使用 PowerShell 刷新代理设置
	script := `
$signature = @'
[DllImport("wininet.dll", SetLastError = true, CharSet=CharSet.Auto)]
public static extern bool InternetSetOption(IntPtr hInternet, int dwOption, IntPtr lpBuffer, int dwBufferLength);
'@
$type = Add-Type -MemberDefinition $signature -Name WinINet -Namespace PInvoke -PassThru
$INTERNET_OPTION_SETTINGS_CHANGED = 39
$INTERNET_OPTION_REFRESH = 37
$type::InternetSetOption([IntPtr]::Zero, $INTERNET_OPTION_SETTINGS_CHANGED, [IntPtr]::Zero, 0) | Out-Null
$type::InternetSetOption([IntPtr]::Zero, $INTERNET_OPTION_REFRESH, [IntPtr]::Zero, 0) | Out-Null
`
	cmd := exec.Command("powershell", "-Command", strings.TrimSpace(script))
	cmd.Run()
}

// GetNetworkServices Windows 不需要此方法，返回空
func (p *ProxyServerDesktop) GetNetworkServices() ([]string, error) {
	return []string{"Windows Internet Settings"}, nil
}

// SetSOCKS5ForService Windows 不需要此方法
func (p *ProxyServerDesktop) SetSOCKS5ForService(service string, config ProxyConfig) error {
	return p.SetSOCKS5Proxy(config)
}

// DisableSOCKS5ForService Windows 不需要此方法
func (p *ProxyServerDesktop) DisableSOCKS5ForService(service string) error {
	return p.DisableSOCKS5Proxy()
}
