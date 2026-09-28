package flags

import "time"

type FeatureFlag struct {
	Key         string    `json:"key"`
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IsRecentlyUpdated reports whether the flag changed within a positive window.
func (f FeatureFlag) IsRecentlyUpdated(now time.Time, window time.Duration) bool {
	if window <= 0 || f.UpdatedAt.IsZero() {
		return false
	}
	age := now.Sub(f.UpdatedAt)
	return age >= 0 && age <= window
}
