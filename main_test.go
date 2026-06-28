package main

import "testing"

func TestZeroFor(t *testing.T) {
	if got := zeroFor[string](); got != "" {
		t.Errorf("zeroFor[string]() = %q, want empty string", got)
	}
	if got := zeroFor[int](); got != 0 {
		t.Errorf("zeroFor[int]() = %d, want 0", got)
	}
}

func TestIsZero(t *testing.T) {
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"empty string", isZero(""), true},
		{"non-empty string", isZero("host"), false},
		{"zero int", isZero(0), true},
		{"non-zero int", isZero(5), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("isZero = %v, want %v", tt.got, tt.want)
			}
		})
	}
}
