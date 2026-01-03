package core

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Config 代理客户端配置
type Config struct {
	ListenAddr  string
	ServerAddr  string
	ServerIP    string
	Token       string
	DNSServer   string
	ECHDomain   string
	RoutingMode RoutingMode
	StoreDir    string
}

// ProxyServer 代理服务器
type ProxyServer struct {
	config   Config
	listener net.Listener
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.RWMutex

	echListMu         sync.RWMutex
	echList           []byte
	chinaIPRangesMu   sync.RWMutex
	chinaIPRanges     []ipRange
	chinaIPV6RangesMu sync.RWMutex
	chinaIPV6Ranges   []ipRangeV6

	dohProxyClientMu   sync.RWMutex
	dohProxyClient     *http.Client
	dohProxyClientPort string

	// 流量统计
	trafficStats *TrafficStats
}

type ipRange struct {
	start uint32
	end   uint32
}

type ipRangeV6 struct {
	start [16]byte
	end   [16]byte
}

const (
	modeSOCKS5      = 1
	modeHTTPConnect = 2
	modeHTTPProxy   = 3
	typeHTTPS       = 65
)

// RoutingMode 路由模式常量
type RoutingMode string

const (
	RoutingModeGlobal   RoutingMode = "global"    // 全局代理
	RoutingModeBypassCN RoutingMode = "bypass_cn" // 跳过中国大陆
	RoutingModeNone     RoutingMode = "none"      // 直连模式
)

// HTTP 客户端配置常量
const (
	defaultHTTPTimeout = 30 * time.Second
	dohTimeout         = 10 * time.Second
	dialTimeout        = 10 * time.Second
	handshakeTimeout   = 10 * time.Second
	connectionDeadline = 30 * time.Second
	pingInterval       = 10 * time.Second
	readBufferSize     = 32768
	maxContentLength   = 10 * 1024 * 1024
)

var defaultHTTPClient = &http.Client{
	Timeout: defaultHTTPTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConnsPerHost: 5,
	},
}

var dohClient = &http.Client{
	Timeout: dohTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        5,
		IdleConnTimeout:     60 * time.Second,
		MaxIdleConnsPerHost: 2,
	},
}

// NewProxyServer 创建新的代理服务器
func NewProxyServer(cfg Config) *ProxyServer {
	ts := NewTrafficStats(cfg.StoreDir)
	upload, download := ts.GetTotalStats()
	if upload > 0 || download > 0 {
		LogInfo("[统计] 已加载历史流量统计: ↑ %s  ↓ %s", FormatBytes(upload), FormatBytes(download))
	}
	return &ProxyServer{
		config:       cfg,
		stopChan:     make(chan struct{}),
		trafficStats: ts,
	}
}

// Start 启动代理服务器
func (s *ProxyServer) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("服务器已在运行")
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	LogInfo("[启动] 正在获取 ECH 配置...")
	if err := s.prepareECH(); err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return fmt.Errorf("获取 ECH 配置失败: %w", err)
	}

	if err := s.loadRoutingData(); err != nil {
		LogError("[警告] 加载分流数据失败: %v", err)
	}

	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return fmt.Errorf("监听失败: %w", err)
	}
	s.listener = listener

	LogInfo("[代理] 服务器启动: %s (支持 SOCKS5 和 HTTP)", s.config.ListenAddr)
	LogInfo("[代理] 后端服务器: %s", s.config.ServerAddr)
	if s.config.ServerIP == "" {
		s.config.ServerIP = "www.visa.com"
	}

	LogInfo("[代理] 使用固定 IP: %s", s.config.ServerIP)
	s.wg.Add(1)
	go s.acceptLoop()

	// 启动定期保存流量统计
	go s.autoSaveStats()

	return nil
}

// Stop 停止代理服务器
func (s *ProxyServer) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return errors.New("服务器未运行")
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopChan)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()

	// 保存流量统计
	if s.trafficStats != nil {
		if err := s.trafficStats.Save(); err != nil {
			LogError("[统计] 保存流量统计失败: %v", err)
		} else {
			upload, download := s.trafficStats.GetTotalStats()
			LogInfo("[统计] 流量统计已保存: ↑ %s  ↓ %s", FormatBytes(upload), FormatBytes(download))
		}
	}

	LogInfo("[代理] 服务器已停止")
	return nil
}

// Restart 重启代理服务器
func (s *ProxyServer) Restart() error {
	LogInfo("[代理] 正在重启服务器...")
	if err := s.Stop(); err != nil && err.Error() != "服务器未运行" {
		return fmt.Errorf("停止服务器失败: %w", err)
	}
	return s.Start()
}

// UpdateConfig 更新配置并重启
func (s *ProxyServer) UpdateConfig(cfg Config) error {
	s.mu.Lock()
	s.config = cfg
	s.mu.Unlock()

	if s.IsRunning() {
		return s.Restart()
	}
	return nil

}

// IsRunning 检查服务器是否运行中
func (s *ProxyServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetConfig 获取当前配置
func (s *ProxyServer) GetConfig() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetTrafficStats 获取流量统计管理器
func (s *ProxyServer) GetTrafficStats() *TrafficStats {
	return s.trafficStats
}

// autoSaveStats 定期自动保存流量统计
func (s *ProxyServer) autoSaveStats() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			if s.trafficStats != nil {
				s.trafficStats.Save()
			}
		}
	}
}

func (s *ProxyServer) acceptLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stopChan:
			return
		default:
		}
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				return
			default:
				LogError("[代理] 接受连接失败: %v", err)
				continue
			}
		}
		go s.handleConnection(conn)
	}
}

func (s *ProxyServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	clientAddr := conn.RemoteAddr().String()
	conn.SetDeadline(time.Now().Add(connectionDeadline))

	buf := make([]byte, 1)
	if n, err := conn.Read(buf); err != nil || n == 0 {
		return
	}

	switch buf[0] {
	case 0x05:
		s.handleSOCKS5(conn, clientAddr, buf[0])
	case 'C', 'G', 'P', 'H', 'D', 'O', 'T':
		s.handleHTTP(conn, clientAddr, buf[0])
	default:
		LogInfo("[代理] %s 未知协议: 0x%02x", clientAddr, buf[0])
	}
}

func (s *ProxyServer) loadRoutingData() error {
	switch s.config.RoutingMode {
	case RoutingModeBypassCN:
		LogInfo("[启动] 分流模式: 跳过中国大陆，正在加载中国IP列表...")
		ipv4Count, ipv6Count := 0, 0
		if err := s.loadChinaIPList(); err != nil {
			LogError("[警告] 加载中国IPv4列表失败: %v", err)
		} else {
			s.chinaIPRangesMu.RLock()
			ipv4Count = len(s.chinaIPRanges)
			s.chinaIPRangesMu.RUnlock()
		}
		if err := s.loadChinaIPV6List(); err != nil {
			LogError("[警告] 加载中国IPv6列表失败: %v", err)
		} else {
			s.chinaIPV6RangesMu.RLock()
			ipv6Count = len(s.chinaIPV6Ranges)
			s.chinaIPV6RangesMu.RUnlock()
		}
		if ipv4Count > 0 || ipv6Count > 0 {
			LogInfo("[启动] 已加载 %d 个中国IPv4段, %d 个中国IPv6段", ipv4Count, ipv6Count)
		} else {
			LogError("[警告] 未加载到任何中国IP列表，将使用默认规则")
		}
	case RoutingModeGlobal:
		LogInfo("[启动] 分流模式: 全局代理")
	case RoutingModeNone:
		LogInfo("[启动] 分流模式: 不改变代理（直连模式）")
	default:
		LogError("[警告] 未知的分流模式: %s，使用默认模式 global", s.config.RoutingMode)
		s.config.RoutingMode = RoutingModeGlobal
	}
	return nil
}

func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func compareIPv6(a, b [16]byte) int {
	for i := 0; i < 16; i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func (s *ProxyServer) isChinaIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	if ip.To4() != nil {
		ipUint32 := ipToUint32(ip)
		if ipUint32 == 0 {
			return false
		}
		s.chinaIPRangesMu.RLock()
		defer s.chinaIPRangesMu.RUnlock()
		left, right := 0, len(s.chinaIPRanges)
		for left < right {
			mid := (left + right) / 2
			r := s.chinaIPRanges[mid]
			if ipUint32 < r.start {
				right = mid
			} else if ipUint32 > r.end {
				left = mid + 1
			} else {
				return true
			}
		}
		return false
	}
	ipBytes := ip.To16()
	if ipBytes == nil {
		return false
	}
	var ipArray [16]byte
	copy(ipArray[:], ipBytes)
	s.chinaIPV6RangesMu.RLock()
	defer s.chinaIPV6RangesMu.RUnlock()
	left, right := 0, len(s.chinaIPV6Ranges)
	for left < right {
		mid := (left + right) / 2
		r := s.chinaIPV6Ranges[mid]
		if compareIPv6(ipArray, r.start) < 0 {
			right = mid
		} else if compareIPv6(ipArray, r.end) > 0 {
			left = mid + 1
		} else {
			return true
		}
	}
	return false
}

// isPrivateIP 检查是否为内网地址
func (s *ProxyServer) isPrivateIP(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return isPrivateIPAddress(ip)
	}
	// 如果不是IP地址，尝试解析域名
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return false
	}
	// 检查解析出的所有IP是否都是内网地址
	for _, resolvedIP := range ips {
		if !isPrivateIPAddress(resolvedIP) {
			return false
		}
	}
	return true
}

// isPrivateIPAddress 检查IP地址是否为内网地址（改为包级函数，无需 receiver）
func isPrivateIPAddress(ip net.IP) bool {
	if ip == nil {
		return false
	}
	// 使用标准库方法简化判断
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return true
	}
	// 169.254.0.0/16 (链路本地地址) - IsLinkLocalUnicast 已覆盖 IPv4
	// fc00::/7 (唯一本地地址) - IsPrivate 已覆盖
	return false
}

func (s *ProxyServer) shouldBypassProxy(targetHost string) bool {
	if s.config.RoutingMode == RoutingModeNone {
		return true
	}

	// 检查是否为内网地址，内网地址始终直连
	if s.isPrivateIP(targetHost) {
		LogInfo("[分流] %s 局域网地址，强制直连", targetHost)
		return true
	}

	if s.config.RoutingMode == RoutingModeGlobal {
		return false
	}
	if s.config.RoutingMode == RoutingModeBypassCN {
		if ip := net.ParseIP(targetHost); ip != nil {
			return s.isChinaIP(targetHost)
		}
		ips, err := net.LookupIP(targetHost)
		if err != nil {
			return false
		}
		for _, ip := range ips {
			if s.isChinaIP(ip.String()) {
				return true
			}
		}
		return false
	}
	return false
}

func downloadIPList(urlStr, filePath string) error {
	LogInfo("[下载] 正在下载 IP 列表: %s", urlStr)
	resp, err := defaultHTTPClient.Get(urlStr)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取下载内容失败: %w", err)
	}
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}
	LogInfo("[下载] 已保存到: %s", filePath)
	return nil
}

func (s *ProxyServer) loadChinaIPList() error {

	ipListFile := filepath.Join(s.config.StoreDir, "chn_ip.txt")
	needDownload := false
	if info, err := os.Stat(ipListFile); os.IsNotExist(err) {
		needDownload = true
		LogInfo("[加载] IPv4 列表文件不存在，将自动下载")
	} else if info.Size() == 0 {
		needDownload = true
		LogInfo("[加载] IPv4 列表文件为空，将自动下载")
	}
	if needDownload {
		urlStr := "https://raw.githubusercontent.com/mayaxcn/china-ip-list/refs/heads/master/chn_ip.txt"
		if err := downloadIPList(urlStr, ipListFile); err != nil {
			return fmt.Errorf("自动下载 IPv4 列表失败: %w", err)
		}
	}
	file, err := os.Open(ipListFile)
	if err != nil {
		return fmt.Errorf("打开IP列表文件失败: %w", err)
	}
	defer file.Close()
	var ranges []ipRange
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		startIP := net.ParseIP(parts[0])
		endIP := net.ParseIP(parts[1])
		if startIP == nil || endIP == nil {
			continue
		}
		start := ipToUint32(startIP)
		end := ipToUint32(endIP)
		if start > 0 && end > 0 && start <= end {
			ranges = append(ranges, ipRange{start: start, end: end})
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取IP列表文件失败: %w", err)
	}
	if len(ranges) == 0 {
		return errors.New("IP列表为空")
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	s.chinaIPRangesMu.Lock()
	s.chinaIPRanges = ranges
	s.chinaIPRangesMu.Unlock()
	return nil
}

func (s *ProxyServer) loadChinaIPV6List() error {
	ipListFile := filepath.Join(s.config.StoreDir, "chn_ip_v6.txt")
	// if _, err := os.Stat(ipListFile); os.IsNotExist(err) {
	// 	ipListFile = "chn_ip_v6.txt"
	// }
	needDownload := false
	if info, err := os.Stat(ipListFile); os.IsNotExist(err) {
		needDownload = true
		LogInfo("[加载] IPv6 列表文件不存在，将自动下载")
	} else if info.Size() == 0 {
		needDownload = true
		LogInfo("[加载] IPv6 列表文件为空，将自动下载")
	}
	if needDownload {
		urlStr := "https://raw.githubusercontent.com/mayaxcn/china-ip-list/refs/heads/master/chn_ip_v6.txt"
		if err := downloadIPList(urlStr, ipListFile); err != nil {
			LogError("[警告] 自动下载 IPv6 列表失败: %v，将跳过 IPv6 支持", err)
			return nil
		}
	}
	file, err := os.Open(ipListFile)
	if err != nil {
		LogError("[警告] 打开 IPv6 IP列表文件失败: %v，将跳过 IPv6 支持", err)
		return nil
	}
	defer file.Close()
	var ranges []ipRangeV6
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		startIP := net.ParseIP(parts[0])
		endIP := net.ParseIP(parts[1])
		if startIP == nil || endIP == nil {
			continue
		}
		startBytes := startIP.To16()
		endBytes := endIP.To16()
		if startBytes == nil || endBytes == nil {
			continue
		}
		var start, end [16]byte
		copy(start[:], startBytes)
		copy(end[:], endBytes)
		if compareIPv6(start, end) <= 0 {
			ranges = append(ranges, ipRangeV6{start: start, end: end})
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取IPv6 IP列表文件失败: %w", err)
	}
	if len(ranges) == 0 {
		return nil
	}
	sort.Slice(ranges, func(i, j int) bool { return compareIPv6(ranges[i].start, ranges[j].start) < 0 })
	s.chinaIPV6RangesMu.Lock()
	s.chinaIPV6Ranges = ranges
	s.chinaIPV6RangesMu.Unlock()
	return nil
}

func (s *ProxyServer) prepareECH() error {
	echBase64, err := s.queryHTTPSRecord(s.config.ECHDomain, s.config.DNSServer)
	if err != nil {
		return fmt.Errorf("DNS 查询失败: %w", err)
	}
	if echBase64 == "" {
		return errors.New("未找到 ECH 参数")
	}
	raw, err := base64.StdEncoding.DecodeString(echBase64)
	if err != nil {
		return fmt.Errorf("ECH 解码失败: %w", err)
	}
	s.echListMu.Lock()
	s.echList = raw
	s.echListMu.Unlock()
	LogInfo("[ECH] 配置已加载，长度: %d 字节", len(raw))
	return nil
}

func (s *ProxyServer) refreshECH() error {
	LogInfo("[ECH] 刷新配置...")
	return s.prepareECH()
}

func (s *ProxyServer) getECHList() ([]byte, error) {
	s.echListMu.RLock()
	defer s.echListMu.RUnlock()
	if len(s.echList) == 0 {
		return nil, errors.New("ECH 配置未加载")
	}
	return s.echList, nil
}

func buildTLSConfigWithECH(serverName string, echList []byte) (*tls.Config, error) {
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("加载系统根证书失败: %w", err)
	}
	if len(echList) == 0 {
		return nil, errors.New("ECH 配置为空，这是必需功能")
	}
	config := &tls.Config{MinVersion: tls.VersionTLS13, ServerName: serverName, RootCAs: roots}
	if err := setECHConfig(config, echList); err != nil {
		return nil, fmt.Errorf("设置 ECH 配置失败（需要 Go 1.23+ 或支持 ECH 的版本）: %w", err)
	}
	return config, nil
}

func setECHConfig(config *tls.Config, echList []byte) error {
	configValue := reflect.ValueOf(config).Elem()
	field1 := configValue.FieldByName("EncryptedClientHelloConfigList")
	if !field1.IsValid() || !field1.CanSet() {
		return fmt.Errorf("EncryptedClientHelloConfigList 字段不可用，需要 Go 1.23+ 版本")
	}
	field1.Set(reflect.ValueOf(echList))
	field2 := configValue.FieldByName("EncryptedClientHelloRejectionVerify")
	if !field2.IsValid() || !field2.CanSet() {
		return fmt.Errorf("EncryptedClientHelloRejectionVerify 字段不可用，需要 Go 1.23+ 版本")
	}
	rejectionFunc := func(cs tls.ConnectionState) error { return errors.New("服务器拒绝 ECH") }
	field2.Set(reflect.ValueOf(rejectionFunc))
	return nil
}

func (s *ProxyServer) queryHTTPSRecord(domain, dnsServer string) (string, error) {
	dohURL := dnsServer
	if !strings.HasPrefix(dohURL, "https://") && !strings.HasPrefix(dohURL, "http://") {
		dohURL = "https://" + dohURL
	}
	return queryDoH(domain, dohURL)
}

func queryDoH(domain, dohURL string) (string, error) {
	u, err := url.Parse(dohURL)
	if err != nil {
		return "", fmt.Errorf("无效的 DoH URL: %v", err)
	}
	dnsQuery := buildDNSQuery(domain, typeHTTPS)
	dnsBase64 := base64.RawURLEncoding.EncodeToString(dnsQuery)
	q := u.Query()
	q.Set("dns", dnsBase64)
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/dns-message")
	req.Header.Set("Content-Type", "application/dns-message")
	resp, err := dohClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("DoH 请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DoH 服务器返回错误: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 DoH 响应失败: %v", err)
	}
	return parseDNSResponse(body)
}

func buildDNSQuery(domain string, qtype uint16) []byte {
	query := make([]byte, 0, 512)
	query = append(query, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00)
	for _, label := range strings.Split(domain, ".") {
		query = append(query, byte(len(label)))
		query = append(query, []byte(label)...)
	}
	query = append(query, 0x00, byte(qtype>>8), byte(qtype), 0x00, 0x01)
	return query
}

func parseDNSResponse(response []byte) (string, error) {
	if len(response) < 12 {
		return "", errors.New("响应过短")
	}
	ancount := binary.BigEndian.Uint16(response[6:8])
	if ancount == 0 {
		return "", errors.New("无应答记录")
	}
	offset := 12
	for offset < len(response) && response[offset] != 0 {
		offset += int(response[offset]) + 1
	}
	offset += 5
	for i := 0; i < int(ancount); i++ {
		if offset >= len(response) {
			break
		}
		if response[offset]&0xC0 == 0xC0 {
			offset += 2
		} else {
			for offset < len(response) && response[offset] != 0 {
				offset += int(response[offset]) + 1
			}
			offset++
		}
		if offset+10 > len(response) {
			break
		}
		rrType := binary.BigEndian.Uint16(response[offset : offset+2])
		offset += 8
		dataLen := binary.BigEndian.Uint16(response[offset : offset+2])
		offset += 2
		if offset+int(dataLen) > len(response) {
			break
		}
		data := response[offset : offset+int(dataLen)]
		offset += int(dataLen)
		if rrType == typeHTTPS {
			if ech := parseHTTPSRecord(data); ech != "" {
				return ech, nil
			}
		}
	}
	return "", nil
}

func parseHTTPSRecord(data []byte) string {
	if len(data) < 2 {
		return ""
	}
	offset := 2
	if offset < len(data) && data[offset] == 0 {
		offset++
	} else {
		for offset < len(data) && data[offset] != 0 {
			offset += int(data[offset]) + 1
		}
		offset++
	}
	for offset+4 <= len(data) {
		key := binary.BigEndian.Uint16(data[offset : offset+2])
		length := binary.BigEndian.Uint16(data[offset+2 : offset+4])
		offset += 4
		if offset+int(length) > len(data) {
			break
		}
		value := data[offset : offset+int(length)]
		offset += int(length)
		if key == 5 {
			return base64.StdEncoding.EncodeToString(value)
		}
	}
	return ""
}

func (s *ProxyServer) getDoHProxyClient(port string) (*http.Client, error) {
	s.dohProxyClientMu.RLock()
	if s.dohProxyClient != nil && s.dohProxyClientPort == port {
		client := s.dohProxyClient
		s.dohProxyClientMu.RUnlock()
		return client, nil
	}
	s.dohProxyClientMu.RUnlock()

	s.dohProxyClientMu.Lock()
	defer s.dohProxyClientMu.Unlock()

	// Double-check after acquiring write lock
	if s.dohProxyClient != nil && s.dohProxyClientPort == port {
		return s.dohProxyClient, nil
	}

	echBytes, err := s.getECHList()
	if err != nil {
		return nil, fmt.Errorf("获取 ECH 配置失败: %w", err)
	}

	tlsCfg, err := buildTLSConfigWithECH("cloudflare-dns.com", echBytes)
	if err != nil {
		return nil, fmt.Errorf("构建 TLS 配置失败: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig:     tlsCfg,
		MaxIdleConns:        5,
		IdleConnTimeout:     60 * time.Second,
		MaxIdleConnsPerHost: 2,
	}
	if s.config.ServerIP != "" {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, p, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			dialer := &net.Dialer{Timeout: dialTimeout}
			return dialer.DialContext(ctx, network, net.JoinHostPort(s.config.ServerIP, p))
		}
	}

	s.dohProxyClient = &http.Client{Transport: transport, Timeout: dohTimeout}
	s.dohProxyClientPort = port
	return s.dohProxyClient, nil
}

func (s *ProxyServer) queryDoHForProxy(dnsQuery []byte) ([]byte, error) {
	_, port, _, err := s.parseServerAddr()
	if err != nil {
		return nil, err
	}
	client, err := s.getDoHProxyClient(port)
	if err != nil {
		return nil, err
	}
	dohURL := fmt.Sprintf("https://cloudflare-dns.com:%s/dns-query", port)
	req, err := http.NewRequest("POST", dohURL, bytes.NewReader(dnsQuery))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("DoH 请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DoH 响应错误: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (s *ProxyServer) parseServerAddr() (host, port, path string, err error) {
	addr := s.config.ServerAddr
	path = "/"
	slashIdx := strings.Index(addr, "/")
	if slashIdx != -1 {
		path = addr[slashIdx:]
		addr = addr[:slashIdx]
	}
	host, port, err = net.SplitHostPort(addr)
	if err != nil {
		return "", "", "", fmt.Errorf("无效的服务器地址格式: %v", err)
	}
	return host, port, path, nil
}

func (s *ProxyServer) dialWebSocketWithECH(maxRetries int) (*websocket.Conn, error) {
	host, port, path, err := s.parseServerAddr()
	if err != nil {
		return nil, err
	}
	wsURL := fmt.Sprintf("wss://%s:%s%s", host, port, path)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		echBytes, echErr := s.getECHList()
		if echErr != nil {
			if attempt < maxRetries {
				s.refreshECH()
				continue
			}
			return nil, echErr
		}

		tlsCfg, tlsErr := buildTLSConfigWithECH(host, echBytes)
		if tlsErr != nil {
			return nil, tlsErr
		}

		dialer := websocket.Dialer{
			TLSClientConfig:  tlsCfg,
			HandshakeTimeout: handshakeTimeout,
		}
		if s.config.Token != "" {
			dialer.Subprotocols = []string{s.config.Token}
		}
		if s.config.ServerIP != "" {
			dialer.NetDial = func(network, address string) (net.Conn, error) {
				_, p, err := net.SplitHostPort(address)
				if err != nil {
					return nil, err
				}
				return net.DialTimeout(network, net.JoinHostPort(s.config.ServerIP, p), dialTimeout)
			}
		}

		wsConn, _, dialErr := dialer.Dial(wsURL, nil)
		if dialErr != nil {
			if strings.Contains(dialErr.Error(), "ECH") && attempt < maxRetries {
				LogInfo("[ECH] 连接失败，尝试刷新配置 (%d/%d)", attempt, maxRetries)
				s.refreshECH()
				time.Sleep(time.Second)
				continue
			}
			return nil, dialErr
		}
		return wsConn, nil
	}
	return nil, errors.New("连接失败，已达最大重试次数")
}

func isNormalCloseError(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		return true
	}
	errStr := err.Error()
	return strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "normal closure")
}

func (s *ProxyServer) handleSOCKS5(conn net.Conn, clientAddr string, firstByte byte) {
	if firstByte != 0x05 {
		LogInfo("[SOCKS5] %s 版本错误: 0x%02x", clientAddr, firstByte)
		return
	}
	buf := make([]byte, 1)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}
	nmethods := buf[0]
	methods := make([]byte, nmethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}
	buf = make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}
	if buf[0] != 5 {
		return
	}
	command := buf[1]
	atyp := buf[3]
	var host string
	switch atyp {
	case 0x01:
		buf = make([]byte, 4)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		host = net.IP(buf).String()
	case 0x03:
		buf = make([]byte, 1)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		domainBuf := make([]byte, buf[0])
		if _, err := io.ReadFull(conn, domainBuf); err != nil {
			return
		}
		host = string(domainBuf)
	case 0x04:
		buf = make([]byte, 16)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		host = net.IP(buf).String()
	default:
		conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}
	buf = make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}
	port := int(buf[0])<<8 | int(buf[1])
	switch command {
	case 0x01:
		var target string
		if atyp == 0x04 {
			target = fmt.Sprintf("[%s]:%d", host, port)
		} else {
			target = fmt.Sprintf("%s:%d", host, port)
		}
		LogInfo("[SOCKS5] %s -> %s", clientAddr, target)
		if err := s.handleTunnel(conn, target, clientAddr, modeSOCKS5, ""); err != nil {
			if !isNormalCloseError(err) {
				LogError("[SOCKS5] %s 代理失败: %v", clientAddr, err)
			}
		}
	case 0x03:
		s.handleUDPAssociate(conn, clientAddr)
	default:
		conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	}
}

func (s *ProxyServer) handleUDPAssociate(tcpConn net.Conn, clientAddr string) {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		LogError("[UDP] %s 解析地址失败: %v", clientAddr, err)
		tcpConn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		LogError("[UDP] %s 监听失败: %v", clientAddr, err)
		tcpConn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}
	localAddr := udpConn.LocalAddr().(*net.UDPAddr)
	port := localAddr.Port
	LogInfo("[UDP] %s UDP ASSOCIATE 监听端口: %d", clientAddr, port)
	response := []byte{0x05, 0x00, 0x00, 0x01}
	response = append(response, 127, 0, 0, 1)
	response = append(response, byte(port>>8), byte(port&0xff))
	if _, err := tcpConn.Write(response); err != nil {
		udpConn.Close()
		return
	}
	stopChan := make(chan struct{})
	go s.handleUDPRelay(udpConn, clientAddr, stopChan)
	buf := make([]byte, 1)
	tcpConn.Read(buf)
	close(stopChan)
	udpConn.Close()
	LogInfo("[UDP] %s UDP ASSOCIATE 连接关闭", clientAddr)
}

func (s *ProxyServer) handleUDPRelay(udpConn *net.UDPConn, clientAddr string, stopChan chan struct{}) {
	buf := make([]byte, 65535)
	for {
		select {
		case <-stopChan:
			return
		default:
		}
		udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}
		if n < 10 {
			continue
		}
		data := buf[:n]
		if data[2] != 0x00 {
			continue
		}
		atyp := data[3]
		var headerLen, dstPort int
		var dstHost string
		switch atyp {
		case 0x01:
			if n < 10 {
				continue
			}
			dstHost = net.IP(data[4:8]).String()
			dstPort = int(data[8])<<8 | int(data[9])
			headerLen = 10
		case 0x03:
			if n < 5 {
				continue
			}
			domainLen := int(data[4])
			if n < 7+domainLen {
				continue
			}
			dstHost = string(data[5 : 5+domainLen])
			dstPort = int(data[5+domainLen])<<8 | int(data[6+domainLen])
			headerLen = 7 + domainLen
		case 0x04:
			if n < 22 {
				continue
			}
			dstHost = net.IP(data[4:20]).String()
			dstPort = int(data[20])<<8 | int(data[21])
			headerLen = 22
		default:
			continue
		}
		udpData := data[headerLen:]
		target := fmt.Sprintf("%s:%d", dstHost, dstPort)
		if dstPort == 53 {
			LogInfo("[UDP-DNS] %s -> %s (DoH 查询)", clientAddr, target)
			go s.handleDNSQuery(udpConn, addr, udpData, data[:headerLen])
		} else {
			LogInfo("[UDP] %s -> %s (暂不支持非 DNS UDP)", clientAddr, target)
		}
	}
}

func (s *ProxyServer) handleDNSQuery(udpConn *net.UDPConn, clientAddr *net.UDPAddr, dnsQuery []byte, socks5Header []byte) {
	dnsResponse, err := s.queryDoHForProxy(dnsQuery)
	if err != nil {
		LogError("[UDP-DNS] DoH 查询失败: %v", err)
		return
	}
	response := make([]byte, 0, len(socks5Header)+len(dnsResponse))
	response = append(response, socks5Header...)
	response = append(response, dnsResponse...)
	_, err = udpConn.WriteToUDP(response, clientAddr)
	if err != nil {
		LogError("[UDP-DNS] 发送响应失败: %v", err)
		return
	}
	LogInfo("[UDP-DNS] DoH 查询成功，响应 %d 字节", len(dnsResponse))
}

func (s *ProxyServer) handleHTTP(conn net.Conn, clientAddr string, firstByte byte) {
	reader := bufio.NewReader(io.MultiReader(strings.NewReader(string(firstByte)), conn))
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	parts := strings.Fields(requestLine)
	if len(parts) < 3 {
		return
	}
	method := parts[0]
	requestURL := parts[1]
	httpVersion := parts[2]
	headers := make(map[string]string)
	var headerLines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		headerLines = append(headerLines, line)
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			headers[strings.ToLower(key)] = value
		}
	}
	switch method {
	case "CONNECT":
		LogInfo("[HTTP-CONNECT] %s -> %s", clientAddr, requestURL)
		if err := s.handleTunnel(conn, requestURL, clientAddr, modeHTTPConnect, ""); err != nil {
			if !isNormalCloseError(err) {
				LogError("[HTTP-CONNECT] %s 代理失败: %v", clientAddr, err)
			}
		}
	case "GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH", "TRACE":
		LogInfo("[HTTP-%s] %s -> %s", method, clientAddr, requestURL)
		var target, path string
		if strings.HasPrefix(requestURL, "http://") {
			urlWithoutScheme := strings.TrimPrefix(requestURL, "http://")
			idx := strings.Index(urlWithoutScheme, "/")
			if idx > 0 {
				target = urlWithoutScheme[:idx]
				path = urlWithoutScheme[idx:]
			} else {
				target = urlWithoutScheme
				path = "/"
			}
		} else {
			target = headers["host"]
			path = requestURL
		}
		if target == "" {
			conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			return
		}
		if !strings.Contains(target, ":") {
			target += ":80"
		}
		var requestBuilder strings.Builder
		requestBuilder.WriteString(fmt.Sprintf("%s %s %s\r\n", method, path, httpVersion))
		for _, line := range headerLines {
			key := strings.Split(line, ":")[0]
			keyLower := strings.ToLower(strings.TrimSpace(key))
			if keyLower != "proxy-connection" && keyLower != "proxy-authorization" {
				requestBuilder.WriteString(line)
				requestBuilder.WriteString("\r\n")
			}
		}
		requestBuilder.WriteString("\r\n")
		if contentLength := headers["content-length"]; contentLength != "" {
			var length int
			fmt.Sscanf(contentLength, "%d", &length)
			if length > 0 && length < 10*1024*1024 {
				body := make([]byte, length)
				if _, err := io.ReadFull(reader, body); err == nil {
					requestBuilder.Write(body)
				}
			}
		}
		firstFrame := requestBuilder.String()
		if err := s.handleTunnel(conn, target, clientAddr, modeHTTPProxy, firstFrame); err != nil {
			if !isNormalCloseError(err) {
				LogError("[HTTP-%s] %s 代理失败: %v", method, clientAddr, err)
			}
		}
	default:
		LogInfo("[HTTP] %s 不支持的方法: %s", clientAddr, method)
		conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
	}
}

func (s *ProxyServer) handleTunnel(conn net.Conn, target, clientAddr string, mode int, firstFrame string) error {
	targetHost, _, err := net.SplitHostPort(target)
	if err != nil {
		targetHost = target
	}

	// 记录连接
	s.trafficStats.RecordConnection(targetHost)

	if s.shouldBypassProxy(targetHost) {
		LogInfo("[分流] %s -> %s (直连，绕过代理)", clientAddr, target)
		return s.handleDirectConnection(conn, target, clientAddr, mode, firstFrame, targetHost)
	}

	LogInfo("[分流] %s -> %s (通过代理)", clientAddr, target)
	wsConn, err := s.dialWebSocketWithECH(2)
	if err != nil {
		sendErrorResponse(conn, mode)
		return err
	}
	defer wsConn.Close()

	var mu sync.Mutex
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ping goroutine
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				wsConn.WriteMessage(websocket.PingMessage, nil)
				mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	conn.SetDeadline(time.Time{})

	// 尝试读取首帧数据
	if firstFrame == "" && mode == modeSOCKS5 {
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		buffer := make([]byte, readBufferSize)
		if n, _ := conn.Read(buffer); n > 0 {
			firstFrame = string(buffer[:n])
		}
		conn.SetReadDeadline(time.Time{})
	}

	// 发送连接请求
	connectMsg := fmt.Sprintf("CONNECT:%s|%s", target, firstFrame)
	mu.Lock()
	err = wsConn.WriteMessage(websocket.TextMessage, []byte(connectMsg))
	mu.Unlock()
	if err != nil {
		sendErrorResponse(conn, mode)
		return err
	}

	// 记录首帧上传流量
	if firstFrame != "" {
		s.trafficStats.RecordUpload(targetHost, int64(len(firstFrame)))
	}

	// 等待连接响应
	_, msg, err := wsConn.ReadMessage()
	if err != nil {
		sendErrorResponse(conn, mode)
		return err
	}

	response := string(msg)
	if strings.HasPrefix(response, "ERROR:") {
		sendErrorResponse(conn, mode)
		return errors.New(response)
	}
	if response != "CONNECTED" {
		sendErrorResponse(conn, mode)
		return fmt.Errorf("意外响应: %s", response)
	}

	if err := sendSuccessResponse(conn, mode); err != nil {
		return err
	}
	LogInfo("[代理] %s 已连接: %s", clientAddr, target)

	// 双向数据转发
	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() { closeOnce.Do(func() { close(done) }) }

	// Client -> WebSocket (上传)
	go func() {
		buf := make([]byte, readBufferSize)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				mu.Lock()
				wsConn.WriteMessage(websocket.TextMessage, []byte("CLOSE"))
				mu.Unlock()
				closeDone()
				return
			}
			s.trafficStats.RecordUpload(targetHost, int64(n))
			mu.Lock()
			err = wsConn.WriteMessage(websocket.BinaryMessage, buf[:n])
			mu.Unlock()
			if err != nil {
				closeDone()
				return
			}
		}
	}()

	// WebSocket -> Client (下载)
	go func() {
		for {
			mt, msg, err := wsConn.ReadMessage()
			if err != nil {
				closeDone()
				return
			}
			if mt == websocket.TextMessage && string(msg) == "CLOSE" {
				closeDone()
				return
			}
			s.trafficStats.RecordDownload(targetHost, int64(len(msg)))
			if _, err := conn.Write(msg); err != nil {
				closeDone()
				return
			}
		}
	}()

	<-done
	LogInfo("[代理] %s 已断开: %s", clientAddr, target)
	return nil
}

func (s *ProxyServer) handleDirectConnection(conn net.Conn, target, clientAddr string, mode int, firstFrame string, targetHost string) error {
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		host = target
		port = "443"
		if mode == modeHTTPProxy {
			port = "80"
		}
		target = net.JoinHostPort(host, port)
	}

	targetConn, err := net.DialTimeout("tcp", target, dialTimeout)
	if err != nil {
		sendErrorResponse(conn, mode)
		return fmt.Errorf("直连失败: %w", err)
	}
	defer targetConn.Close()

	if err := sendSuccessResponse(conn, mode); err != nil {
		return err
	}

	if firstFrame != "" {
		if _, err := targetConn.Write([]byte(firstFrame)); err != nil {
			return err
		}
		s.trafficStats.RecordUpload(targetHost, int64(len(firstFrame)))
	}

	// 双向数据转发
	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() { closeOnce.Do(func() { close(done) }) }

	// 上传
	go func() {
		buf := make([]byte, readBufferSize)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				closeDone()
				return
			}
			s.trafficStats.RecordUpload(targetHost, int64(n))
			if _, err := targetConn.Write(buf[:n]); err != nil {
				closeDone()
				return
			}
		}
	}()
	// 下载
	go func() {
		buf := make([]byte, readBufferSize)
		for {
			n, err := targetConn.Read(buf)
			if err != nil {
				closeDone()
				return
			}
			s.trafficStats.RecordDownload(targetHost, int64(n))
			if _, err := conn.Write(buf[:n]); err != nil {
				closeDone()
				return
			}
		}
	}()

	<-done
	LogInfo("[分流] %s 直连已断开: %s", clientAddr, target)
	return nil
}

func sendErrorResponse(conn net.Conn, mode int) {
	switch mode {
	case modeSOCKS5:
		conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	case modeHTTPConnect, modeHTTPProxy:
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
	}
}

func sendSuccessResponse(conn net.Conn, mode int) error {
	switch mode {
	case modeSOCKS5:
		_, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return err
	case modeHTTPConnect:
		_, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		return err
	case modeHTTPProxy:
		return nil
	}
	return nil
}
