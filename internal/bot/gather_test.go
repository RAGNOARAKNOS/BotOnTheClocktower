package bot

import (
	"strings"
	"testing"
	"time"
)

func TestParseGather(t *testing.T) {
	tests := []struct {
		args       string
		want       time.Duration
		wantCancel bool
		wantErr    bool
	}{
		{"", 60 * time.Second, false, false},
		{"1", time.Minute, false, false},
		{"10", 10 * time.Minute, false, false},
		{"cancel", 0, true, false},
		{"CANCEL", 0, true, false},
		{"0", 0, false, true},
		{"11", 0, false, true},
		{"-1", 0, false, true},
		{"2.5", 0, false, true},
		{"abc", 0, false, true},
		{"5 minutes", 0, false, true},
	}
	for _, tt := range tests {
		words := append([]string{"!botc", "gather"}, strings.Fields(tt.args)...)
		got, cancel, err := parseGather(words)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseGather(%q) error = %v, want error %v", tt.args, err, tt.wantErr)
			continue
		}
		if got != tt.want || cancel != tt.wantCancel {
			t.Errorf("parseGather(%q) = %v, %v; want %v, %v", tt.args, got, cancel, tt.want, tt.wantCancel)
		}
	}
}

func TestFormatCountdown(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{time.Second, "1 second"},
		{30 * time.Second, "30 seconds"},
		{time.Minute, "60 seconds"},
		{2 * time.Minute, "2 minutes"},
		{2*time.Minute + 17*time.Second, "2 minutes 17 seconds"},
		{time.Minute + time.Second + 400*time.Millisecond, "1 minute 1 second"},
	}
	for _, tt := range tests {
		if got := formatCountdown(tt.d); got != tt.want {
			t.Errorf("formatCountdown(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
