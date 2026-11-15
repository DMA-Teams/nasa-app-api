package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/catalog"
	"github.com/dma-teams/nasa-app-api/internal/config"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/httpapi"
	"github.com/dma-teams/nasa-app-api/internal/plan"
	"github.com/dma-teams/nasa-app-api/internal/store"
)

const (
	redisStartupAttempts = 10
	redisStartupDelay    = 2 * time.Second
	versionReadTimeout   = 2 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	cat, err := catalog.Load()
	if err != nil {
		return fmt.Errorf("load catalog: %w", err)
	}

	engine, err := agro.NewEngine(cat.Crops, cat.Templates)
	if err != nil {
		return fmt.Errorf("build engine: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr, Username: cfg.RedisUsername, Password: cfg.RedisPassword,
		DialTimeout: 2 * time.Second, ReadTimeout: 500 * time.Millisecond, WriteTimeout: 500 * time.Millisecond,
		ContextTimeoutEnabled: true, PoolTimeout: time.Second, MaxRetries: 2,
	})
	defer func() { _ = rdb.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st := store.New(rdb)
	if err := waitForRedis(ctx, st, logger); err != nil {
		return err
	}

	var directory atomic.Pointer[geo.Index]

	watcher := store.NewWatcher(st, cfg.VersionPollInterval, versionReadTimeout, func(ctx context.Context, v store.Versions) error {
		data, err := st.Boundaries(ctx, v.Dataset)
		if err != nil {
			return fmt.Errorf("boundaries for %s: %w", v.Dataset, err)
		}

		ix, err := geo.DecodeBoundaries(data)
		if err != nil {
			return err
		}

		directory.Store(ix)
		logger.InfoContext(ctx, "dataset loaded", slog.String("dataset", v.Dataset), slog.String("forecast", v.Forecast))

		return nil
	}, logger)

	if err := watcher.Refresh(ctx); err != nil {
		logger.WarnContext(ctx, "dataset not available yet; the API starts as not ready", slog.String("error", err.Error()))
	}

	gazetteer := gazetteerFunc(func() (*geo.Index, bool) {
		ix := directory.Load()

		return ix, ix != nil
	})

	svc := plan.NewService(
		plan.Config{CatalogHash: cat.Hash, Location: cfg.TimeZone, ComputeTimeout: cfg.ComputeTimeout, MaxParallel: cfg.HeavyMaxInFlight},
		plan.Dependencies{
			Profiles: st, Climate: st, Forecasts: st, Cache: st, Engine: engine, Logger: logger,
			Versions: versionsFunc(watcher.Current), Gazetteer: gazetteer,
		},
	)

	api := httpapi.New(apiConfig(cfg), svc, func() (httpapi.Directory, bool) {
		ix := directory.Load()

		return ix, ix != nil
	}, cat, func(ctx context.Context) error {
		if _, ok := watcher.Current(); !ok {
			return plan.ErrNotReady
		}

		return st.Ping(ctx)
	}, logger)

	return serve(ctx, cfg, api, watcher, logger)
}

func serve(ctx context.Context, cfg config.Config, api *httpapi.API, watcher *store.Watcher, logger *slog.Logger) error {
	srv := &http.Server{
		Addr: cfg.HTTPAddr, Handler: api.Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: cfg.DepartmentTimeout + 10*time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 64 << 10, ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}
	diag := &http.Server{Addr: cfg.DiagAddr, Handler: diagnostics(), ReadHeaderTimeout: 5 * time.Second}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error { return listen(srv, logger, "api") })
	g.Go(func() error { return listen(diag, logger, "diagnostics") })
	g.Go(func() error { return watcher.Run(gctx) })
	g.Go(func() error { return api.RunBackground(gctx) })
	g.Go(func() error {
		<-gctx.Done()
		api.StartShutdown()
		logger.Info("shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(gctx), cfg.ShutdownTimeout)
		defer cancel()

		return errors.Join(srv.Shutdown(shutdownCtx), diag.Shutdown(shutdownCtx))
	})

	return g.Wait()
}

func listen(srv *http.Server, logger *slog.Logger, name string) error {
	logger.Info("listening", slog.String("server", name), slog.String("addr", srv.Addr))

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("%s server: %w", name, err)
	}

	return nil
}

func diagnostics() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/vars", expvar.Handler())

	return mux
}

func waitForRedis(ctx context.Context, st *store.Store, logger *slog.Logger) error {
	var err error

	for attempt := 1; attempt <= redisStartupAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, redisStartupDelay)
		err = st.Ping(pingCtx)

		cancel()

		if err == nil {
			return nil
		}

		logger.WarnContext(ctx, "redis not reachable yet", slog.Int("attempt", attempt), slog.String("error", err.Error()))

		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for redis: %w", ctx.Err())
		case <-time.After(redisStartupDelay):
		}
	}

	return fmt.Errorf("redis unreachable after %d attempts: %w", redisStartupAttempts, err)
}

func apiConfig(cfg config.Config) httpapi.Config {
	return httpapi.Config{
		CORSOrigins: cfg.CORSOrigins, TrustedProxies: cfg.TrustedProxies,
		MunicipalTimeout: cfg.MunicipalTimeout, DepartmentTimeout: cfg.DepartmentTimeout,
		MunicipalRate:         httpapi.RateRule{PerSecond: cfg.MunicipalRate.PerSecond, Burst: cfg.MunicipalRate.Burst},
		DepartmentRate:        httpapi.RateRule{PerSecond: cfg.DepartmentRate.PerSecond, Burst: cfg.DepartmentRate.Burst},
		HeavyMaxInFlight:      cfg.HeavyMaxInFlight,
		ReadinessTimeout:      time.Second,
		PlanCacheMaxAgeSecond: int(cfg.PlanCacheMaxAge.Seconds()),
	}
}

type versionsFunc func() (store.Versions, bool)

func (f versionsFunc) Versions() (plan.Versions, bool) {
	v, ok := f()

	return plan.Versions{Dataset: v.Dataset, Forecast: v.Forecast}, ok
}

type gazetteerFunc func() (*geo.Index, bool)

func (g gazetteerFunc) index() (*geo.Index, error) {
	ix, ok := g()
	if !ok {
		return nil, plan.ErrNotReady
	}

	return ix, nil
}

func (g gazetteerFunc) Municipality(code string) (geo.Municipality, error) {
	ix, err := g.index()
	if err != nil {
		return geo.Municipality{}, err
	}

	return ix.Municipality(code)
}

func (g gazetteerFunc) Department(code string) (geo.Department, error) {
	ix, err := g.index()
	if err != nil {
		return geo.Department{}, err
	}

	return ix.Department(code)
}

func (g gazetteerFunc) MunicipalityCodesOf(departmentCode string) ([]string, error) {
	ix, err := g.index()
	if err != nil {
		return nil, err
	}

	return ix.MunicipalityCodesOf(departmentCode)
}
