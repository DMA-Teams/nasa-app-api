package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func env(values map[string]string) func(string) string {
	return func(k string) string { return values[k] }
}

func TestLoad_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(map[string]string{"REDIS_PASSWORD": "secret"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.MunicipalTimeout != 5*time.Second || cfg.DepartmentTimeout != 20*time.Second || cfg.DiagAddr != "127.0.0.1:6060" {
		t.Errorf("defaults = %+v", cfg)
	}

	if cfg.TimeZone.String() != "America/Tegucigalpa" || len(cfg.TrustedProxies) != 3 {
		t.Errorf("time zone %v proxies %v", cfg.TimeZone, cfg.TrustedProxies)
	}
}

func TestLoad_RequiresRedisPasswordOutsideDevelopment(t *testing.T) {
	t.Parallel()

	if _, err := Load(env(nil)); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("err = %v, want ErrInvalidConfig", err)
	}

	if _, err := Load(env(map[string]string{"APP_ENV": "development"})); err != nil {
		t.Errorf("development without password: %v", err)
	}
}

func TestLoad_ReportsEveryInvalidValue(t *testing.T) {
	t.Parallel()

	_, err := Load(env(map[string]string{
		"REDIS_PASSWORD": "x", "TIMEOUT_MUNICIPAL": "soon", "RATE_MUNICIPAL_BURST": "many", "TRUSTED_PROXY_CIDRS": "nope",
	}))

	for _, key := range []string{"TIMEOUT_MUNICIPAL", "RATE_MUNICIPAL_BURST", "TRUSTED_PROXY_CIDRS"} {
		if err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("error %v does not mention %s", err, key)
		}
	}
}
