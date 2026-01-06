package ratelimiter

import (
	"sync"
	"time"
)

type FixedWindowRateLimiter struct {
	sync.RWMutex
	clients map[string]int
	limit   int           // 20 req
	window  time.Duration // 5s
}

func NewFixedWindowRateLimiter(limit int, window time.Duration) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{
		clients: make(map[string]int),
		limit:   limit,
		window:  window,
	}
}

func (rl *FixedWindowRateLimiter) Allow(ip string) (bool, time.Duration) {
	rl.RLock()
	count, exists := rl.clients[ip]
	rl.RUnlock()

	if !exists || count < rl.limit { // есть там пусто и не достигли лимита
		rl.Lock()

		if !exists { // сюда сразу заходим не блокируемся потому что выполняемся в горутине
			go rl.resetCount(ip)
		}

		rl.clients[ip]++
		rl.Unlock()

		return true, 0 // не достиг лимита
	}

	return false, rl.window // запрещаем дальше выполняться
}

func (rl *FixedWindowRateLimiter) resetCount(ip string) {
	time.Sleep(rl.window)
	rl.Lock()
	delete(rl.clients, ip)
	rl.Unlock()
}

// {
//   "73.23.23.123": 4
// }
