package httpapi

import (
	"context"
	"math"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	limiterIdleTTL         = 10 * time.Minute
	limiterCleanupInterval = time.Minute
	maxTrackedClients      = 100_000
)

type RateRule struct {
	PerSecond float64
	Burst     int
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipLimiter struct {
	rule    RateRule
	mu      sync.Mutex
	clients map[string]*limiterEntry
	now     func() time.Time
}

func newIPLimiter(rule RateRule) *ipLimiter {
	return &ipLimiter{rule: rule, clients: map[string]*limiterEntry{}, now: time.Now}
}

func (l *ipLimiter) reserve(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	e, ok := l.clients[ip]

	if !ok {
		if len(l.clients) >= maxTrackedClients {
			return false, time.Second
		}

		e = &limiterEntry{limiter: rate.NewLimiter(rate.Limit(l.rule.PerSecond), l.rule.Burst)}
		l.clients[ip] = e
	}

	e.lastSeen = now
	if e.limiter.AllowN(now, 1) {
		return true, 0
	}

	wait := time.Duration(float64(time.Second) / math.Max(l.rule.PerSecond, 1e-9))

	return false, wait
}

func (l *ipLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := l.now().Add(-limiterIdleTTL)
	for ip, e := range l.clients {
		if e.lastSeen.Before(cutoff) {
			delete(l.clients, ip)
		}
	}
}

func (l *ipLimiter) run(ctx context.Context) {
	ticker := time.NewTicker(limiterCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.cleanup()
		}
	}
}

func (a *API) limit(l *ipLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, wait := l.reserve(a.clientIP(r))
		if !ok {
			seconds := max(int(math.Ceil(wait.Seconds())), 1)
			writeProblem(w, r, Problem{Type: problemType("rate-limited"), Title: "Demasiados pedidos", Status: http.StatusTooManyRequests,
				Detail: "Reintentar en " + strconv.Itoa(seconds) + " s"}, seconds)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *API) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	remote, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}

	if !a.trusted(remote) {
		return remote.String()
	}

	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		return remote.String()
	}

	for _, part := range slices.Backward(strings.Split(forwarded, ",")) {
		addr, err := netip.ParseAddr(strings.TrimSpace(part))
		if err == nil && !a.trusted(addr) {
			return addr.String()
		}
	}

	return remote.String()
}

func (a *API) trusted(addr netip.Addr) bool {
	for _, p := range a.cfg.TrustedProxies {
		if p.Contains(addr.Unmap()) {
			return true
		}
	}

	return false
}
