package app

import (
	"testing"
	"time"
)

func TestGetTimeDisplayString(t *testing.T) {

	const day = time.Hour * 24
	base := time.Unix(0, 0).UTC()
	tcs := []struct {
		name     string
		now      time.Time
		ref      time.Time
		expected string
	}{
		{
			name:     "same day",
			ref:      base.Add(time.Second),
			expected: "today at 12:00:01AM",
		},
		{
			name:     "same day no mins or secs",
			ref:      base.Add(time.Hour),
			expected: "today at 1AM",
		},
		{
			name:     "same day no secs",
			ref:      base.Add(time.Hour + time.Minute),
			expected: "today at 1:01AM",
		},
		{

			name:     "same day edge case",
			now:      base.Add(23*time.Hour + 59*time.Minute + 59*time.Second),
			ref:      base.Add(23*time.Hour + 59*time.Minute + 59*time.Second).Add(2 * time.Second),
			expected: "tomorrow at 12:00:01AM",
		},
		{
			name:     "tomorrow",
			ref:      base.Add(time.Second).Add(day),
			expected: "tomorrow at 12:00:01AM",
		},
		{
			name:     "2 days",
			ref:      base.Add(time.Second).Add(day * 2),
			expected: "in 2 days at 12:00:01AM",
		},
		{
			name:     "more than a week",
			ref:      base.Add(time.Second).Add(day * 8),
			expected: "on 1970-01-09 at 12:00:01AM",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			now := base
			if !tc.now.IsZero() {
				now = tc.now
			}

			res := getTimeDisplayString(now, tc.ref)
			if res != tc.expected {
				t.Errorf("Comparison failed, expected '%s', got '%s", tc.expected, res)
			}
		})
	}
}
