package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/zeke-john/komplete/internal/config"
	"github.com/zeke-john/komplete/internal/suggest"
)

type Request struct {
	Buffer string `json:"buffer"`
	CWD    string `json:"cwd"`
	Shell  string `json:"shell"`
}

type cacheEntry struct {
	suggestion string
	timestamp  time.Time
}

type Server struct {
	listener     net.Listener
	client       *suggest.Client
	historyCache *HistoryCache
	portFile     string

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

const (
	cacheMaxEntries = 128
	cacheTTL        = 60 * time.Second
	historyRefresh  = 30 * time.Second
	requestTimeout  = 3 * time.Second
)

func NewServer(portFile string) (*Server, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GROQ_API_KEY not set")
	}

	model := loadGroqModel()

	httpClient := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	suggestClient := suggest.NewClient(apiKey, model, suggest.WithHTTPClient(httpClient))

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "zsh"
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	s := &Server{
		listener:     listener,
		client:       suggestClient,
		historyCache: NewHistoryCache(shell, historyRefresh),
		portFile:     portFile,
		cache:        make(map[string]cacheEntry),
	}

	return s, nil
}

func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Server) Run() error {
	addr := s.listener.Addr().(*net.TCPAddr)
	if err := os.WriteFile(s.portFile, []byte(fmt.Sprintf("%d", addr.Port)), 0o644); err != nil {
		s.listener.Close()
		return fmt.Errorf("write port file: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		s.Shutdown()
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return nil
		}
		go s.handleConn(conn)
	}
}

func (s *Server) Shutdown() {
	s.historyCache.Stop()
	s.listener.Close()
	os.Remove(s.portFile)
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}

	var req Request
	if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
		fmt.Fprintln(conn, "")
		return
	}

	if req.Buffer == "" {
		fmt.Fprintln(conn, "")
		return
	}

	cacheKey := req.CWD + "\x00" + req.Buffer
	if entry, ok := s.cacheGet(cacheKey); ok {
		fmt.Fprintln(conn, entry)
		return
	}

	historyStr := s.historyCache.Get()

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	suggestion, err := s.client.Complete(ctx, req.Buffer, req.CWD, req.Shell, historyStr)
	if err != nil || suggestion == "" {
		fmt.Fprintln(conn, "")
		return
	}

	s.cachePut(cacheKey, suggestion)
	fmt.Fprintln(conn, suggestion)
}

func (s *Server) cacheGet(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.cache[key]
	if !ok || time.Since(entry.timestamp) > cacheTTL {
		return "", false
	}
	return entry.suggestion, true
}

func (s *Server) cachePut(key, suggestion string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cache) >= cacheMaxEntries {
		s.evictOldest()
	}
	s.cache[key] = cacheEntry{suggestion: suggestion, timestamp: time.Now()}
}

func (s *Server) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true
	for k, v := range s.cache {
		if first || v.timestamp.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.timestamp
			first = false
		}
	}
	if oldestKey != "" {
		delete(s.cache, oldestKey)
	}
}

func loadGroqModel() string {
	path, err := config.ConfigPath()
	if err != nil {
		return ""
	}
	cfg, err := config.Load(path)
	if err != nil {
		return ""
	}
	return cfg["groq_model"]
}
