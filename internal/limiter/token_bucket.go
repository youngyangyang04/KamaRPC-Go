package limiter

import (
	"sync"
	"time"
)

type TokenBucket struct {
	tokens int
	rate   int
	mu     sync.Mutex
}

func NewTokenBucket(rate int) *TokenBucket {
	tb := &TokenBucket{tokens: rate, rate: rate}
	go func() {
		for {
			time.Sleep(time.Second)
			tb.mu.Lock()
			tb.tokens = tb.rate
			tb.mu.Unlock()
		}
	}()
	return tb
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}
