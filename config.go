package main

import (
	"fmt"
	"os"
	"strconv"
)

// Settings holds the rate limiter configuration.
type Settings struct {
	Capacity     float64
	RefillPerSec float64
	IdleTTLSec   float64
}

// envFloat reads key as a positive float64, returning def if it is unset or empty.
func envFloat(key string, def float64) (float64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("%s must be a positive number", key)
	}
	return v, nil
}

// LoadSettings loads Settings from the environment, returning the first error hit.
func LoadSettings() (Settings, error) {
	capacity, err := envFloat("RATE_LIMIT_CAPACITY", 10)
	if err != nil {
		return Settings{}, err
	}
	refill, err := envFloat("RATE_LIMIT_REFILL_PER_SEC", 1.0)
	if err != nil {
		return Settings{}, err
	}
	ttl, err := envFloat("RATE_LIMIT_IDLE_TTL_SEC", 300)
	if err != nil {
		return Settings{}, err
	}
	return Settings{Capacity: capacity, RefillPerSec: refill, IdleTTLSec: ttl}, nil
}
