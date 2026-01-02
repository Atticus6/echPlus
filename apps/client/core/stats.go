package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// SiteStats 单个站点的流量统计
type SiteStats struct {
	Host        string    `json:"host"`
	Upload      int64     `json:"upload"`       // 上传字节数
	Download    int64     `json:"download"`     // 下载字节数
	Connections int64     `json:"connections"`  // 连接次数
	LastAccess  time.Time `json:"last_access"`  // 最后访问时间
	FirstAccess time.Time `json:"first_access"` // 首次访问时间
}

// TrafficStats 流量统计管理器
type TrafficStats struct {
	mu       sync.RWMutex
	sites    map[string]*SiteStats
	storeDir string

	// 全局统计
	totalUpload   int64
	totalDownload int64
}

// NewTrafficStats 创建流量统计管理器
func NewTrafficStats(storeDir string) *TrafficStats {
	ts := &TrafficStats{
		sites:    make(map[string]*SiteStats),
		storeDir: storeDir,
	}
	ts.load()
	return ts
}

// RecordConnection 记录新连接
func (ts *TrafficStats) RecordConnection(host string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	now := time.Now()
	if stats, ok := ts.sites[host]; ok {
		stats.Connections++
		stats.LastAccess = now
	} else {
		ts.sites[host] = &SiteStats{
			Host:        host,
			Connections: 1,
			FirstAccess: now,
			LastAccess:  now,
		}
	}
}

// RecordUpload 记录上传流量
func (ts *TrafficStats) RecordUpload(host string, bytes int64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.totalUpload += bytes
	if stats, ok := ts.sites[host]; ok {
		stats.Upload += bytes
		stats.LastAccess = time.Now()
	}
}

// RecordDownload 记录下载流量
func (ts *TrafficStats) RecordDownload(host string, bytes int64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.totalDownload += bytes
	if stats, ok := ts.sites[host]; ok {
		stats.Download += bytes
		stats.LastAccess = time.Now()
	}
}

// GetSiteStats 获取单个站点统计
func (ts *TrafficStats) GetSiteStats(host string) *SiteStats {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if stats, ok := ts.sites[host]; ok {
		return &SiteStats{
			Host:        stats.Host,
			Upload:      stats.Upload,
			Download:    stats.Download,
			Connections: stats.Connections,
			FirstAccess: stats.FirstAccess,
			LastAccess:  stats.LastAccess,
		}
	}
	return nil
}

// GetAllStats 获取所有站点统计
func (ts *TrafficStats) GetAllStats() []*SiteStats {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make([]*SiteStats, 0, len(ts.sites))
	for _, stats := range ts.sites {
		result = append(result, &SiteStats{
			Host:        stats.Host,
			Upload:      stats.Upload,
			Download:    stats.Download,
			Connections: stats.Connections,
			FirstAccess: stats.FirstAccess,
			LastAccess:  stats.LastAccess,
		})
	}
	return result
}

// GetTopSites 获取流量最大的 N 个站点
func (ts *TrafficStats) GetTopSites(n int) []*SiteStats {
	all := ts.GetAllStats()
	sort.Slice(all, func(i, j int) bool {
		return (all[i].Upload + all[i].Download) > (all[j].Upload + all[j].Download)
	})
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// GetTotalStats 获取总流量统计
func (ts *TrafficStats) GetTotalStats() (upload, download int64) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.totalUpload, ts.totalDownload
}

// Reset 重置所有统计
func (ts *TrafficStats) Reset() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.sites = make(map[string]*SiteStats)
	ts.totalUpload = 0
	ts.totalDownload = 0
}

// 最小保存流量阈值 (10KB)
const minSaveThreshold = 10 * 1024

// Save 保存统计数据到文件
func (ts *TrafficStats) Save() error {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// 过滤小流量站点
	filteredSites := make(map[string]*SiteStats)
	for host, stats := range ts.sites {
		if stats.Upload+stats.Download >= minSaveThreshold {
			filteredSites[host] = stats
		}
	}

	data := struct {
		Sites         map[string]*SiteStats `json:"sites"`
		TotalUpload   int64                 `json:"total_upload"`
		TotalDownload int64                 `json:"total_download"`
		SavedAt       time.Time             `json:"saved_at"`
	}{
		Sites:         filteredSites,
		TotalUpload:   ts.totalUpload,
		TotalDownload: ts.totalDownload,
		SavedAt:       time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化统计数据失败: %w", err)
	}

	filePath := filepath.Join(ts.storeDir, "traffic_stats.json")
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("保存统计数据失败: %w", err)
	}
	return nil
}

// load 从文件加载统计数据
func (ts *TrafficStats) load() {
	filePath := filepath.Join(ts.storeDir, "traffic_stats.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return // 文件不存在或读取失败，使用空数据
	}

	var saved struct {
		Sites         map[string]*SiteStats `json:"sites"`
		TotalUpload   int64                 `json:"total_upload"`
		TotalDownload int64                 `json:"total_download"`
	}

	if err := json.Unmarshal(data, &saved); err != nil {
		return
	}

	ts.sites = saved.Sites
	ts.totalUpload = saved.TotalUpload
	ts.totalDownload = saved.TotalDownload
}

// FormatBytes 格式化字节数为可读字符串
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// PrintStats 打印统计摘要
func (ts *TrafficStats) PrintStats() string {
	upload, download := ts.GetTotalStats()
	topSites := ts.GetTopSites(10)

	var sb strings.Builder
	fmt.Fprintf(&sb, "\n========== 流量统计 ==========\n")
	fmt.Fprintf(&sb, "总上传: %s\n", FormatBytes(upload))
	fmt.Fprintf(&sb, "总下载: %s\n", FormatBytes(download))
	fmt.Fprintf(&sb, "总流量: %s\n", FormatBytes(upload+download))
	fmt.Fprintf(&sb, "站点数: %d\n", len(ts.sites))

	if len(topSites) > 0 {
		fmt.Fprintf(&sb, "\n--- Top %d 站点 ---\n", len(topSites))
		for i, site := range topSites {
			total := site.Upload + site.Download
			fmt.Fprintf(&sb, "%d. %s\n", i+1, site.Host)
			fmt.Fprintf(&sb, "   ↑ %s  ↓ %s  总计: %s  连接: %d\n",
				FormatBytes(site.Upload), FormatBytes(site.Download),
				FormatBytes(total), site.Connections)
		}
	}
	sb.WriteString("==============================\n")
	return sb.String()
}
