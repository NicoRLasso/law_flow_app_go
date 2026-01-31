package services

import (
	"fmt"
	"law_flow_app_go/config"
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

	// 2. Send Email to Compliance Officer asynchronously
	// Note: This runs in a goroutine to avoid blocking
	go m.sendSecurityAlertEmail(ip, "", reason, "WARNING")
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

// sendSecurityAlertEmail sends a breach notification email to the admin
func (m *SecurityEventMonitor) sendSecurityAlertEmail(ip, userID, reason, alertType string) {
	cfg := config.Load()
	if cfg == nil {
		log.Printf("Failed to load config for security alert email")
		return
	}

	// Get admin email from firm or config
	adminEmail := cfg.EmailFrom // Fallback to system email
	if adminEmail == "" {
		log.Println("No admin email configured for security alerts")
		return
	}

	email := BuildBreachNotificationEmail(
		adminEmail,
		"System Administrator",
		"LexLegal Cloud",
		alertType,
		reason,
		ip,
		userID,
		"es", // Default to Spanish
	)

	if err := SendEmail(cfg, email); err != nil {
		log.Printf("Failed to send security alert email: %v", err)
	} else {
		log.Printf("Security alert email sent to %s", adminEmail)
	}
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
