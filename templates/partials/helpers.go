package partials

import (
	"fmt"
	"strings"
	"time"
)

// Helper function to format file size
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Helper function to escape JavaScript strings
func escapeJS(s string) string {
	// Escape special characters for JavaScript strings
	s = strings.ReplaceAll(s, "\\", "\\\\") // Backslash first
	s = strings.ReplaceAll(s, "'", "\\'")   // Single quote
	s = strings.ReplaceAll(s, "\"", "\\\"") // Double quote
	s = strings.ReplaceAll(s, "\n", "\\n")  // Newline
	s = strings.ReplaceAll(s, "\r", "\\r")  // Carriage return
	s = strings.ReplaceAll(s, "\t", "\\t")  // Tab
	return s
}

// Helper function to format relative time
func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}
