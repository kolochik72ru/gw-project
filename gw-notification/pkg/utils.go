package pkg

import (
	"fmt"
	"time"
)

// FormatDuration форматирует duration в удобочитаемый формат
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.2fm", d.Minutes())
	}
	return fmt.Sprintf("%.2fh", d.Hours())
}

// FormatRate форматирует скорость обработки
func FormatRate(messagesProcessed int64, duration time.Duration) string {
	if duration.Seconds() == 0 {
		return "0 msg/s"
	}
	rate := float64(messagesProcessed) / duration.Seconds()
	return fmt.Sprintf("%.2f msg/s", rate)
}

// FormatBytes форматирует размер в байтах
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
