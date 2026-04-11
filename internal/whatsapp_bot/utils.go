package whatsappbot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func decodeInto[T any](raw map[string]any) (T, error) {
	var out T
	buf, err := json.Marshal(raw)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(buf, &out)
	return out, err
}

func fmtStonky(micros int64) string {
	dollars := micros / 1_000_000
	cents := (micros % 1_000_000) / 10_000
	if cents < 0 {
		cents = -cents
	}
	sign := ""
	if micros < 0 && dollars == 0 {
		sign = "-"
	}
	return fmt.Sprintf("%s$%d.%02d", sign, dollars, cents)
}

func formatMaybeMicros(val any) string {
	n, ok := val.(float64)
	if !ok {
		return "$0.00"
	}
	return fmtStonky(int64(n))
}

func progressBar(val, max int, width int) string {
	if max == 0 {
		return strings.Repeat("░", width)
	}
	pct := float64(val) / float64(max)
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}
	filled := int(pct * float64(width))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func sparkline(data []int64) string {
	if len(data) == 0 {
		return ""
	}
	ticks := []rune(" ▂▃▄▅▆▇█")
	min := data[0]
	max := data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if min == max {
		return strings.Repeat(string(ticks[0]), len(data))
	}
	var sb strings.Builder
	for _, v := range data {
		idx := int((float64(v-min) / float64(max-min)) * float64(len(ticks)-1))
		sb.WriteRune(ticks[idx])
	}
	return sb.String()
}

func formatTimeRemaining(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
