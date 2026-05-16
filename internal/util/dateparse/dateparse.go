package dateparse

import (
	"errors"
	"strings"
	"time"
)

// Parse tries to parse a string as RFC3339 first, then falls back to
// "2006-01-02". It returns an error when neither format matches.
func Parse(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, ErrEmpty
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// ParseAggregated tries multiple common layouts used by aggregated query
// results (JSON, CSV, DB dumps). It returns (zero, false) on empty or
// unrecognised input rather than an error.
func ParseAggregated(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}

// ParseAny is like ParseAggregated but accepts an any value. It handles
// nil, time.Time, *time.Time, string and []byte.
func ParseAny(raw any) (time.Time, bool) {
	switch value := raw.(type) {
	case nil:
		return time.Time{}, false
	case time.Time:
		return value, !value.IsZero()
	case *time.Time:
		if value == nil {
			return time.Time{}, false
		}
		return *value, !value.IsZero()
	case string:
		return ParseAggregated(value)
	case []byte:
		return ParseAggregated(string(value))
	default:
		return time.Time{}, false
	}
}

// ErrEmpty is returned by Parse when the input string is empty.
var ErrEmpty = errors.New("empty date string")
