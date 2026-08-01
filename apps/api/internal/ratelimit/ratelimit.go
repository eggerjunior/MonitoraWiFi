// Package ratelimit implementa limitação de taxa por chave (ex.: IP) em memória.
// Suficiente para uma única instância de API na Fase 1 (Seção 2.2: "rate
// limiting"); uma versão distribuída (Redis) é um ADR futuro se o backend passar
// a rodar com múltiplas réplicas atrás de um load balancer sem sticky session
// para este endpoint.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*rate.Limiter
	r        rate.Limit
	burst    int
	lastSeen map[string]time.Time
}

func New(eventsPerMinute int, burst int) *Limiter {
	return &Limiter{
		buckets:  make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
		r:        rate.Every(time.Minute / time.Duration(eventsPerMinute)),
		burst:    burst,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = rate.NewLimiter(l.r, l.burst)
		l.buckets[key] = b
	}
	l.lastSeen[key] = time.Now()
	return b.Allow()
}
