package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Simple in-memory rate limiter
type RateLimiter struct {
	requests map[string]*clientInfo
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

type clientInfo struct {
	count     int
	resetTime time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requests int, windowSeconds int) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientInfo),
		limit:    requests,
		window:   time.Duration(windowSeconds) * time.Second,
	}

	// Cleanup goroutine
	go rl.cleanup()

	return rl
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !rl.allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.requests[clientIP]

	if !exists || now.After(client.resetTime) {
		rl.requests[clientIP] = &clientInfo{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	if client.count >= rl.limit {
		return false
	}

	client.count++
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, client := range rl.requests {
			if now.After(client.resetTime) {
				delete(rl.requests, ip)
			}
		}
		rl.mu.Unlock()
	}
}
