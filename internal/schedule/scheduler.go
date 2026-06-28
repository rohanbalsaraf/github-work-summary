package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Schedule represents a simplified cron-style time definition.
// Supported: "HH:MM" (Daily) or "day HH:MM" (Weekly)
type Schedule struct {
	Hour   int
	Minute int
	Day    time.Weekday // -1 for daily
}

// Parse parses a schedule string like "09:00" or "Monday 09:00".
func Parse(s string) (Schedule, error) {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)

	if len(parts) == 1 {
		// HH:MM
		h, m, err := parseTime(parts[0])
		if err != nil {
			return Schedule{}, err
		}
		return Schedule{Hour: h, Minute: m, Day: -1}, nil
	}

	if len(parts) == 2 {
		// Day HH:MM
		day, err := parseDay(parts[0])
		if err != nil {
			return Schedule{}, err
		}
		h, m, err := parseTime(parts[1])
		if err != nil {
			return Schedule{}, err
		}
		return Schedule{Hour: h, Minute: m, Day: day}, nil
	}

	return Schedule{}, fmt.Errorf("invalid schedule format: %s", s)
}

func parseTime(s string) (int, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time format")
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("time out of range")
	}
	return h, m, nil
}

func parseDay(s string) (time.Weekday, error) {
	s = cases.Title(language.English).String(strings.ToLower(s))
	switch s {
	case "Sunday":
		return time.Sunday, nil
	case "Monday":
		return time.Monday, nil
	case "Tuesday":
		return time.Tuesday, nil
	case "Wednesday":
		return time.Wednesday, nil
	case "Thursday":
		return time.Thursday, nil
	case "Friday":
		return time.Friday, nil
	case "Saturday":
		return time.Saturday, nil
	}
	return -1, fmt.Errorf("invalid day: %s", s)
}

// NextRun calculates the next time this schedule should trigger.
func (s Schedule) NextRun(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), s.Hour, s.Minute, 0, 0, now.Location())

	if s.Day == -1 {
		// Daily
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
	} else {
		// Weekly
		daysUntil := (int(s.Day) - int(now.Weekday()) + 7) % 7
		if daysUntil == 0 && !next.After(now) {
			daysUntil = 7
		}
		next = next.AddDate(0, 0, daysUntil)
	}

	return next
}
