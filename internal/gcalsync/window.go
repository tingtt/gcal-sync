package gcalsync

import (
	"fmt"
	"time"
)

// Window represents the time range for a sync operation.
type Window struct {
	Start time.Time
	End   *time.Time // nil means no upper bound
}

// BuildWindow constructs a sync window from --month / --next-month flags.
// When both are unset the window starts at today with no end limit.
func BuildWindow(month string, nextMonth bool) (*Window, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if !nextMonth && month == "" {
		return &Window{Start: today}, nil
	}

	var targetMonth time.Time
	if nextMonth {
		targetMonth = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	} else {
		t, err := time.Parse("200601", month)
		if err != nil {
			return nil, fmt.Errorf("invalid --month value %q (expected YYYYMM): %w", month, err)
		}
		targetMonth = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	start := time.Date(targetMonth.Year(), targetMonth.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0)
	return &Window{Start: start, End: &end}, nil
}
