package models

import (
	"testing"
	"time"
)

func TestResolveDirection(t *testing.T) {
	inStr := "in"
	outStr := "out"
	entryStr := "Entry"
	exitStr := "Exit"
	emptyStr := ""
	invalidStr := "unknown"

	tests := []struct {
		name      string
		devDir    *string
		punchType uint8
		want      uint8
	}{
		{
			name:      "device in",
			devDir:    &inStr,
			punchType: 255,
			want:      DirectionIn,
		},
		{
			name:      "device out",
			devDir:    &outStr,
			punchType: 255,
			want:      DirectionOut,
		},
		{
			name:      "device Entry case-insensitive",
			devDir:    &entryStr,
			punchType: 0,
			want:      DirectionIn,
		},
		{
			name:      "device Exit case-insensitive",
			devDir:    &exitStr,
			punchType: 1,
			want:      DirectionOut,
		},
		{
			name:      "nil device with punchType 0 (in)",
			devDir:    nil,
			punchType: 0,
			want:      DirectionIn,
		},
		{
			name:      "nil device with punchType 1 (out)",
			devDir:    nil,
			punchType: 1,
			want:      DirectionOut,
		},
		{
			name:      "nil device with punchType 255 (unknown)",
			devDir:    nil,
			punchType: 255,
			want:      DirectionUnknown,
		},
		{
			name:      "empty device with punchType 0",
			devDir:    &emptyStr,
			punchType: 0,
			want:      DirectionIn,
		},
		{
			name:      "invalid device with punchType 255",
			devDir:    &invalidStr,
			punchType: 255,
			want:      DirectionUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveDirection(tt.devDir, tt.punchType)
			if got != tt.want {
				t.Errorf("ResolveDirection(%v, %d) = %d; want %d", tt.devDir, tt.punchType, got, tt.want)
			}
		})
	}
}

func TestMakeDedupeHash(t *testing.T) {
	devID := uint(1)
	pin := "1234"
	tm := time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC)

	hash1 := MakeDedupeHash(&devID, pin, tm)
	hash2 := MakeDedupeHash(&devID, pin, tm)
	if hash1 == "" || hash1 != hash2 {
		t.Errorf("expected deterministic non-empty hash, got %q and %q", hash1, hash2)
	}

	devID2 := uint(2)
	hashOtherDev := MakeDedupeHash(&devID2, pin, tm)
	if hash1 == hashOtherDev {
		t.Errorf("expected different hash for different device IDs")
	}
}
