package plan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/dma-teams/nasa-app-api/internal/agro"
)

const (
	defaultComputeTimeout = 10 * time.Second
	defaultResultTTL      = 35 * 24 * time.Hour
	staleForecastAge      = 45 * 24 * time.Hour
	wideElevationRangeM   = 800
	planningHorizonDays   = 365
)

type Options struct {
	Irrigated       bool
	Priority        agro.Priority
	CurrentRotation []string
	CurrentCrop     string
	FieldFreeFrom   time.Time
}

type Dependencies struct {
	Profiles  ProfileReader
	Climate   ClimateReader
	Forecasts ForecastReader
	Cache     ResultCache
	Versions  VersionSource
	Gazetteer Gazetteer
	Engine    *agro.Engine
	Logger    *slog.Logger
}

type Config struct {
	CatalogHash    string
	Location       *time.Location
	Now            func() time.Time
	ComputeTimeout time.Duration
	ResultTTL      time.Duration
	MaxParallel    int
}

type Service struct {
	deps  Dependencies
	cfg   Config
	group singleflight.Group
}

func NewService(cfg Config, deps Dependencies) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	if cfg.Location == nil {
		cfg.Location = time.UTC
	}

	if cfg.ComputeTimeout <= 0 {
		cfg.ComputeTimeout = defaultComputeTimeout
	}

	if cfg.ResultTTL <= 0 {
		cfg.ResultTTL = defaultResultTTL
	}

	if cfg.MaxParallel <= 0 {
		cfg.MaxParallel = 1
	}

	if deps.Logger == nil {
		deps.Logger = slog.New(slog.DiscardHandler)
	}

	return &Service{deps: deps, cfg: cfg}
}

func (s *Service) Today() time.Time {
	now := s.cfg.Now().In(s.cfg.Location)

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *Service) versions() (Versions, error) {
	v, ok := s.deps.Versions.Versions()
	if !ok || v.Dataset == "" {
		return Versions{}, ErrNotReady
	}

	v.Catalog = s.cfg.CatalogHash

	return v, nil
}

func (s *Service) MunicipalPlan(ctx context.Context, code string, opts Options) (*MunicipalPlan, error) {
	versions, err := s.versions()
	if err != nil {
		return nil, err
	}

	mun, err := s.deps.Gazetteer.Municipality(code)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", code, ErrMunicipalityNotFound)
	}

	start := s.Today()
	if err := s.validate(start, &opts); err != nil {
		return nil, err
	}

	key := resultHash(code, start, opts, versions)

	if cached, ok := s.cached(ctx, key); ok {
		return cached, nil
	}

	ch := s.group.DoChan(key, func() (any, error) {
		computeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.ComputeTimeout)
		defer cancel()

		p, err := s.compute(computeCtx, Municipality{Code: mun.Code, Name: mun.Name, DepartmentCode: mun.DepartmentCode}, start, opts, versions)
		if err != nil {
			return nil, err
		}

		s.store(computeCtx, key, p)

		return p, nil
	})

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("municipal plan %s: %w", code, ctx.Err())
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}

		p, _ := res.Val.(*MunicipalPlan)

		return p, nil
	}
}

func (s *Service) validate(start time.Time, opts *Options) error {
	if opts.Priority == agro.PriorityUnknown {
		opts.Priority = agro.PriorityBalanced
	}

	for _, id := range opts.CurrentRotation {
		if _, ok := s.deps.Engine.Crop(id); !ok {
			return &InputError{Field: "currentRotation", Message: fmt.Sprintf("unknown crop %q", id)}
		}
	}

	if opts.CurrentCrop != "" {
		if _, ok := s.deps.Engine.Crop(opts.CurrentCrop); !ok {
			return &InputError{Field: "currentCrop", Message: fmt.Sprintf("unknown crop %q", opts.CurrentCrop)}
		}
	}

	if opts.FieldFreeFrom.IsZero() {
		opts.FieldFreeFrom = start
	}

	offset := int(opts.FieldFreeFrom.Sub(start).Hours() / 24)
	if offset < 0 || offset >= planningHorizonDays {
		return &InputError{Field: "fieldFreeFrom", Message: "must be between today and 364 days from today"}
	}

	return nil
}

func resultHash(code string, start time.Time, opts Options, v Versions) string {
	payload, _ := json.Marshal(struct {
		Code     string
		Start    string
		Options  Options
		Versions Versions
	}{code, start.Format(time.DateOnly), opts, v})
	sum := sha256.Sum256(payload)

	return hex.EncodeToString(sum[:16])
}

func (s *Service) cached(ctx context.Context, key string) (*MunicipalPlan, bool) {
	data, ok, err := s.deps.Cache.Result(ctx, key)
	if err != nil {
		s.deps.Logger.WarnContext(ctx, "result cache read failed", slog.String("error", err.Error()))

		return nil, false
	}

	if !ok {
		return nil, false
	}

	var p MunicipalPlan
	if err := json.Unmarshal(data, &p); err != nil {
		s.deps.Logger.WarnContext(ctx, "cached result is corrupt", slog.String("error", err.Error()))

		return nil, false
	}

	return &p, true
}

func (s *Service) store(ctx context.Context, key string, p *MunicipalPlan) {
	data, err := json.Marshal(p)
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "encode result failed", slog.String("error", err.Error()))

		return
	}

	if err := s.deps.Cache.PutResult(ctx, key, data, s.cfg.ResultTTL); err != nil {
		s.deps.Logger.WarnContext(ctx, "result cache write failed", slog.String("error", err.Error()))
	}
}

func (s *Service) compute(ctx context.Context, mun Municipality, start time.Time, opts Options, v Versions) (*MunicipalPlan, error) {
	profile, err := s.deps.Profiles.Profile(ctx, v.Dataset, mun.Code)
	if err != nil {
		return nil, fmt.Errorf("profile %s: %w", mun.Code, errors.Join(ErrDataUnavailable, err))
	}

	series, err := s.siteSeries(ctx, v.Dataset, profile)
	if err != nil {
		return nil, err
	}

	forecast, warnings := s.forecast(ctx, v.Forecast, profile, start)

	eval, err := s.deps.Engine.Evaluate(ctx, agro.EvaluationRequest{
		Start: start, Series: series, Profile: profile, Irrigated: opts.Irrigated, Priority: opts.Priority,
		CurrentRotation: opts.CurrentRotation, CurrentCrop: opts.CurrentCrop,
		FieldFreeOffset: int(opts.FieldFreeFrom.Sub(start).Hours() / 24), Forecast: forecast,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluate %s: %w", mun.Code, err)
	}

	mun.DepartmentName = profile.DepartmentName

	return &MunicipalPlan{
		Municipality: mun,
		GeneratedAt:  s.cfg.Now().UTC().Format(time.RFC3339),
		Site:         siteOf(profile, forecast),
		Evaluation:   eval,
		Versions:     v,
		Warnings:     append(warnings, profileWarnings(profile)...),
		Sources:      Sources,
		Disclaimer:   Disclaimer,
	}, nil
}

func (s *Service) siteSeries(ctx context.Context, dataset string, p *agro.SiteProfile) (*agro.SiteSeries, error) {
	cells := make([]agro.WeightedCell, 0, len(p.Cells))

	for _, c := range p.Cells {
		series, err := s.deps.Climate.CellSeries(ctx, dataset, c.CellID)
		if err != nil {
			return nil, fmt.Errorf("climate %s: %w", c.CellID, errors.Join(ErrDataUnavailable, err))
		}

		cells = append(cells, agro.WeightedCell{Series: series, Weight: c.Weight, Elevation: c.Elevation})
	}

	elevation := p.ElevationMedian
	if elevation == 0 && len(p.Cells) > 0 {
		elevation = p.Cells[0].Elevation
	}

	series, err := agro.BuildSiteSeries(cells, agro.SiteSpec{
		LatitudeDeg: p.LatitudeDeg, Elevation: elevation, Excluded: agro.ExcludedSet(p.ExcludedMonths),
	})
	if err != nil {
		return nil, fmt.Errorf("site series %s: %w", p.MunicipalityCode, errors.Join(ErrDataUnavailable, err))
	}

	return series, nil
}

func (s *Service) forecast(ctx context.Context, version string, p *agro.SiteProfile, start time.Time) ([]*agro.TercileProbabilities, []string) {
	if version == "" {
		return nil, []string{"forecastUnavailable"}
	}

	forecasts := make([]agro.CellForecast, 0, len(p.Cells))
	weights := make([]float64, 0, len(p.Cells))
	warnings := []string{}

	for _, c := range p.Cells {
		f, err := s.deps.Forecasts.Forecast(ctx, version, c.CellID)
		if err != nil {
			s.deps.Logger.WarnContext(ctx, "forecast missing for cell", slog.String("cell", c.CellID), slog.String("error", err.Error()))

			continue
		}

		if issued, err := time.Parse(time.DateOnly, f.Issued); err == nil && s.cfg.Now().Sub(issued) > staleForecastAge && !slices.Contains(warnings, "forecastStale") {
			warnings = append(warnings, "forecastStale")
		}

		forecasts = append(forecasts, f)
		weights = append(weights, c.Weight)
	}

	if len(forecasts) == 0 {
		return nil, append(warnings, "forecastUnavailable")
	}

	return agro.BlendForecasts(forecasts, weights, start), warnings
}

func siteOf(p *agro.SiteProfile, forecast []*agro.TercileProbabilities) Site {
	lo, hi := p.Soil.AvailableWaterRange()

	return Site{
		ElevationMedianM: float64(p.ElevationMedian), ElevationMinM: float64(p.ElevationMin), ElevationMaxM: float64(p.ElevationMax),
		Cells: p.Cells, Soil: p.Soil,
		AvailableWaterMmPerM:  math.Round(float64(p.Soil.AvailableWaterPerMeter())),
		AvailableWaterRangeMm: [2]float64{math.Round(float64(lo)), math.Round(float64(hi))},
		Fortnights:            p.Fortnights[:], OnsetMedianDayOfYear: p.OnsetMedianDOY,
		Canicula: p.Canicula, CaniculaYears: p.CaniculaYears, Years: p.Years,
		ExcludedMonths: p.ExcludedMonths, Forecast: forecast,
	}
}

func profileWarnings(p *agro.SiteProfile) []string {
	var out []string

	if p.ElevationMax-p.ElevationMin > wideElevationRangeM {
		out = append(out, "wideElevationRange")
	}

	if len(p.ExcludedMonths) > 0 {
		out = append(out, "climateMonthsExcludedByQualityControl")
	}

	if strings.HasPrefix(p.Soil.Method, "SoilGrids REST") {
		out = append(out, "soilFromSinglePoint")
	}

	if p.Soil.ClayP95-p.Soil.ClayP05 > 40 {
		out = append(out, "soilHighUncertainty")
	}

	return out
}
