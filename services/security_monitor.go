package services

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// SecurityEventMonitor aggregates security events and triggers alerts
type SecurityEventMonitor struct {
	mu           sync.Mutex
	failedLogins map[string][]time.Time // Map of IP -> list of failure timestamps
	alertedIPs   map[string]time.Time   // Map of IP -> last alert time
	alerts       []SecurityAlert        // History of alerts for dashboard
}

// SecurityAlert represents a triggered security alert
type SecurityAlert struct {
	Timestamp time.Time
	IP        string
	Reason    string
	Level     string // "WARNING", "CRITICAL"
}

// Global monitor instance
var Monitor *SecurityEventMonitor

// InitSecurityMonitor initializes the global monitor
func InitSecurityMonitor() {
	Monitor = &SecurityEventMonitor{
		failedLogins: make(map[string][]time.Time),
		alertedIPs:   make(map[string]time.Time),
		alerts:       make([]SecurityAlert, 0),
	}
	// Start cleanup goroutine
	go Monitor.cleanup()
}

// TrackFailedLogin records a failed login attempt and checks for threshold
func (m *SecurityEventMonitor) TrackFailedLogin(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	// Initialize slice if nil
	if _, exists := m.failedLogins[ip]; !exists {
		m.failedLogins[ip] = []time.Time{}
	}

	// Add timestamp
	m.failedLogins[ip] = append(m.failedLogins[ip], now)

	// Filter out old attempts (older than 10 minutes)
	validAttempts := []time.Time{}
	windowStart := now.Add(-10 * time.Minute)
	for _, t := range m.failedLogins[ip] {
		if t.After(windowStart) {
			validAttempts = append(validAttempts, t)
		}
	}
	m.failedLogins[ip] = validAttempts

	// Check threshold (e.g., 5 attempts in 10 minutes)
	if len(validAttempts) >= 5 {
		m.triggerAlertLocked(ip, "Multiple failed logins detected")
	}
}

// triggerAlertLocked sends an alert (log + potentially email) - called from within lock
func (m *SecurityEventMonitor) triggerAlertLocked(ip, reason string) {
	// Rate limit alerts: Max 1 per hour per IP
	lastAlert, alerted := m.alertedIPs[ip]
	if alerted && time.Since(lastAlert) < 1*time.Hour {
		return
	}

	// Update last alert time
	m.alertedIPs[ip] = time.Now()

	// Store alert in history
	alert := SecurityAlert{
		Timestamp: time.Now(),
		IP:        ip,
		Reason:    reason,
		Level:     "CRITICAL",
	}
	// Prepend to alerts (newest first), keep max 100
	m.alerts = append([]SecurityAlert{alert}, m.alerts...)
	if len(m.alerts) > 100 {
		m.alerts = m.alerts[:100]
	}

	// 1. Log critical alert
	alertMsg := fmt.Sprintf("[SECURITY ALERT] %s from IP: %s", reason, ip)
	log.Println(alertMsg)

	// 2. Send Email to Admin (if configured)
	// TODO: Implement email alert once AdminEmail is added to config
	/*
	cfg := config.Load()
	if cfg.AdminEmail != "" {
		email := &Email{
			To:      []string{cfg.AdminEmail},
			Subject: fmt.Sprintf("Security Alert: %s", reason),
			TextBody:    fmt.Sprintf("System detected a security event:\n\nType: %s\nIP Address: %s\nTime: %s\n\nPlease investigate.", reason, ip, time.Now().Format(time.RFC1123)),
		}
		SendEmailAsync(cfg, email)
	}
	*/
}

// GetRecentAlerts returns a copy of recent alerts
func (m *SecurityEventMonitor) GetRecentAlerts() []SecurityAlert {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return copy to be safe
	alertsCopy := make([]SecurityAlert, len(m.alerts))
	copy(alertsCopy, m.alerts)
	return alertsCopy
}

// cleanup periodically removes stale data
func (m *SecurityEventMonitor) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		// Cleanup failed logins map
		for ip, attempts := range m.failedLogins {
			if len(attempts) > 0 {
				lastAttempt := attempts[len(attempts)-1]
				if now.Sub(lastAttempt) > 10*time.Minute {
					delete(m.failedLogins, ip)
				}
			} else {
				delete(m.failedLogins, ip)
			}
		}
		// Cleanup alerted IPs map (if alert allows re-triggering after 1h)
		for ip, lastAlert := range m.alertedIPs {
			if now.Sub(lastAlert) > 1*time.Hour {
				delete(m.alertedIPs, ip)
			}
		}
		m.mu.Unlock()
	}
}
