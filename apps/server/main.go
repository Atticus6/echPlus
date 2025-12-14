package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var (
	token string
	port  int64
)

func init() {
	// 默认值
	defaultToken := "147258369"
	defaultPort := int64(3325)

	// 环境变量覆盖默认值
	if envToken := os.Getenv("TOKEN"); envToken != "" {
		defaultToken = envToken
	}
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := parseInt64(envPort); err == nil {
			defaultPort = p
		}
	}

	flag.StringVar(&token, "t", defaultToken, "Authentication Token (env: TOKEN)")
	flag.Int64Var(&port, "p", defaultPort, "Server Port (env: PORT)")
}

func parseInt64(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
}

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mux.HandleFunc("/health", healthHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("ECH PLUS listening on :%d", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
	log.Println("Server stopped")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handler(w http.ResponseWriter, r *http.Request) {
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))

	if upgrade != "websocket" {
		if r.URL.Path == "/" {
			w.Write([]byte("Bad Request"))
		} else {
			log.Printf("[WARN] Expected WebSocket, got Upgrade: %s", r.Header.Get("Upgrade"))
			http.Error(w, "Expected WebSocket", http.StatusUpgradeRequired)
		}
		return
	}

	protocol := r.Header.Get("Sec-WebSocket-Protocol")
	if token != "" && protocol != token {
		log.Printf("[WARN] Unauthorized: expected %s, got %s", token, protocol)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var respHeader http.Header
	if token != "" {
		respHeader = http.Header{"Sec-WebSocket-Protocol": {token}}
	}

	ws, err := upgrader.Upgrade(w, r, respHeader)
	if err != nil {
		log.Printf("[ERROR] WebSocket upgrade failed: %v", err)
		return
	}

	log.Printf("[INFO] New connection from %s", r.RemoteAddr)
	handleSession(ws, r.RemoteAddr)
}

func handleSession(ws *websocket.Conn, clientAddr string) {
	var (
		remoteConn net.Conn
		mu         sync.Mutex
		closed     bool
	)

	cleanup := func() {
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return
		}
		closed = true
		if remoteConn != nil {
			remoteConn.Close()
			remoteConn = nil
		}
		ws.Close()
		log.Printf("[INFO] Connection closed: %s", clientAddr)
	}
	defer cleanup()

	// 设置 ping/pong 保活
	ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 定期发送 ping
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			mu.Lock()
			if closed {
				mu.Unlock()
				return
			}
			if err := ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				mu.Unlock()
				return
			}
			mu.Unlock()
		}
	}()

	pumpRemoteToWS := func(conn net.Conn) {
		buf := make([]byte, 32*1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				break
			}
			mu.Lock()
			if closed {
				mu.Unlock()
				break
			}
			err = ws.WriteMessage(websocket.BinaryMessage, buf[:n])
			mu.Unlock()
			if err != nil {
				break
			}
		}
		mu.Lock()
		if !closed {
			ws.WriteMessage(websocket.TextMessage, []byte("CLOSE"))
		}
		mu.Unlock()
		cleanup()
	}

	parseAddress := func(addr string) (host string, port string) {
		if strings.HasPrefix(addr, "[") {
			end := strings.Index(addr, "]")
			if end == -1 || len(addr) < end+3 {
				return addr, ""
			}
			return addr[1:end], addr[end+2:]
		}
		sep := strings.LastIndex(addr, ":")
		if sep == -1 {
			return addr, ""
		}
		return addr[:sep], addr[sep+1:]
	}

	connectToRemote := func(targetAddr, firstFrame string) error {
		host, port := parseAddress(targetAddr)
		if host == "" || port == "" {
			return fmt.Errorf("invalid address: %s", targetAddr)
		}

		dialer := net.Dialer{Timeout: 10 * time.Second}
		conn, err := dialer.Dial("tcp", net.JoinHostPort(host, port))
		if err != nil {
			return err
		}

		if firstFrame != "" {
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := conn.Write([]byte(firstFrame)); err != nil {
				conn.Close()
				return err
			}
			conn.SetWriteDeadline(time.Time{})
		}

		mu.Lock()
		remoteConn = conn
		mu.Unlock()

		log.Printf("[INFO] Connected to remote: %s", targetAddr)
		ws.WriteMessage(websocket.TextMessage, []byte("CONNECTED"))
		go pumpRemoteToWS(conn)
		return nil
	}

	for {
		msgType, data, err := ws.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("[WARN] Read error from %s: %v", clientAddr, err)
			}
			break
		}

		// 重置读取超时
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))

		mu.Lock()
		if closed {
			mu.Unlock()
			break
		}
		mu.Unlock()

		switch msgType {
		case websocket.TextMessage:
			msg := string(data)
			switch {
			case strings.HasPrefix(msg, "CONNECT:"):
				rest := msg[8:]
				sep := strings.Index(rest, "|")
				if sep == -1 {
					ws.WriteMessage(websocket.TextMessage, []byte("ERROR:invalid CONNECT format"))
					continue
				}
				addr := rest[:sep]
				firstFrame := rest[sep+1:]
				if err := connectToRemote(addr, firstFrame); err != nil {
					log.Printf("[ERROR] Connect to %s failed: %v", addr, err)
					ws.WriteMessage(websocket.TextMessage, []byte("ERROR:"+err.Error()))
					return
				}

			case strings.HasPrefix(msg, "DATA:"):
				mu.Lock()
				if remoteConn != nil {
					remoteConn.Write([]byte(msg[5:]))
				}
				mu.Unlock()

			case msg == "CLOSE":
				return
			}
		case websocket.BinaryMessage:
			mu.Lock()
			if remoteConn != nil {
				remoteConn.Write(data)
			}
			mu.Unlock()
		}
	}
}
