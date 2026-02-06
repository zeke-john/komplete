package daemon

import (
	"sync"
	"time"

	"github.com/zeke-john/komplete/internal/history"
)

type HistoryCache struct {
	mu       sync.RWMutex
	shell    string
	cached   string
	interval time.Duration
	stopCh   chan struct{}
}

func NewHistoryCache(shell string, interval time.Duration) *HistoryCache {
	hc := &HistoryCache{
		shell:    shell,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
	hc.refresh()
	go hc.loop()
	return hc
}

func (hc *HistoryCache) Get() string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.cached
}

func (hc *HistoryCache) Stop() {
	close(hc.stopCh)
}

func (hc *HistoryCache) refresh() {
	result := history.GetShellHistory(hc.shell)
	hc.mu.Lock()
	hc.cached = result
	hc.mu.Unlock()
}

func (hc *HistoryCache) loop() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			hc.refresh()
		case <-hc.stopCh:
			return
		}
	}
}
