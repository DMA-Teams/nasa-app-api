package plan

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/paulmach/orb"
	"go.uber.org/goleak"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/catalog"
	"github.com/dma-teams/nasa-app-api/internal/geo"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

const testCell = "14.50_-88.750"

func dryCorridorCell() *agro.CellSeries {
	monthly := [12]float64{25, 17, 15, 30, 126, 197, 138, 173, 206, 154, 54, 34}
	start := time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)
	days := int(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Sub(start).Hours() / 24)
	s := &agro.CellSeries{
		Start: start, Precipitation: make([]agro.Millimeters, days), TMean: make([]agro.Celsius, days),
		TMax: make([]agro.Celsius, days), TMin: make([]agro.Celsius, days), RelativeHumid: make([]float64, days),
		DewPoint: make([]agro.Celsius, days), Wind: make([]agro.MetersPerSecond, days),
		SolarRadiation: make([]agro.MegajoulesPerSquareMeterDay, days), Pressure: make([]agro.Kilopascals, days),
	}

	for i := range days {
		d := start.AddDate(0, 0, i)
		rain := monthly[d.Month()-1] / 15

		if d.Day()%2 == 0 || (d.Month() == time.July && d.Day() >= 5 && d.Day() <= 28) {
			rain = 0
		}

		s.Precipitation[i] = agro.Millimeters(rain)
		s.TMean[i], s.TMax[i], s.TMin[i] = 23, 30, 17
		s.RelativeHumid[i], s.DewPoint[i], s.Wind[i] = 70, 16, 2
		s.SolarRadiation[i] = agro.MegajoulesPerSquareMeterDay(19 + 3*math.Sin(2*math.Pi*float64(d.YearDay()-80)/365))
		s.Pressure[i] = 89.9
	}

	return s
}

type fakeData struct {
	series       *agro.CellSeries
	profiles     map[string]*agro.SiteProfile
	climateReads atomic.Int32
	mu           sync.Mutex
	results      map[string][]byte
	ready        bool
}

func (f *fakeData) Profile(_ context.Context, _, code string) (*agro.SiteProfile, error) {
	p, ok := f.profiles[code]
	if !ok {
		return nil, errors.New("missing profile")
	}

	return p, nil
}

func (f *fakeData) CellSeries(context.Context, string, string) (*agro.CellSeries, error) {
	f.climateReads.Add(1)
	time.Sleep(20 * time.Millisecond)

	return f.series, nil
}

func (f *fakeData) Forecast(context.Context, string, string) (agro.CellForecast, error) {
	return agro.CellForecast{Issued: "2026-11-05", Months: []agro.MonthlyTerciles{
		{Year: 2026, Month: time.December, Probabilities: agro.TercileProbabilities{Below: 0.6, Normal: 0.3, Above: 0.1}},
	}}, nil
}

func (f *fakeData) Result(_ context.Context, hash string) ([]byte, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, ok := f.results[hash]

	return data, ok, nil
}

func (f *fakeData) PutResult(_ context.Context, hash string, data []byte, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.results[hash] = data

	return nil
}

func (f *fakeData) Versions() (Versions, bool) {
	return Versions{Dataset: "d1", Forecast: "f1"}, f.ready
}

func square(lon, lat float64) orb.MultiPolygon {
	return orb.MultiPolygon{{{{lon, lat}, {lon + 0.1, lat}, {lon + 0.1, lat + 0.1}, {lon, lat + 0.1}, {lon, lat}}}}
}

func newTestService(tb testing.TB) (*Service, *fakeData) {
	tb.Helper()

	cat, err := catalog.Load()
	if err != nil {
		tb.Fatal(err)
	}

	engine, err := agro.NewEngine(cat.Crops, cat.Templates)
	if err != nil {
		tb.Fatal(err)
	}

	data := &fakeData{series: dryCorridorCell(), profiles: map[string]*agro.SiteProfile{}, results: map[string][]byte{}, ready: true}
	site, err := agro.BuildSiteSeries([]agro.WeightedCell{{Series: data.series, Weight: 1, Elevation: 1000}}, agro.SiteSpec{LatitudeDeg: 14.3, Elevation: 1000})
	if err != nil {
		tb.Fatal(err)
	}

	for _, code := range []string{"M1", "M2"} {
		p := &agro.SiteProfile{
			MunicipalityCode: code, DepartmentCode: "D1", DepartmentName: "Lempira", LatitudeDeg: 14.3,
			Cells:           []agro.CellWeight{{CellID: testCell, Weight: 1, Elevation: 1000}},
			ElevationMedian: 1000, ElevationMin: 700, ElevationMax: 1300,
			Soil: agro.SoilSummary{ClayPercent: 30, SandPercent: 35, SiltPercent: 35, OrganicCarbonGPerKg: 20, PH: 6.3,
				ClayP05: 10, ClayP95: 60, SandP05: 15, SandP95: 55, Method: "SoilGrids REST: punto"},
		}
		p.FillClimate(site)
		data.profiles[code] = p
	}

	gaz := geo.NewIndex(
		[]geo.Municipality{
			{Code: "M1", Name: "Uno", DepartmentCode: "D1", Geometry: square(-88.7, 14.2)},
			{Code: "M2", Name: "Dos", DepartmentCode: "D1", Geometry: square(-88.5, 14.2)},
		},
		[]geo.Department{{Code: "D1", Name: "Lempira"}},
	)

	svc := NewService(
		Config{CatalogHash: cat.Hash, Now: func() time.Time { return time.Date(2026, 11, 15, 15, 0, 0, 0, time.UTC) }, MaxParallel: 2},
		Dependencies{Profiles: data, Climate: data, Forecasts: data, Cache: data, Versions: data, Gazetteer: gaz, Engine: engine},
	)

	return svc, data
}

func TestMunicipalPlan_ComputesAndCaches(t *testing.T) {
	t.Parallel()

	svc, data := newTestService(t)

	p, err := svc.MunicipalPlan(context.Background(), "M1", Options{CurrentRotation: []string{"maize", "beans"}})
	if err != nil {
		t.Fatalf("MunicipalPlan: %v", err)
	}

	if len(p.Evaluation.Rotations) < 3 || p.Evaluation.Rotations[0].Kind != agro.KindCurrent {
		t.Fatalf("rotations = %d, first %q", len(p.Evaluation.Rotations), p.Evaluation.Rotations[0].Kind)
	}

	if p.Versions.Catalog == "" || p.Disclaimer == "" || len(p.Sources) == 0 || p.Evaluation.ForecastMonths == 0 {
		t.Errorf("plan metadata incomplete: versions %+v forecast months %d", p.Versions, p.Evaluation.ForecastMonths)
	}

	if _, err := svc.MunicipalPlan(context.Background(), "M1", Options{CurrentRotation: []string{"maize", "beans"}}); err != nil {
		t.Fatalf("second MunicipalPlan: %v", err)
	}

	if reads := data.climateReads.Load(); reads != 1 {
		t.Errorf("climate reads = %d, want 1 (second call served from cache)", reads)
	}
}

func TestMunicipalPlan_ConcurrentRequestsComputeOnce(t *testing.T) {
	t.Parallel()

	svc, data := newTestService(t)

	var wg sync.WaitGroup

	errs := make(chan error, 8)

	for range 8 {
		wg.Go(func() {
			if _, err := svc.MunicipalPlan(context.Background(), "M2", Options{}); err != nil {
				errs <- err
			}
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("MunicipalPlan: %v", err)
	}

	if reads := data.climateReads.Load(); reads != 1 {
		t.Errorf("climate reads = %d, want 1 with singleflight", reads)
	}
}

func TestMunicipalPlan_Errors(t *testing.T) {
	t.Parallel()

	svc, data := newTestService(t)
	ctx := context.Background()

	if _, err := svc.MunicipalPlan(ctx, "NOPE", Options{}); !errors.Is(err, ErrMunicipalityNotFound) {
		t.Errorf("unknown municipality err = %v", err)
	}

	var input *InputError
	if _, err := svc.MunicipalPlan(ctx, "M1", Options{CurrentRotation: []string{"teff"}}); !errors.As(err, &input) || input.Field != "currentRotation" {
		t.Errorf("unknown crop err = %v", err)
	}

	if _, err := svc.MunicipalPlan(ctx, "M1", Options{FieldFreeFrom: time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("field free date err = %v", err)
	}

	data.ready = false
	if _, err := svc.MunicipalPlan(ctx, "M1", Options{}); !errors.Is(err, ErrNotReady) {
		t.Errorf("not ready err = %v", err)
	}
}

func TestMunicipalPlan_CallerCancellationDoesNotBreakSharedComputation(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := svc.MunicipalPlan(ctx, "M1", Options{}); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled caller err = %v, want context.Canceled", err)
	}

	if _, err := svc.MunicipalPlan(context.Background(), "M1", Options{}); err != nil {
		t.Errorf("follow-up call failed: %v", err)
	}
}

func TestDepartmentPlan_GroupsMunicipalitiesWithSameRecommendation(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)

	d, err := svc.DepartmentPlan(context.Background(), "D1", Options{})
	if err != nil {
		t.Fatalf("DepartmentPlan: %v", err)
	}

	if d.Municipalities != 2 || len(d.Groups) != 1 || len(d.Groups[0].Municipalities) != 2 {
		t.Fatalf("department plan = %d municipalities, %d groups", d.Municipalities, len(d.Groups))
	}

	if _, err := svc.DepartmentPlan(context.Background(), "D9", Options{}); !errors.Is(err, ErrDepartmentNotFound) {
		t.Errorf("unknown department err = %v", err)
	}
}

func BenchmarkMunicipalPlanWithoutCache(b *testing.B) {
	svc, data := newTestService(b)
	opts := Options{CurrentRotation: []string{"maize", "beans"}}

	for b.Loop() {
		data.mu.Lock()
		clear(data.results)
		data.mu.Unlock()

		if _, err := svc.MunicipalPlan(context.Background(), "M1", opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMunicipalPlanCached(b *testing.B) {
	svc, _ := newTestService(b)
	opts := Options{CurrentRotation: []string{"maize", "beans"}}

	if _, err := svc.MunicipalPlan(context.Background(), "M1", opts); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		if _, err := svc.MunicipalPlan(context.Background(), "M1", opts); err != nil {
			b.Fatal(err)
		}
	}
}
