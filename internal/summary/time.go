package summary

import (
	"fmt"
	"strings"
	"time"
)

func ParseFlexibleTime(input string, reference time.Time) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse("2006-01-02", input); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return t, nil
	}
	if d, err := ParseFlexibleDuration(input); err == nil {
		return reference.Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("invalid format")
}

func ParseFlexibleDuration(input string) (time.Duration, error) {
	input = strings.TrimSpace(input)
	if strings.HasSuffix(input, "d") {
		d, err := time.ParseDuration(strings.TrimSuffix(input, "d") + "h")
		if err != nil {
			return 0, err
		}
		return d * 24, nil
	}
	if strings.HasSuffix(input, "w") {
		d, err := time.ParseDuration(strings.TrimSuffix(input, "w") + "h")
		if err != nil {
			return 0, err
		}
		return d * 168, nil
	}
	return time.ParseDuration(input)
}
