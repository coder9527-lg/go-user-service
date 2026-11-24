package common

import (
	"sync"
	"time"
)

// counter 单个 key 的计数窗口。
type counter struct {
	count    int
	expireAt time.Time
}

// IPRateLimiter 进程内固定窗口限流器，按 key（通常是客户端 IP）分别计数。
//
// 设计说明：
//   - 惰性过期：只在该 key 再次访问时检查窗口是否到期，不启动后台 goroutine，
//     因而没有 goroutine 泄漏，也无需 Stop 方法，测试友好。
//   - 单实例适用；多实例部署时各进程计数独立，生产环境应替换为 Redis 方案
//     （如 INCR + EXPIRE 或令牌桶 Lua 脚本），接口签名可保持不变。
type IPRateLimiter struct {
	mu     sync.Mutex
	store  map[string]*counter
	limit  int
	window time.Duration
	now    func() time.Time // 可在测试中注入时钟
}

// NewIPRateLimiter 创建限流器：每个窗口内同一 key 最多允许 limit 次请求。
func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		store:  make(map[string]*counter),
		limit:  limit,
		window: window,
		now:    time.Now,
	}
}

// Allow 判断 key 的本次请求是否放行，并返回当前窗口剩余秒数（用于 Retry-After）。
// 返回 allow=false 时，retryAfter 表示需要等待多久窗口才会重置。
func (l *IPRateLimiter) Allow(key string) (allow bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	w, ok := l.store[key]
	if !ok || now.After(w.expireAt) {
		// 首次访问或窗口已过期：惰性清理旧条目后开新窗口，
		// 防止不活跃 IP 的记录永久驻留导致 map 无限增长。
		if ok {
			delete(l.store, key)
		}
		l.store[key] = &counter{count: 1, expireAt: now.Add(l.window)}
		return true, 0
	}

	if w.count >= l.limit {
		return false, time.Until(w.expireAt)
	}
	w.count++
	return true, 0
}

// Len 返回当前存活的 key 数量，主要用于测试和监控。
func (l *IPRateLimiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.store)
}
