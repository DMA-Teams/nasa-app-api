package etl

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/sources"
)

func ForecastVersion(now time.Time) string {
	return "f" + now.UTC().Format("2006-01-02")
}

func (b *Builder) Forecast(ctx context.Context, version string, now time.Time) error {
	versions, err := b.Store.CurrentVersions(ctx)
	if err != nil {
		return fmt.Errorf("forecast needs a published dataset: %w", err)
	}

	data, err := b.Store.Boundaries(ctx, versions.Dataset)
	if err != nil {
		return err
	}

	ix, err := geo.DecodeBoundaries(data)
	if err != nil {
		return err
	}

	municipalities, err := b.scope(ix)
	if err != nil {
		return err
	}

	cells := uniqueCells(municipalities)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(b.parallelism())

	for _, c := range cells {
		g.Go(func() error { return b.forecastCell(gctx, versions.Dataset, version, c, now) })
	}

	if err := g.Wait(); err != nil {
		return err
	}

	if err := b.Store.SetCurrentForecast(ctx, version); err != nil {
		return err
	}

	b.Logger.InfoContext(ctx, "forecast published", slog.String("forecast", version), slog.Int("cells", len(cells)))

	return nil
}

func (b *Builder) forecastCell(ctx context.Context, dataset, version string, c geo.Cell, now time.Time) error {
	series, err := b.Store.CellSeries(ctx, dataset, c.ID())
	if err != nil {
		return fmt.Errorf("climate for forecast %s: %w", c.ID(), err)
	}

	seasonal, err := b.OpenMeteo.Seasonal(ctx, c.Lat, c.Lon)
	if err != nil {
		return fmt.Errorf("seasonal %s: %w", c.ID(), err)
	}

	stdDevs := monthlyStdDevs(series)
	f := agro.CellForecast{Issued: now.UTC().Format(time.DateOnly), Model: sources.SeasonalModel}

	for _, ym := range seasonal.FullMonths() {
		anomalies, ok := seasonal.MemberAnomalies(ym[0], ym[1])
		if !ok {
			continue
		}

		probs, ok := agro.TercilesFromEnsemble(anomalies, stdDevs[ym[1]-1])
		if !ok {
			continue
		}

		f.Months = append(f.Months, agro.MonthlyTerciles{Year: ym[0], Month: time.Month(ym[1]), Probabilities: probs})
	}

	return b.Store.PutForecast(ctx, version, c.ID(), f)
}

func monthlyStdDevs(s *agro.CellSeries) [12]float64 {
	totals := map[agro.YearMonth]float64{}

	for i := range s.Len() {
		d := s.Start.AddDate(0, 0, i)
		totals[agro.YearMonth{Year: d.Year(), Month: d.Month()}] += float64(s.Precipitation[i])
	}

	excluded := agro.ExcludedSet(agro.MonthlyPrecipitationOutliers(s))

	var out [12]float64

	for m := range 12 {
		var values []float64

		for k, v := range totals {
			if int(k.Month) == m+1 && !excluded[k] && !math.IsNaN(v) {
				values = append(values, v)
			}
		}

		out[m] = agro.StdDev(values)
	}

	return out
}
