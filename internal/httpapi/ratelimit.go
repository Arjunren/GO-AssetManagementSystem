package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type attempt struct {
	count int
	reset time.Time
}
type loginLimiter struct {
	mu      sync.Mutex
	limit   int
	clients map[string]attempt
}

func newLoginLimiter(limit int) *loginLimiter {
	return &loginLimiter{limit: limit, clients: map[string]attempt{}}
}
func (l *loginLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		now := time.Now()
		l.mu.Lock()
		a := l.clients[host]
		if now.After(a.reset) {
			a = attempt{reset: now.Add(time.Minute)}
		}
		a.count++
		l.clients[host] = a
		blocked := a.count > l.limit
		l.mu.Unlock()
		if blocked {
			w.Header().Set("Retry-After", "60")
			apiError(w, 429, "RATE_LIMITED", "Too many login attempts", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
