package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityMonitor(t *testing.T) {
	InitSecurityMonitor()
	ip := "127.0.0.1"

	t.Run("TrackFailedLogin", func(t *testing.T) {
		// Mock 5 failed logins
		for i := 0; i < 5; i++ {
			Monitor.TrackFailedLogin(ip)
		}

		alerts := Monitor.GetRecentAlerts()
		assert.NotEmpty(t, alerts)
		assert.Equal(t, ip, alerts[0].IP)
		assert.Contains(t, alerts[0].Reason, "Multiple failed logins")
	})

	t.Run("Duplicate Alert Rate Limit", func(t *testing.T) {
		// Already alerted for this IP in the first subtest.
		// Try to trigger again - should be rate limited.
		initialAlertCount := len(Monitor.GetRecentAlerts())

		Monitor.TrackFailedLogin(ip) // Adds another attempt (6th)
		Monitor.TrackFailedLogin(ip) // 7th
		Monitor.TrackFailedLogin(ip) // 8th
		Monitor.TrackFailedLogin(ip) // 9th
		Monitor.TrackFailedLogin(ip) // 10th -> should trigger but will be rate limited

		assert.Equal(t, initialAlertCount, len(Monitor.GetRecentAlerts()))
	})
}
