package flags

import (
	"testing"
	"time"
)

func TestFeatureFlagIsRecentlyUpdated(t *testing.T) {
	now := time.Date(2026, time.September, 28, 10, 0, 0, 0, time.UTC)
	window := time.Hour
	tests := []struct {
		name      string
		updatedAt time.Time
		window    time.Duration
		want      bool
	}{
		{name: "recent", updatedAt: now.Add(-30 * time.Minute), window: window, want: true},
		{name: "boundary", updatedAt: now.Add(-window), window: window, want: true},
		{name: "stale", updatedAt: now.Add(-2 * time.Hour), window: window},
		{name: "future", updatedAt: now.Add(time.Minute), window: window},
		{name: "missing timestamp", window: window},
		{name: "invalid window", updatedAt: now, window: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := FeatureFlag{UpdatedAt: tt.updatedAt}
			if got := flag.IsRecentlyUpdated(now, tt.window); got != tt.want {
				t.Fatalf("IsRecentlyUpdated() = %t, want %t", got, tt.want)
			}
		})
	}
}
