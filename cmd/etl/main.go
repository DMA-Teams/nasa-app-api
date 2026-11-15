package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"

	"github.com/dma-teams/nasa-app-api/internal/etl"
	"github.com/dma-teams/nasa-app-api/internal/sources"
	"github.com/dma-teams/nasa-app-api/internal/store"
)

var errUsage = errors.New("usage: etl build|forecast [flags]")

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errUsage
	}

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	version := fs.String("version", "", "dataset or forecast version (default: derived from today's date)")
	soilSource := fs.String("soil", "rest", "soil source: rest (SoilGrids REST, 5 calls/min) or raster (GDAL ENVI files)")
	soilDir := fs.String("soil-dir", "data/soil", "directory with <property>_<depth>_mean.bin files when -soil=raster")
	from := fs.String("from", "1996-01-01", "first day of the climate series")
	to := fs.String("to", "2025-12-31", "last day of the climate series")
	parallel := fs.Int("parallel", 4, "concurrent units of work")
	department := fs.String("department", "", "limit the load to one department, by code (HN-LE) or name (Lempira); for local testing")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rdb := redis.NewClient(&redis.Options{
		Addr: env("REDIS_ADDR", "localhost:6379"), Username: os.Getenv("REDIS_USERNAME"), Password: os.Getenv("REDIS_PASSWORD"),
		DialTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, ContextTimeoutEnabled: true,
	})
	defer func() { _ = rdb.Close() }()

	b, err := builder(store.New(rdb), logger, *soilSource, *soilDir, *from, *to, *parallel)
	if err != nil {
		return err
	}

	b.Department = *department
	now := time.Now()

	switch args[0] {
	case "build":
		return b.Build(ctx, orDefault(*version, "d"+now.UTC().Format("2006-01-02")))
	case "forecast":
		return b.Forecast(ctx, orDefault(*version, etl.ForecastVersion(now)), now)
	default:
		return errUsage
	}
}

func builder(st *store.Store, logger *slog.Logger, soilSource, soilDir, from, to string, parallel int) (*etl.Builder, error) {
	start, err := time.Parse(time.DateOnly, from)
	if err != nil {
		return nil, fmt.Errorf("-from: %w", err)
	}

	end, err := time.Parse(time.DateOnly, to)
	if err != nil {
		return nil, fmt.Errorf("-to: %w", err)
	}

	power := sources.NewClient(90*time.Second, sources.WithRate(rate.Limit(1), 1))
	meteo := sources.NewClient(30*time.Second, sources.WithRate(rate.Limit(2), 2))
	general := sources.NewClient(60*time.Second, sources.WithMaxBody(64<<20))

	var soil etl.SoilProvider

	switch soilSource {
	case "rest":
		soil = etl.RESTSoil{Client: sources.NewSoilGridsClient(sources.NewClient(60*time.Second, sources.WithRate(rate.Every(12*time.Second), 1)))}
	case "raster":
		soil = &etl.RasterSoil{Dir: soilDir}
	default:
		return nil, fmt.Errorf("-soil must be rest or raster, got %q", soilSource)
	}

	return &etl.Builder{
		Store: st, Power: sources.NewPowerClient(power), OpenMeteo: sources.NewOpenMeteoClient(meteo),
		Boundaries: sources.NewBoundariesClient(general), Soil: soil, Logger: logger,
		From: start, To: end, Parallelism: parallel,
	}, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}

func orDefault(v, def string) string {
	if v != "" {
		return v
	}

	return def
}
