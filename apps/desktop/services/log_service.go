package services

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/atticus6/echPlus/apps/desktop/config"
)

type LogService struct{}

type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type LogFile struct {
	Name string `json:"name"`
	Date string `json:"date"`
	Type string `json:"type"`
}

// GetLogDir 返回日志目录路径
func (s *LogService) GetLogDir() string {
	return filepath.Join(config.StoreDir, "logs")
}

// ListLogFiles 列出所有日志文件
func (s *LogService) ListLogFiles() ([]LogFile, error) {
	logDir := s.GetLogDir()
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, err
	}

	var files []LogFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		name := entry.Name()
		// 解析文件名: info_2024-01-01.log
		parts := strings.Split(strings.TrimSuffix(name, ".log"), "_")
		if len(parts) >= 2 {
			files = append(files, LogFile{
				Name: name,
				Type: parts[0],
				Date: parts[1],
			})
		}
	}

	// 按日期倒序排列
	sort.Slice(files, func(i, j int) bool {
		return files[i].Date > files[j].Date
	})

	return files, nil
}

// ReadLogFile 读取指定日志文件的内容
func (s *LogService) ReadLogFile(filename string, lines int) ([]LogEntry, error) {
	logDir := s.GetLogDir()
	filePath := filepath.Join(logDir, filename)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		entry := parseLogLine(line)
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 返回最后 N 行，并倒序（最新的在前面）
	if lines > 0 && len(entries) > lines {
		entries = entries[len(entries)-lines:]
	}

	// 倒序排列，最新的在前面
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

// GetTodayLogs 获取今天的所有日志
func (s *LogService) GetTodayLogs(logType string, lines int) ([]LogEntry, error) {
	today := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s_%s.log", logType, today)
	return s.ReadLogFile(filename, lines)
}

// parseLogLine 解析日志行
func parseLogLine(line string) LogEntry {
	entry := LogEntry{Message: line}

	// 格式: [LEVEL] HH:MM:SS file:line: message
	if strings.HasPrefix(line, "[") {
		endBracket := strings.Index(line, "]")
		if endBracket > 0 {
			entry.Level = line[1:endBracket]
			rest := strings.TrimSpace(line[endBracket+1:])

			// 提取时间
			if len(rest) >= 8 {
				entry.Time = rest[:8]
				entry.Message = strings.TrimSpace(rest[8:])
			}
		}
	}

	return entry
}
