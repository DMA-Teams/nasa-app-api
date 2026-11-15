package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/time/rate"

	"github.com/dma-teams/nasa-app-api/internal/sources"
)

var errUsage = errors.New("usage: calibrate imerg|canicula [flags]")

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
	from := fs.String("from", "2000-06-01", "first day")
	to := fs.String("to", "2025-09-30", "last day")
	out := fs.String("out", "data/calibration", "output directory")
	parallel := fs.Int("parallel", 4, "concurrent downloads")
	testdata := fs.String("testdata", "testdata/lempira", "recorded profiles and climate")
	boundaries := fs.String("boundaries", "data/calibration/appeears/hnd_adm2.geojson", "geoBoundaries ADM2 GeoJSON")
	firstYear := fs.Int("first-year", 2001, "first year of the canícula report")
	lastYear := fs.Int("last-year", 2025, "last year of the canícula report")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	start, err := time.Parse(time.DateOnly, *from)
	if err != nil {
		return fmt.Errorf("parse -from: %w", err)
	}

	end, err := time.Parse(time.DateOnly, *to)
	if err != nil {
		return fmt.Errorf("parse -to: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	switch args[0] {
	case "imerg":
		token := os.Getenv("EARTHDATA_TOKEN")
		if token == "" {
			return errors.New("EARTHDATA_TOKEN is not set")
		}

		d := &imergDownloader{
			cmr:      sources.NewClient(time.Minute, sources.WithRate(rate.Limit(2), 1)),
			data:     sources.NewClient(2*time.Minute, sources.WithBearer(token), sources.WithRate(rate.Limit(*parallel), *parallel)),
			cmrURL:   cmrGranulesURL,
			dir:      *out,
			parallel: *parallel,
			logger:   logger,
		}

		return d.Run(ctx, start, end)
	case "canicula":
		grid, err := loadImergGrid(filepath.Join(*out, "imerg"), start, end)
		if err != nil {
			return err
		}

		series, err := loadMunicipalitySeries(*testdata, *boundaries, grid)
		if err != nil {
			return err
		}

		return caniculaReport(os.Stdout, series, *firstYear, *lastYear)
	default:
		return errUsage
	}
}
