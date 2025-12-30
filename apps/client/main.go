package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/atticus6/echPlus/apps/client/core"
)

var (
	listenAddr  string
	serverAddr  string
	serverIP    string
	token       string
	dnsServer   string
	echDomain   string
	routingMode string
)

func init() {
	flag.StringVar(&listenAddr, "l", getEnv("ECHPLUS_LISTEN", "127.0.0.1:30000"), "代理监听地址 (支持 SOCKS5 和 HTTP) [环境变量: ECHPLUS_LISTEN]")
	flag.StringVar(&serverAddr, "f", getEnv("ECHPLUS_SERVER", ""), "服务端地址 (格式: x.x.workers.dev:443) [环境变量: ECHPLUS_SERVER]")
	flag.StringVar(&serverIP, "ip", getEnv("ECHPLUS_SERVER_IP", ""), "指定服务端 IP（绕过 DNS 解析）[环境变量: ECHPLUS_SERVER_IP]")
	flag.StringVar(&token, "token", getEnv("ECHPLUS_TOKEN", "147258369"), "身份验证令牌 [环境变量: ECHPLUS_TOKEN]")
	flag.StringVar(&dnsServer, "dns", getEnv("ECHPLUS_DNS", "dns.alidns.com/dns-query"), "ECH 查询 DoH 服务器 [环境变量: ECHPLUS_DNS]")
	flag.StringVar(&echDomain, "ech", getEnv("ECHPLUS_ECH_DOMAIN", "cloudflare-ech.com"), "ECH 查询域名 [环境变量: ECHPLUS_ECH_DOMAIN]")
	flag.StringVar(&routingMode, "routing", getEnv("ECHPLUS_ROUTING", "global"), "分流模式: global(全局代理), bypass_cn(跳过中国大陆), none(不改变代理) [环境变量: ECHPLUS_ROUTING]")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	flag.Parse()
	if serverAddr == "" {
		log.Fatal("必须指定服务端地址 -f\n\n示例:\n  ./client -l 127.0.0.1:1080 -f your-worker.workers.dev:443 -token your-token")
	}
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("获取可执行文件路径失败: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	StoreDir := filepath.Join(exeDir, ".echplus")
	
	// 确保存储目录存在
	if err := os.MkdirAll(StoreDir, 0755); err != nil {
		log.Fatalf("创建存储目录失败: %v", err)
	}
	cfg := core.Config{
		ListenAddr:  listenAddr,
		ServerAddr:  serverAddr,
		ServerIP:    serverIP,
		Token:       token,
		DNSServer:   dnsServer,
		ECHDomain:   echDomain,
		RoutingMode: core.RoutingMode(routingMode),
		StoreDir:    StoreDir,
	}

	server := core.NewProxyServer(cfg)

	if err := server.Start(); err != nil {
		log.Fatalf("[启动] 服务器启动失败: %v", err)
	}

	// 启动命令行交互
	go handleCommands(server)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("[退出] 正在关闭服务器...")
	server.Stop()
}

func handleCommands(server *core.ProxyServer) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n[命令] 可用命令: restart, status, routing <mode>, quit")

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			continue
		}

		input = strings.TrimSpace(input)
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "restart":
			fmt.Println("[命令] 正在重启服务器...")
			if err := server.Restart(); err != nil {
				fmt.Printf("[命令] 重启失败: %v\n", err)
			} else {
				fmt.Println("[命令] 服务器已重启")
			}

		case "status":
			cfg := server.GetConfig()
			running := "运行中"
			if !server.IsRunning() {
				running = "已停止"
			}
			fmt.Printf("[状态] %s\n", running)
			fmt.Printf("  监听地址: %s\n", cfg.ListenAddr)
			fmt.Printf("  服务端: %s\n", cfg.ServerAddr)
			fmt.Printf("  分流模式: %s\n", cfg.RoutingMode)

		case "routing":
			if len(parts) < 2 {
				fmt.Println("[命令] 用法: routing <global|bypass_cn|none>")
				continue
			}
			mode := core.RoutingMode(strings.ToLower(parts[1]))
			if mode != core.RoutingModeGlobal && mode != core.RoutingModeBypassCN && mode != core.RoutingModeNone {
				fmt.Println("[命令] 无效的分流模式，可选: global, bypass_cn, none")
				continue
			}
			cfg := server.GetConfig()
			cfg.RoutingMode = mode
			fmt.Printf("[命令] 正在切换分流模式为 %s 并重启...\n", mode)
			if err := server.UpdateConfig(cfg); err != nil {
				fmt.Printf("[命令] 切换失败: %v\n", err)
			} else {
				fmt.Printf("[命令] 分流模式已切换为 %s\n", mode)
			}

		case "quit", "exit", "q":
			fmt.Println("[命令] 正在退出...")
			server.Stop()
			os.Exit(0)

		case "help":
			fmt.Println("[命令] 可用命令:")
			fmt.Println("  restart        - 重启代理服务器")
			fmt.Println("  status         - 查看服务器状态")
			fmt.Println("  routing <mode> - 切换分流模式 (global/bypass_cn/none)")
			fmt.Println("  quit/exit/q    - 退出程序")

		default:
			fmt.Printf("[命令] 未知命令: %s，输入 help 查看帮助\n", cmd)
		}
	}
}
