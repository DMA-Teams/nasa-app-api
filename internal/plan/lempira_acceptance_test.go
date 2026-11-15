package plan_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/catalog"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/plan"
	"github.com/dma-teams/nasa-app-api/internal/store"
)

const (
	lempiraDir      = "../../testdata/lempira"
	lempiraDataset  = "d-dev-lempira"
	lempiraForecast = "f2026-11-15"
	maxBeanLoss     = 0.30
)

var lempiraStart = time.Date(2026, time.November, 15, 12, 0, 0, 0, time.UTC)

type recordedLempira struct {
	profiles map[string]*agro.SiteProfile
	mu       sync.Mutex
	series   map[string]*agro.CellSeries
}

func loadRecordedLempira(tb testing.TB) *recordedLempira {
	tb.Helper()

	files, err := filepath.Glob(filepath.Join(lempiraDir, "profile", "*.json"))
	if err != nil || len(files) == 0 {
		tb.Fatalf("recorded Lempira profiles not found in %s: %v", lempiraDir, err)
	}

	r := &recordedLempira{profiles: map[string]*agro.SiteProfile{}, series: map[string]*agro.CellSeries{}}

	for _, f := range files {
		var p agro.SiteProfile
		readJSON(tb, f, &p)
		r.profiles[p.MunicipalityCode] = &p
	}

	return r
}

func readJSON(tb testing.TB, path string, v any) {
	tb.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatal(err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		tb.Fatalf("%s: %v", path, err)
	}
}

func (r *recordedLempira) Profile(_ context.Context, _, code string) (*agro.SiteProfile, error) {
	p, ok := r.profiles[code]
	if !ok {
		return nil, os.ErrNotExist
	}

	return p, nil
}

func (r *recordedLempira) CellSeries(_ context.Context, _, cell string) (*agro.CellSeries, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s, ok := r.series[cell]; ok {
		return s, nil
	}

	compressed, err := os.ReadFile(filepath.Join(lempiraDir, "climate", cell+".bin.gz"))
	if err != nil {
		return nil, err
	}

	zr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}

	s, err := store.DecodeCellSeries(data)
	if err != nil {
		return nil, err
	}

	r.series[cell] = s

	return s, nil
}

func (r *recordedLempira) Forecast(_ context.Context, _, cell string) (agro.CellForecast, error) {
	var f agro.CellForecast

	data, err := os.ReadFile(filepath.Join(lempiraDir, "forecast", cell+".json"))
	if err != nil {
		return f, err
	}

	return f, json.Unmarshal(data, &f)
}

func (r *recordedLempira) Result(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}

func (r *recordedLempira) PutResult(context.Context, string, []byte, time.Duration) error {
	return nil
}

func (r *recordedLempira) Versions() (plan.Versions, bool) {
	return plan.Versions{Dataset: lempiraDataset, Forecast: lempiraForecast}, true
}

func (r *recordedLempira) gazetteer() *geo.Index {
	municipalities := make([]geo.Municipality, 0, len(r.profiles))
	for _, p := range r.profiles {
		municipalities = append(municipalities, geo.Municipality{Code: p.MunicipalityCode, Name: p.MunicipalityName, DepartmentCode: p.DepartmentCode})
	}

	return geo.NewIndex(municipalities, []geo.Department{{Code: "HN-LE", Name: "Lempira"}})
}

func (r *recordedLempira) codes() []string {
	codes := make([]string, 0, len(r.profiles))
	for code := range r.profiles {
		codes = append(codes, code)
	}

	slices.Sort(codes)

	return codes
}

func lempiraPlans(t *testing.T) []*plan.MunicipalPlan {
	t.Helper()

	data := loadRecordedLempira(t)

	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}

	engine, err := agro.NewEngine(cat.Crops, cat.Templates)
	if err != nil {
		t.Fatal(err)
	}

	svc := plan.NewService(
		plan.Config{CatalogHash: cat.Hash, Now: func() time.Time { return lempiraStart }, ComputeTimeout: time.Minute},
		plan.Dependencies{Profiles: data, Climate: data, Forecasts: data, Cache: data, Versions: data, Gazetteer: data.gazetteer(), Engine: engine},
	)

	plans := make([]*plan.MunicipalPlan, 0, len(data.profiles))

	for _, code := range data.codes() {
		p, err := svc.MunicipalPlan(context.Background(), code, plan.Options{CurrentRotation: []string{"maize", "beans"}})
		if err != nil {
			t.Fatalf("plan %s: %v", code, err)
		}

		plans = append(plans, p)
		logPlan(t, p)
	}

	return plans
}

func logPlan(t *testing.T, p *plan.MunicipalPlan) {
	t.Helper()

	for _, r := range p.Evaluation.Rotations {
		if r.ID != p.Evaluation.Best && r.ID != "maize-beans" {
			continue
		}

		steps := make([]string, len(r.Plantings))
		for i, x := range r.Plantings {
			steps[i] = x.CropID + " " + x.SowDate + ">" + x.HarvestDate
			if x.Relay {
				steps[i] += " relevo"
			}
		}

		t.Logf("%-22s %-12s %5.1f  %s", p.Municipality.Name, r.ID, r.Scores.Total, strings.Join(steps, " | "))
	}
}

func rotation(p *plan.MunicipalPlan, id string) (agro.RotationResult, bool) {
	for _, r := range p.Evaluation.Rotations {
		if r.ID == id {
			return r, true
		}
	}

	return agro.RotationResult{}, false
}

func TestLempiraAcceptance(t *testing.T) {
	if testing.Short() {
		t.Skip("acceptance with recorded Lempira data")
	}

	plans := lempiraPlans(t)

	checks := []struct {
		name  string
		check func(t *testing.T, p *plan.MunicipalPlan)
	}{
		{"maize-beans is viable with bean loss below 30 percent", checkMaizeBeansViable},
		{"cover crops are relayed only into maize or sorghum", checkCoverRelayHosts},
		{"generated rotations include a harvest crop", checkGeneratedHarvest},
		{"no rainfed sowing between January and March", checkNoDrySeasonSowing},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			for _, p := range plans {
				c.check(t, p)
			}
		})
	}
}

func checkMaizeBeansViable(t *testing.T, p *plan.MunicipalPlan) {
	t.Helper()

	r, ok := rotation(p, "maize-beans")
	if !ok || !r.Feasible || len(r.Plantings) != 2 {
		t.Errorf("%s: maize-beans not feasible", p.Municipality.Name)

		return
	}

	if loss := r.Plantings[1].LossProbability; loss >= maxBeanLoss {
		t.Errorf("%s: beans sown %s lose %.0f%%, want < %.0f%%", p.Municipality.Name, r.Plantings[1].SowDate, loss*100, maxBeanLoss*100)
	}
}

func checkCoverRelayHosts(t *testing.T, p *plan.MunicipalPlan) {
	t.Helper()

	for _, r := range p.Evaluation.Rotations {
		for i, x := range r.Plantings {
			if !x.Relay || x.Role != "cover" {
				continue
			}

			if host := r.Plantings[i-1].CropID; host != "maize" && host != "sorghum" {
				t.Errorf("%s %s: %s relayed into %s", p.Municipality.Name, r.ID, x.CropID, host)
			}
		}
	}
}

func checkGeneratedHarvest(t *testing.T, p *plan.MunicipalPlan) {
	t.Helper()

	for _, r := range p.Evaluation.Rotations {
		if r.Kind == agro.KindGenerated && !slices.ContainsFunc(r.Plantings, func(x agro.PlantingResult) bool { return x.Role == "harvest" }) {
			t.Errorf("%s %s: only cover crops (%s)", p.Municipality.Name, r.ID, r.Signature)
		}
	}
}

func checkNoDrySeasonSowing(t *testing.T, p *plan.MunicipalPlan) {
	t.Helper()

	for _, r := range p.Evaluation.Rotations {
		for _, x := range r.Plantings {
			if sowDate(t, x).Month() <= time.March {
				t.Errorf("%s %s: rainfed %s sown %s", p.Municipality.Name, r.ID, x.CropID, x.SowDate)
			}
		}
	}
}

func sowDate(t *testing.T, x agro.PlantingResult) time.Time {
	t.Helper()

	sow, err := time.Parse(time.DateOnly, x.SowDate)
	if err != nil {
		t.Fatal(err)
	}

	return sow
}
