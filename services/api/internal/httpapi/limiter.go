package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type client struct {
	tokens   float64
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*client
	rate     float64
	capacity float64
}

func NewRateLimiter(rate float64, capacity float64) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*client),
		rate:     rate,
		capacity: capacity,
	}

	go func() {
		for {
			time.Sleep(time.Minute)
			rl.mu.Lock()
			for ip, c := range rl.clients {
				if time.Since(c.lastSeen) > 3*time.Minute {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cl, exists := rl.clients[ip]
	if !exists {
		rl.clients[ip] = &client{
			tokens:   rl.capacity - 1,
			lastSeen: now,
		}
		return true
	}

	elapsed := now.Sub(cl.lastSeen).Seconds()
	cl.lastSeen = now
	cl.tokens += elapsed * rl.rate
	if cl.tokens > rl.capacity {
		cl.tokens = rl.capacity
	}

	if cl.tokens < 1 {
		return false
	}

	cl.tokens -= 1
	return true
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		if !rl.Allow(ip) {
			writeError(w, http.StatusTooManyRequests, "слишком много запросов, подождите немного")
			return
		}

		next.ServeHTTP(w, r)
	})
}
