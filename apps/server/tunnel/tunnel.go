package tunnel

import (
	"context"
	"log"
	"sync"

	"github.com/wizzard0/trycloudflared"
)

// Tunnel 管理 Cloudflare Argo 隧道
type Tunnel struct {
	LocalPort int
	URL       string

	cancel context.CancelFunc
	mu     sync.RWMutex
}

// New 创建新的隧道实例
func New(localPort int) *Tunnel {
	return &Tunnel{
		LocalPort: localPort,
	}
}

// Start 启动 Argo 隧道
func (t *Tunnel) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	url, err := trycloudflared.CreateCloudflareTunnel(ctx, t.LocalPort)
	if err != nil {
		cancel()
		return err
	}

	t.mu.Lock()
	t.URL = url
	t.mu.Unlock()

	log.Printf("[Tunnel] Argo tunnel established: %s", url)
	return nil
}

// Stop 停止隧道
func (t *Tunnel) Stop() {
	if t.cancel != nil {
		t.cancel()
	}
	log.Println("[Tunnel] Argo tunnel stopped")
}

// GetURL 获取隧道 URL
func (t *Tunnel) GetURL() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.URL
}
