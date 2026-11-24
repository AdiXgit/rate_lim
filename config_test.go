package main

import "testing"

var envKeys = []string{
	"RATE_LIMIT_CAPACITY",
	"RATE_LIMIT_REFILL_PER_SEC",
	"RATE_LIMIT_IDLE_TTL_SEC",
}

func TestLoadSettingsDefaults(t *testing.T) {
	for _, k := range envKeys {
		t.Setenv(k, "")
	}
	s, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.Capacity != 10 || s.RefillPerSec != 1.0 || s.IdleTTLSec != 300 {
		t.Fatalf("unexpected defaults: %+v", s)
	}
}

func TestLoadSettingsOverrides(t *testing.T) {
	t.Setenv("RATE_LIMIT_CAPACITY", "5")
	t.Setenv("RATE_LIMIT_REFILL_PER_SEC", "2.5")
	t.Setenv("RATE_LIMIT_IDLE_TTL_SEC", "60")
	s, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.Capacity != 5 || s.RefillPerSec != 2.5 || s.IdleTTLSec != 60 {
		t.Fatalf("unexpected settings: %+v", s)
	}
}

func TestLoadSettingsRejectsBadValues(t *testing.T) {
	for _, bad := range []string{"abc", "0", "-1", "NaN", "Inf", "+Inf", "-Inf"} {
		t.Run(bad, func(t *testing.T) {
			t.Setenv("RATE_LIMIT_CAPACITY", bad)
			if _, err := LoadSettings(); err == nil {
				t.Fatalf("expected error for %q", bad)
			}
		})
	}
}
