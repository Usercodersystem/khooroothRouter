package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

type rateLimiter struct {
	mu    sync.Mutex
	items map[string]*bucket
	rps   float64
	burst float64
}

func newRateLimiter(rps, burst float64) *rateLimiter {
	return &rateLimiter{items: make(map[string]*bucket), rps: rps, burst: burst}
}

func (l *rateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.items[ip]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.items[ip] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rps
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func clientIP(r *http.Request) string {
	// Do not trust X-Forwarded-For unless a trusted proxy is configured.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func errorJSON(message, kind string) map[string]any {
	return map[string]any{"error": map[string]string{"message": message, "type": kind}}
}

func main() {
	listenAddr := env("LISTEN_ADDR", ":8080")
	upstreamRaw := strings.TrimRight(env("UPSTREAM_URL", "https://api.openai.com"), "/")
	upstreamKey := os.Getenv("UPSTREAM_API_KEY")
	clientKey := os.Getenv("KHOOROOTH_API_KEY")
	maxBodyBytes := int64(envInt("MAX_BODY_BYTES", 1<<20))
	maxConcurrency := envInt("MAX_CONCURRENCY", 4)
	timeout := time.Duration(envInt("UPSTREAM_TIMEOUT_SECONDS", 120)) * time.Second
	rps := float64(envInt("RATE_LIMIT_RPS", 3))
	burst := float64(envInt("RATE_LIMIT_BURST", 6))

	upstreamURL, err := url.Parse(upstreamRaw)
	if err != nil || upstreamURL.Scheme == "" || upstreamURL.Host == "" {
		log.Fatal("UPSTREAM_URL must be an absolute URL")
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   4,
		MaxConnsPerHost:       maxConcurrency + 2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
		DisableCompression:    true,
	}
	proxy := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(upstreamURL)
			pr.Out.URL.Path = strings.TrimRight(upstreamURL.Path, "/") + pr.In.URL.Path
			pr.Out.URL.RawPath = ""
			pr.Out.URL.RawQuery = pr.In.URL.RawQuery
			pr.Out.Host = upstreamURL.Host
			pr.Out.Header.Del("Authorization")
			if upstreamKey != "" {
				pr.Out.Header.Set("Authorization", "Bearer "+upstreamKey)
			}
			pr.Out.Header.Set("User-Agent", "KhooroothRouter/0.1")
			pr.SetXForwarded()
		},
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, e error) {
			log.Printf("upstream proxy error: %v", e)
			writeJSON(w, http.StatusBadGateway, errorJSON("upstream request failed", "upstream_error"))
		},
	}

	limiter := newRateLimiter(rps, burst)
	semaphore := make(chan struct{}, maxConcurrency)
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, errorJSON("method not allowed", "invalid_request_error"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "Khoorooth Router"})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" && r.URL.Path != "/v1/chat/completions" {
			writeJSON(w, http.StatusNotFound, errorJSON("route not supported", "not_found"))
			return
		}
		if (r.URL.Path == "/v1/models" && r.Method != http.MethodGet) ||
			(r.URL.Path == "/v1/chat/completions" && r.Method != http.MethodPost) {
			writeJSON(w, http.StatusMethodNotAllowed, errorJSON("method not allowed", "invalid_request_error"))
			return
		}

		if clientKey != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got != clientKey {
				writeJSON(w, http.StatusUnauthorized, errorJSON("invalid API key", "authentication_error"))
				return
			}
		}
		if !limiter.allow(clientIP(r)) {
			writeJSON(w, http.StatusTooManyRequests, errorJSON("rate limit exceeded", "rate_limit_error"))
			return
		}

		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
		default:
			writeJSON(w, http.StatusServiceUnavailable, errorJSON("gateway concurrency limit reached", "server_error"))
			return
		}

		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		proxy.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       45 * time.Second,
		MaxHeaderBytes:    16 << 10,
		BaseContext: func(net.Listener) context.Context {
			return context.Background()
		},
	}

	log.Printf("Khoorooth Router listening on %s; upstream=%s; max_concurrency=%d", listenAddr, upstreamURL.Host, maxConcurrency)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// Keep io imported for compile-time interface documentation and future bounded
// streaming extensions. The proxy itself streams without buffering full bodies.
var _ io.Reader
