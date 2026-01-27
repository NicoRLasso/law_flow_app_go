package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// RateLimitConfig defines the configuration for rate limiting
type RateLimitConfig struct {
	// Requests is the maximum number of requests allowed within the window
	Requests int
	// Window is the time window for rate limiting
	Window time.Duration
	// KeyFunc is a function that returns a unique key for rate limiting (defaults to IP)
	KeyFunc func(c echo.Context) string
	// Message is the error message returned when rate limit is exceeded
	Message string
}

// rateLimitEntry tracks request count and window expiration
type rateLimitEntry struct {
	count     int
	expiresAt time.Time
}

// RateLimiter is a per-endpoint rate limiter
type RateLimiter struct {
	config RateLimitConfig
	store  map[string]*rateLimitEntry
	mu     sync.RWMutex
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if config.KeyFunc == nil {
		config.KeyFunc = func(c echo.Context) string {
			return c.RealIP()
		}
	}
	if config.Message == "" {
		config.Message = "Too many requests. Please try again later."
	}

	rl := &RateLimiter{
		config: config,
		store:  make(map[string]*rateLimitEntry),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := rl.config.KeyFunc(c)

			rl.mu.Lock()
			entry, exists := rl.store[key]
			now := time.Now()

			if !exists || now.After(entry.expiresAt) {
				// Create new entry or reset expired entry
				rl.store[key] = &rateLimitEntry{
					count:     1,
					expiresAt: now.Add(rl.config.Window),
				}
				rl.mu.Unlock()
				return next(c)
			}

			if entry.count >= rl.config.Requests {
				rl.mu.Unlock()
				// Return rate limit exceeded error
				if c.Request().Header.Get("HX-Request") == "true" {
					return c.HTML(http.StatusTooManyRequests, `<div class="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-xl flex items-center gap-3"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">`+rl.config.Message+`</span></div>`)
				}
				return echo.NewHTTPError(http.StatusTooManyRequests, rl.config.Message)
			}

			entry.count++
			rl.mu.Unlock()
			return next(c)
		}
	}
}

// cleanup removes expired entries every minute
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.store {
			if now.After(entry.expiresAt) {
				delete(rl.store, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Pre-configured rate limiters for common use cases

// LoginRateLimiter limits login attempts to 5 per minute per IP
var LoginRateLimiter = NewRateLimiter(RateLimitConfig{
	Requests: 5,
	Window:   1 * time.Minute,
	Message:  "Too many login attempts. Please wait a minute before trying again.",
})

// PasswordResetRateLimiter limits password reset requests to 3 per hour per IP
var PasswordResetRateLimiter = NewRateLimiter(RateLimitConfig{
	Requests: 3,
	Window:   1 * time.Hour,
	Message:  "Too many password reset requests. Please try again later.",
})

// PublicFormRateLimiter limits public form submissions to 10 per minute per IP
var PublicFormRateLimiter = NewRateLimiter(RateLimitConfig{
	Requests: 10,
	Window:   1 * time.Minute,
	Message:  "Too many form submissions. Please wait before trying again.",
})

// APIRateLimiter limits general API requests to 60 per minute per IP
var APIRateLimiter = NewRateLimiter(RateLimitConfig{
	Requests: 60,
	Window:   1 * time.Minute,
	Message:  "Rate limit exceeded. Please slow down your requests.",
})
