package summary

import (
	"testing"
	"time"
)

func TestParseFlexibleTime(t *testing.T) {
	now := time.Date(2024, 3, 20, 15, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		input     string
		reference time.Time
		want      time.Time
		wantErr   bool
	}{
		{
			name:  "absolute date",
			input: "2024-01-01",
			want:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "rfc3339",
			input: "2024-03-20T10:00:00Z",
			want:  time.Date(2024, 3, 20, 10, 0, 0, 0, time.UTC),
		},
		{
			name:      "relative hours",
			input:     "24h",
			reference: now,
			want:      now.Add(-24 * time.Hour),
		},
		{
			name:      "relative days",
			input:     "2d",
			reference: now,
			want:      now.Add(-48 * time.Hour),
		},
		{
			name:      "relative weeks",
			input:     "1w",
			reference: now,
			want:      now.Add(-168 * time.Hour),
		},
		{
			name:    "empty input",
			input:   "",
			want:    time.Time{},
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFlexibleTime(tt.input, tt.reference)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlexibleTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("ParseFlexibleTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
