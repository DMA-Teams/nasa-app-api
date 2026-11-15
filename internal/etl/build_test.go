package etl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/sources"
	"github.com/dma-teams/nasa-app-api/internal/store"
)

type memoryStore struct {
	mu        sync.Mutex
	keys      map[string]bool
	bounds    map[string][]byte
	series    map[string]*agro.CellSeries
	metas     map[string]store.CellMeta
	profiles  map[string]*agro.SiteProfile
	forecasts map[string]agro.CellForecast
	versions  store.Versions
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		keys: map[string]bool{}, bounds: map[string][]byte{}, series: map[string]*agro.CellSeries{},
		metas: map[string]store.CellMeta{}, profiles: map[string]*agro.SiteProfile{}, forecasts: map[string]agro.CellForecast{},
	}
}

func (m *memoryStore) Exists(_ context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.keys[key], nil
}

func (m *memoryStore) mark(key string) { m.keys[key] = true }

func (m *memoryStore) PutBoundaries(_ context.Context, d string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bounds[d] = data

	return nil
}

func (m *memoryStore) Boundaries(_ context.Context, d string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, ok := m.bounds[d]
	if !ok {
		return nil, store.ErrNotFound
	}

	return data, nil
}

func (m *memoryStore) PutCellSeries(_ context.Context, d, c string, s *agro.CellSeries) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.series[d+c] = s
	m.mark(store.ClimateKey(d, c))

	return nil
}

func (m *memoryStore) CellSeries(_ context.Context, d, c string) (*agro.CellSeries, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.series[d+c]
	if !ok {
		return nil, store.ErrNotFound
	}

	return s, nil
}

func (m *memoryStore) PutCellMeta(_ context.Context, d string, meta store.CellMeta) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metas[d+meta.Cell.ID()] = meta
	m.mark(store.CellMetaKey(d, meta.Cell.ID()))

	return nil
}

func (m *memoryStore) CellMeta(_ context.Context, d, c string) (store.CellMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.metas[d+c], nil
}

func (m *memoryStore) PutProfile(_ context.Context, d string, p *agro.SiteProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[p.MunicipalityCode] = p
	m.mark(store.ProfileKey(d, p.MunicipalityCode))

	return nil
}

func (m *memoryStore) PutForecast(_ context.Context, v, c string, f agro.CellForecast) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forecasts[v+c] = f

	return nil
}

func (m *memoryStore) SetCurrentDataset(_ context.Context, v string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.versions.Dataset = v

	return nil
}

func (m *memoryStore) SetCurrentForecast(_ context.Context, v string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.versions.Forecast = v

	return nil
}

func (m *memoryStore) CurrentVersions(context.Context) (store.Versions, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.versions.Dataset == "" {
		return m.versions, store.ErrNotFound
	}

	return m.versions, nil
}

const (
	adm1JSON = `{"features":[{"properties":{"shapeName":"Lempira","shapeISO":"HN-LE"},"geometry":{"type":"Polygon","coordinates":[[[-88.9,14.0],[-88.3,14.0],[-88.3,14.6],[-88.9,14.6],[-88.9,14.0]]]}}]}`
	adm2JSON = `{"features":[
{"properties":{"shapeName":"Gracias","shapeID":"M-1"},"geometry":{"type":"Polygon","coordinates":[[[-88.7,14.3],[-88.6,14.3],[-88.6,14.4],[-88.7,14.4],[-88.7,14.3]]]}},
{"properties":{"shapeName":"Erandique","shapeID":"M-2"},"geometry":{"type":"MultiPolygon","coordinates":[[[[-88.6,14.3],[-88.5,14.3],[-88.5,14.4],[-88.6,14.4],[-88.6,14.3]]]]}}]}`
	soilJSON = `{"properties":{"layers":[` +
		`{"name":"clay","unit_measure":{"d_factor":10},"depths":[{"label":"0-5cm","values":{"mean":300,"Q0.05":100,"Q0.95":500}},{"label":"5-15cm","values":{"mean":300,"Q0.05":100,"Q0.95":500}},{"label":"15-30cm","values":{"mean":300,"Q0.05":100,"Q0.95":500}},{"label":"30-60cm","values":{"mean":350}}]},` +
		`{"name":"sand","unit_measure":{"d_factor":10},"depths":[{"label":"0-5cm","values":{"mean":350,"Q0.05":150,"Q0.95":550}},{"label":"5-15cm","values":{"mean":350,"Q0.05":150,"Q0.95":550}},{"label":"15-30cm","values":{"mean":350,"Q0.05":150,"Q0.95":550}},{"label":"30-60cm","values":{"mean":330}}]},` +
		`{"name":"phh2o","unit_measure":{"d_factor":10},"depths":[{"label":"0-5cm","values":{"mean":63}},{"label":"5-15cm","values":{"mean":63}},{"label":"15-30cm","values":{"mean":63}}]},` +
		`{"name":"soc","unit_measure":{"d_factor":10},"depths":[{"label":"0-5cm","values":{"mean":200}},{"label":"5-15cm","values":{"mean":150}},{"label":"15-30cm","values":{"mean":100}},{"label":"30-60cm","values":{"mean":60}}]}]}}`
)

type fakeWeb struct {
	mu    sync.Mutex
	hits  map[string]int
	from  time.Time
	until time.Time
}

func (f *fakeWeb) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.hits[r.URL.Host]++
	f.mu.Unlock()

	body := f.route(r)

	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}, Request: r}, nil
}

func (f *fakeWeb) route(r *http.Request) string {
	switch {
	case r.URL.Host == "www.geoboundaries.org":
		level := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/current/gbOpen/HND/"), "/")

		return fmt.Sprintf(`{"simplifiedGeometryGeoJSON":"https://files.test/%s.geojson"}`, level)
	case r.URL.Host == "files.test" && strings.Contains(r.URL.Path, "ADM1"):
		return adm1JSON
	case r.URL.Host == "files.test":
		return adm2JSON
	case r.URL.Host == "power.larc.nasa.gov":
		return powerJSON(f.from, f.until)
	case r.URL.Host == "api.open-meteo.com":
		n := len(strings.Split(r.URL.Query().Get("latitude"), ","))

		return `{"elevation":[` + strings.TrimSuffix(strings.Repeat("950,", n), ",") + `]}`
	case r.URL.Host == "rest.isric.org":
		return soilJSON
	case r.URL.Host == "seasonal-api.open-meteo.com" && r.URL.Query().Get("monthly") != "":
		return `{"monthly":{"time":["2027-01-01","2027-02-01"],"precipitation_anomaly":[-12.0,3.0]}}`
	default:
		return seasonalDailyJSON()
	}
}

func powerJSON(from, until time.Time) string {
	params := map[string]map[string]float64{}
	names := []string{"PRECTOTCORR", "T2M", "T2M_MAX", "T2M_MIN", "RH2M", "T2MDEW", "WS2M", "ALLSKY_SFC_SW_DWN", "PS"}
	monthly := [12]float64{25, 17, 15, 30, 126, 197, 138, 173, 206, 154, 54, 34}

	for _, n := range names {
		params[n] = map[string]float64{}
	}

	for d := from; !d.After(until); d = d.AddDate(0, 0, 1) {
		k := d.Format("20060102")
		rain := 0.0

		if d.Day()%2 == 1 {
			rain = monthly[d.Month()-1] / 15
		}

		params["PRECTOTCORR"][k] = rain
		params["T2M"][k], params["T2M_MAX"][k], params["T2M_MIN"][k] = 23, 30, 17
		params["RH2M"][k], params["T2MDEW"][k], params["WS2M"][k] = 70, 16, 2
		params["ALLSKY_SFC_SW_DWN"][k], params["PS"][k] = 19, 89.9
	}

	data, _ := json.Marshal(map[string]any{
		"geometry":   map[string]any{"coordinates": []float64{-88.75, 14.5, 1021.4}},
		"properties": map[string]any{"parameter": params},
		"header":     map[string]any{"fill_value": -999.0, "sources": []string{"MERRA2"}},
	})

	return string(data)
}

func seasonalDailyJSON() string {
	daily := map[string]any{}

	var times []string

	for d := time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC); d.Before(time.Date(2027, 3, 5, 0, 0, 0, 0, time.UTC)); d = d.AddDate(0, 0, 1) {
		times = append(times, d.Format(time.DateOnly))
	}

	daily["time"] = times

	for m := 1; m <= 20; m++ {
		values := make([]float64, len(times))
		for i := range values {
			values[i] = float64(m%5) * 0.4
		}

		daily[fmt.Sprintf("precipitation_sum_member%02d", m)] = values
	}

	data, _ := json.Marshal(map[string]any{"daily": daily})

	return string(data)
}

func TestBuilder_BuildAndForecastPublishVersionsAndResume(t *testing.T) {
	t.Parallel()

	from := time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	web := &fakeWeb{hits: map[string]int{}, from: from, until: to}
	client := sources.NewClient(time.Second, sources.WithTransport(web))
	st := newMemoryStore()

	b := &Builder{
		Store: st, Power: sources.NewPowerClient(client), OpenMeteo: sources.NewOpenMeteoClient(client),
		Boundaries: sources.NewBoundariesClient(client), Soil: RESTSoil{Client: sources.NewSoilGridsClient(client)},
		Logger: slog.New(slog.DiscardHandler), From: from, To: to,
	}

	ctx := context.Background()
	if err := b.Build(ctx, "d-test"); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if st.versions.Dataset != "d-test" || len(st.profiles) != 2 {
		t.Fatalf("dataset %q profiles %d, want d-test and 2", st.versions.Dataset, len(st.profiles))
	}

	p := st.profiles["M-1"]
	if p.DepartmentCode != "HN-LE" || p.ElevationMedian != 950 || p.Soil.PH != 6.3 || p.OnsetMedianDOY == 0 || p.Years != 30 {
		t.Errorf("profile = dep %s elev %v pH %v onset %d years %d", p.DepartmentCode, p.ElevationMedian, p.Soil.PH, p.OnsetMedianDOY, p.Years)
	}

	powerCalls := web.hits["power.larc.nasa.gov"]
	if err := b.Build(ctx, "d-test"); err != nil {
		t.Fatalf("second Build: %v", err)
	}

	if web.hits["power.larc.nasa.gov"] != powerCalls {
		t.Errorf("resumed build called POWER again: %d -> %d", powerCalls, web.hits["power.larc.nasa.gov"])
	}

	if err := b.Forecast(ctx, "f-test", time.Date(2026, 12, 6, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Forecast: %v", err)
	}

	if st.versions.Forecast != "f-test" {
		t.Fatalf("forecast version = %q", st.versions.Forecast)
	}

	for _, f := range st.forecasts {
		if len(f.Months) != 2 || f.Months[0].Month != time.January {
			t.Errorf("forecast months = %+v, want January and February 2027", f.Months)
		}
	}
}

func TestBuilder_ScopeFiltersByDepartmentCodeOrName(t *testing.T) {
	t.Parallel()

	adm1, err := geoFeatures(adm1JSON)
	if err != nil {
		t.Fatal(err)
	}

	adm2, err := geoFeatures(adm2JSON)
	if err != nil {
		t.Fatal(err)
	}

	municipalities, departments, err := BuildBoundaries(adm1, adm2)
	if err != nil {
		t.Fatal(err)
	}

	ix := geo.NewIndex(municipalities, departments)

	for _, dep := range []string{"HN-LE", "lempira"} {
		got, err := (&Builder{Department: dep}).scope(ix)
		if err != nil || len(got) != 2 {
			t.Errorf("scope(%q) = %d municipalities, %v; want 2", dep, len(got), err)
		}
	}

	if _, err := (&Builder{Department: "Olancho"}).scope(ix); !errors.Is(err, ErrIncompleteBuild) {
		t.Errorf("unknown department err = %v", err)
	}
}

func geoFeatures(s string) (geo.FeatureCollection, error) {
	return geo.ParseFeatureCollection([]byte(s))
}
