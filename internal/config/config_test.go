package config

import "testing"

func TestIntervalFallback(t *testing.T) {
	if got := interval("bad"); got != defaultInterval {
		t.Fatalf("expected fallback interval, got %s", got)
	}
	if got := interval("2"); got != defaultInterval {
		t.Fatalf("expected minimum guard fallback, got %s", got)
	}
}
