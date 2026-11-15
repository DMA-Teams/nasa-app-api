package etl

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/sources"
	"github.com/dma-teams/nasa-app-api/internal/store"
)

const (
	elevationSpacingDeg    = 0.045
	minimumElevationPts    = 9
	defaultParallelism     = 4
	minimumProfileCoverage = 0.95
)

var ErrIncompleteBuild = errors.New("incomplete dataset build")

type Store interface {
	Exists(ctx context.Context, key string) (bool, error)
	PutBoundaries(ctx context.Context, dataset string, data []byte) error
	Boundaries(ctx context.Context, dataset string) ([]byte, error)
	PutCellSeries(ctx context.Context, dataset, cell string, s *agro.CellSeries) error
	CellSeries(ctx context.Context, dataset, cell string) (*agro.CellSeries, error)
	PutCellMeta(ctx context.Context, dataset string, meta store.CellMeta) error
	CellMeta(ctx context.Context, dataset, cell string) (store.CellMeta, error)
	PutProfile(ctx context.Context, dataset string, p *agro.SiteProfile) error
	PutForecast(ctx context.Context, version, cell string, f agro.CellForecast) error
	SetCurrentDataset(ctx context.Context, version string) error
	SetCurrentForecast(ctx context.Context, version string) error
	CurrentVersions(ctx context.Context) (store.Versions, error)
}

type Builder struct {
	Store       Store
	Power       *sources.PowerClient
	OpenMeteo   *sources.OpenMeteoClient
	Boundaries  *sources.BoundariesClient
	Soil        SoilProvider
	Logger      *slog.Logger
	From        time.Time
	To          time.Time
	Parallelism int
	Department  string
}

func (b *Builder) scope(ix *geo.Index) ([]geo.Municipality, error) {
	all := ix.Municipalities()
	if b.Department == "" {
		return all, nil
	}

	var out []geo.Municipality

	for _, m := range all {
		dep, _ := ix.Department(m.DepartmentCode)
		if strings.EqualFold(m.DepartmentCode, b.Department) || strings.EqualFold(dep.Name, b.Department) {
			out = append(out, m)
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("department %q has no municipalities: %w", b.Department, ErrIncompleteBuild)
	}

	return out, nil
}

func (b *Builder) parallelism() int {
	if b.Parallelism > 0 {
		return b.Parallelism
	}

	return defaultParallelism
}

func (b *Builder) Build(ctx context.Context, dataset string) error {
	ix, err := b.boundaries(ctx, dataset)
	if err != nil {
		return err
	}

	municipalities, err := b.scope(ix)
	if err != nil {
		return err
	}

	cells := uniqueCells(municipalities)
	b.Logger.InfoContext(ctx, "dataset scope", slog.String("department", b.Department),
		slog.Int("municipalities", len(municipalities)), slog.Int("cells", len(cells)))

	if err := b.loadClimate(ctx, dataset, cells); err != nil {
		return err
	}

	built, err := b.loadProfiles(ctx, dataset, ix, municipalities)
	if err != nil {
		return err
	}

	if float64(built) < minimumProfileCoverage*float64(len(municipalities)) {
		return fmt.Errorf("%d of %d profiles built: %w", built, len(municipalities), ErrIncompleteBuild)
	}

	if err := b.Store.SetCurrentDataset(ctx, dataset); err != nil {
		return err
	}

	b.Logger.InfoContext(ctx, "dataset published", slog.String("dataset", dataset), slog.Int("profiles", built))

	return nil
}

func (b *Builder) boundaries(ctx context.Context, dataset string) (*geo.Index, error) {
	if data, err := b.Store.Boundaries(ctx, dataset); err == nil {
		return geo.DecodeBoundaries(data)
	}

	adm1, err := b.Boundaries.Fetch(ctx, "ADM1")
	if err != nil {
		return nil, fmt.Errorf("fetch departments: %w", err)
	}

	adm2, err := b.Boundaries.Fetch(ctx, "ADM2")
	if err != nil {
		return nil, fmt.Errorf("fetch municipalities: %w", err)
	}

	municipalities, departments, err := BuildBoundaries(adm1, adm2)
	if err != nil {
		return nil, err
	}

	data, err := geo.EncodeBoundaries(municipalities, departments)
	if err != nil {
		return nil, err
	}

	if err := b.Store.PutBoundaries(ctx, dataset, data); err != nil {
		return nil, err
	}

	return geo.NewIndex(municipalities, departments), nil
}

func uniqueCells(municipalities []geo.Municipality) []geo.Cell {
	seen := map[string]geo.Cell{}

	for _, m := range municipalities {
		for _, s := range geo.CellShares(m.Geometry) {
			seen[s.Cell.ID()] = s.Cell
		}
	}

	out := make([]geo.Cell, 0, len(seen))
	for _, c := range seen {
		out = append(out, c)
	}

	slices.SortFunc(out, func(a, b geo.Cell) int { return strings.Compare(a.ID(), b.ID()) })

	return out
}

func (b *Builder) loadClimate(ctx context.Context, dataset string, cells []geo.Cell) error {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(b.parallelism())

	for _, c := range cells {
		g.Go(func() error { return b.loadCell(gctx, dataset, c) })
	}

	return g.Wait()
}

func (b *Builder) loadCell(ctx context.Context, dataset string, c geo.Cell) error {
	done, err := b.Store.Exists(ctx, store.CellMetaKey(dataset, c.ID()))
	if err != nil || done {
		return err
	}

	ps, err := b.Power.Daily(ctx, c, b.From, b.To)
	if err != nil {
		return fmt.Errorf("POWER %s: %w", c.ID(), err)
	}

	if err := b.Store.PutCellSeries(ctx, dataset, c.ID(), ps.Series); err != nil {
		return err
	}

	excluded := agro.MonthlyPrecipitationOutliers(ps.Series)
	b.Logger.InfoContext(ctx, "cell loaded", slog.String("cell", c.ID()), slog.Int("excluded_months", len(excluded)))

	return b.Store.PutCellMeta(ctx, dataset, store.CellMeta{
		Cell: c, Elevation: ps.Elevation, Excluded: excluded, Source: strings.Join(ps.Sources, ","),
	})
}

func (b *Builder) loadProfiles(ctx context.Context, dataset string, ix *geo.Index, municipalities []geo.Municipality) (int, error) {
	var (
		mu    sync.Mutex
		built int
		fails []error
	)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(b.parallelism())

	for _, m := range municipalities {
		g.Go(func() error {
			err := b.loadProfile(gctx, dataset, ix, m)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				if gctx.Err() != nil {
					return gctx.Err()
				}

				b.Logger.ErrorContext(gctx, "profile failed", slog.String("municipality", m.Code), slog.String("error", err.Error()))
				fails = append(fails, err)

				return nil
			}

			built++

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return built, err
	}

	if len(fails) > 0 {
		b.Logger.WarnContext(ctx, "some profiles failed", slog.Int("failed", len(fails)), slog.String("first", fails[0].Error()))
	}

	return built, nil
}

func (b *Builder) loadProfile(ctx context.Context, dataset string, ix *geo.Index, m geo.Municipality) error {
	done, err := b.Store.Exists(ctx, store.ProfileKey(dataset, m.Code))
	if err != nil {
		return err
	}

	if done {
		return nil
	}

	profile, cells, err := b.baseProfile(ctx, dataset, ix, m)
	if err != nil {
		return err
	}

	series, err := agro.BuildSiteSeries(cells, agro.SiteSpec{
		LatitudeDeg: profile.LatitudeDeg, Elevation: profile.ElevationMedian, Excluded: agro.ExcludedSet(profile.ExcludedMonths),
	})
	if err != nil {
		return fmt.Errorf("site series %s: %w", m.Code, err)
	}

	profile.FillClimate(series)

	return b.Store.PutProfile(ctx, dataset, profile)
}

func (b *Builder) baseProfile(ctx context.Context, dataset string, ix *geo.Index, m geo.Municipality) (*agro.SiteProfile, []agro.WeightedCell, error) {
	center := geo.InteriorPoint(m.Geometry)
	dep, _ := ix.Department(m.DepartmentCode)
	profile := &agro.SiteProfile{
		MunicipalityCode: m.Code, MunicipalityName: m.Name, DepartmentCode: m.DepartmentCode, DepartmentName: dep.Name,
		LatitudeDeg: center.Lat(), LongitudeDeg: center.Lon(), DatasetVersion: dataset,
	}

	excluded := map[agro.YearMonth]bool{}

	var cells []agro.WeightedCell

	for _, share := range geo.CellShares(m.Geometry) {
		meta, err := b.Store.CellMeta(ctx, dataset, share.Cell.ID())
		if err != nil {
			return nil, nil, fmt.Errorf("cell meta %s: %w", share.Cell.ID(), err)
		}

		series, err := b.Store.CellSeries(ctx, dataset, share.Cell.ID())
		if err != nil {
			return nil, nil, fmt.Errorf("cell series %s: %w", share.Cell.ID(), err)
		}

		for _, ym := range meta.Excluded {
			excluded[ym] = true
		}

		profile.Cells = append(profile.Cells, agro.CellWeight{
			CellID: share.Cell.ID(), Latitude: share.Cell.Lat, Longitude: share.Cell.Lon, Weight: share.Share, Elevation: meta.Elevation,
		})
		cells = append(cells, agro.WeightedCell{Series: series, Weight: share.Share, Elevation: meta.Elevation})
	}

	for ym := range excluded {
		profile.ExcludedMonths = append(profile.ExcludedMonths, ym)
	}

	if err := b.fillElevation(ctx, profile, m); err != nil {
		return nil, nil, err
	}

	soil, err := b.Soil.Summary(ctx, m)
	if err != nil {
		return nil, nil, err
	}

	profile.Soil = soil

	return profile, cells, nil
}

func (b *Builder) fillElevation(ctx context.Context, p *agro.SiteProfile, m geo.Municipality) error {
	points := geo.SamplePoints(m.Geometry, elevationSpacingDeg, minimumElevationPts)

	elevations, err := b.OpenMeteo.Elevations(ctx, points)
	if err != nil {
		return fmt.Errorf("elevation %s: %w", m.Code, err)
	}

	lo, hi := math.Inf(1), math.Inf(-1)
	for _, e := range elevations {
		lo, hi = math.Min(lo, e), math.Max(hi, e)
	}

	p.ElevationMedian = agro.Meters(math.Round(agro.Percentile(elevations, 0.5)))
	p.ElevationMin, p.ElevationMax = agro.Meters(lo), agro.Meters(hi)
	p.ElevationSamples = len(elevations)

	return nil
}
