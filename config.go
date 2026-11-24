package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

type Settings struct {
	Capacity     float64
	RefillPerSec float64
	IdleTTLSec   float64
}

func envFloat(key string, def float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || f <= 0 {
		return 0, fmt.Errorf("%s must be a positive number", key)
	}
	return f, nil
}

func LoadSettings() (Settings, error) {
	var s Settings
	var err error
	if s.Capacity, err = envFloat("RATE_LIMIT_CAPACITY", 10); err != nil {
		return s, err
	}
	if s.RefillPerSec, err = envFloat("RATE_LIMIT_REFILL_PER_SEC", 1.0); err != nil {
		return s, err
	}
	if s.IdleTTLSec, err = envFloat("RATE_LIMIT_IDLE_TTL_SEC", 300); err != nil {
		return s, err
	}
	return s, nil
}
