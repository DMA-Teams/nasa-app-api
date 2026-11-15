package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidConfig = errors.New("invalid configuration")

type Rate struct {
	PerSecond float64
	Burst     int
}

type Config struct {
	Environment         string
	HTTPAddr            string
	DiagAddr            string
	RedisAddr           string
	RedisUsername       string
	RedisPassword       string
	CORSOrigins         []string
	TrustedProxies      []netip.Prefix
	MunicipalTimeout    time.Duration
	DepartmentTimeout   time.Duration
	ComputeTimeout      time.Duration
	ShutdownTimeout     time.Duration
	VersionPollInterval time.Duration
	MunicipalRate       Rate
	DepartmentRate      Rate
	HeavyMaxInFlight    int
	PlanCacheMaxAge     time.Duration
	TimeZone            *time.Location
	LogLevel            slog.Level
}

type loader struct {
	getenv func(string) string
	errs   []error
}

func Load(getenv func(string) string) (Config, error) {
	l := &loader{getenv: getenv}

	cfg := Config{
		Environment:         l.str("APP_ENV", "production"),
		HTTPAddr:            l.str("HTTP_ADDR", ":8080"),
		DiagAddr:            l.str("DIAG_ADDR", "127.0.0.1:6060"),
		RedisAddr:           l.str("REDIS_ADDR", "redis:6379"),
		RedisUsername:       l.str("REDIS_USERNAME", ""),
		RedisPassword:       l.str("REDIS_PASSWORD", ""),
		CORSOrigins:         l.list("CORS_ORIGINS"),
		TrustedProxies:      l.prefixes("TRUSTED_PROXY_CIDRS", "172.16.0.0/12,10.0.0.0/8,127.0.0.1/32"),
		MunicipalTimeout:    l.duration("TIMEOUT_MUNICIPAL", 5*time.Second),
		DepartmentTimeout:   l.duration("TIMEOUT_DEPARTMENT", 20*time.Second),
		ComputeTimeout:      l.duration("TIMEOUT_COMPUTE", 10*time.Second),
		ShutdownTimeout:     l.duration("TIMEOUT_SHUTDOWN", 15*time.Second),
		VersionPollInterval: l.duration("VERSION_POLL_INTERVAL", time.Minute),
		MunicipalRate:       Rate{PerSecond: l.float("RATE_MUNICIPAL_PER_SECOND", 2), Burst: l.int("RATE_MUNICIPAL_BURST", 10)},
		DepartmentRate:      Rate{PerSecond: l.float("RATE_DEPARTMENT_PER_SECOND", 0.1), Burst: l.int("RATE_DEPARTMENT_BURST", 2)},
		HeavyMaxInFlight:    l.int("HEAVY_MAX_INFLIGHT", runtime.GOMAXPROCS(0)),
		PlanCacheMaxAge:     l.duration("PLAN_CACHE_MAX_AGE", time.Hour),
		TimeZone:            l.location("TIME_ZONE", "America/Tegucigalpa"),
		LogLevel:            l.level("LOG_LEVEL", slog.LevelInfo),
	}

	l.validate(cfg)

	if err := errors.Join(l.errs...); err != nil {
		return Config{}, errors.Join(ErrInvalidConfig, err)
	}

	return cfg, nil
}

func (l *loader) validate(cfg Config) {
	if cfg.RedisPassword == "" && cfg.Environment != "development" {
		l.errs = append(l.errs, errors.New("REDIS_PASSWORD is required outside APP_ENV=development"))
	}

	if cfg.HeavyMaxInFlight < 1 || cfg.MunicipalRate.Burst < 1 || cfg.DepartmentRate.Burst < 1 {
		l.errs = append(l.errs, errors.New("in-flight limits and rate bursts must be >= 1"))
	}

	if cfg.MunicipalRate.PerSecond <= 0 || cfg.DepartmentRate.PerSecond <= 0 {
		l.errs = append(l.errs, errors.New("rates must be > 0"))
	}

	if cfg.ComputeTimeout > cfg.DepartmentTimeout {
		l.errs = append(l.errs, errors.New("TIMEOUT_COMPUTE must not exceed TIMEOUT_DEPARTMENT"))
	}
}

func (l *loader) str(key, def string) string {
	if v := strings.TrimSpace(l.getenv(key)); v != "" {
		return v
	}

	return def
}

func (l *loader) list(key string) []string {
	var out []string

	for v := range strings.SplitSeq(l.getenv(key), ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}

	return out
}

func (l *loader) duration(key string, def time.Duration) time.Duration {
	v := l.getenv(key)
	if v == "" {
		return def
	}

	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		l.errs = append(l.errs, fmt.Errorf("%s: positive duration required, got %q", key, v))

		return def
	}

	return d
}

func (l *loader) float(key string, def float64) float64 {
	v := l.getenv(key)
	if v == "" {
		return def
	}

	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s: number required, got %q", key, v))

		return def
	}

	return f
}

func (l *loader) int(key string, def int) int {
	v := l.getenv(key)
	if v == "" {
		return def
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s: integer required, got %q", key, v))

		return def
	}

	return n
}

func (l *loader) prefixes(key, def string) []netip.Prefix {
	raw := l.str(key, def)

	var out []netip.Prefix

	for v := range strings.SplitSeq(raw, ",") {
		p, err := netip.ParsePrefix(strings.TrimSpace(v))
		if err != nil {
			l.errs = append(l.errs, fmt.Errorf("%s: invalid CIDR %q", key, v))

			continue
		}

		out = append(out, p)
	}

	return out
}

func (l *loader) location(key, def string) *time.Location {
	name := l.str(key, def)

	loc, err := time.LoadLocation(name)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s: unknown time zone %q", key, name))

		return time.UTC
	}

	return loc
}

func (l *loader) level(key string, def slog.Level) slog.Level {
	v := l.getenv(key)
	if v == "" {
		return def
	}

	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(v)); err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s: invalid log level %q", key, v))

		return def
	}

	return lvl
}
