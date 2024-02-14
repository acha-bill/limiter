package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Limiter interface {
	Set(ip string) error
	Blacklisted(ip string) (bool, time.Time)
}

type limiter struct {
	mu          *sync.Mutex
	reqs        map[string][]time.Time
	maxReqs     int
	window      time.Duration
	blacklistMu *sync.RWMutex
	blacklist   map[string]time.Time
	backoff     time.Duration
}

func NewLimiter(maxReqs int, window time.Duration, backoff time.Duration) Limiter {
	return &limiter{
		mu:          new(sync.Mutex),
		reqs:        make(map[string][]time.Time),
		blacklistMu: new(sync.RWMutex),
		blacklist:   make(map[string]time.Time),
		maxReqs:     maxReqs,
		window:      window,
		backoff:     backoff,
	}
}

func (l *limiter) Set(ip string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if ok, t := l.Blacklisted(ip); ok {
		if time.Since(t) < l.backoff {
			return fmt.Errorf("rate limited")
		}
		l.unblock(ip)
	}

	if _, ok := l.reqs[ip]; !ok {
		l.reqs[ip] = []time.Time{time.Now()}
		return nil
	}

	l.reqs[ip] = append(l.reqs[ip], time.Now())
	var r int
	var reset []time.Time
	for _, t := range l.reqs[ip] {
		if time.Since(t) <= l.window {
			r++
			reset = append(reset, t)
		}
	}
	l.reqs[ip] = reset
	if r > l.maxReqs {
		l.block(ip)
		l.reqs[ip] = nil
		return fmt.Errorf("rate limited")
	}
	return nil
}

func (l *limiter) block(ip string) {
	l.blacklistMu.Lock()
	defer l.blacklistMu.Unlock()
	l.blacklist[ip] = time.Now()
}

func (l *limiter) unblock(ip string) {
	l.blacklistMu.Lock()
	defer l.blacklistMu.Unlock()
	delete(l.blacklist, ip)
}

func (l *limiter) Blacklisted(ip string) (bool, time.Time) {
	l.blacklistMu.RLock()
	defer l.blacklistMu.RUnlock()
	t, ok := l.blacklist[ip]
	return ok, t
}

func main() {
	rl := NewLimiter(5, time.Minute, 30*time.Second)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Forwarded-For")
		err := rl.Set(ip)
		if err != nil {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte("ok"))
	})
	http.ListenAndServe(":8080", nil)
}
