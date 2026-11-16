package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/paulmach/orb"
	"go.uber.org/goleak"

	"github.com/dma-teams/nasa-app-api/internal/catalog"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/plan"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type fakePlanner struct {
	mu       sync.Mutex
	lastOpts plan.Options
	err      error
	block    chan struct{}
	entered  chan struct{}
	panicNow bool
}

func (f *fakePlanner) MunicipalPlan(ctx context.Context, code string, opts plan.Options) (*plan.MunicipalPlan, error) {
	if f.panicNow {
		panic("boom")
	}

	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	f.mu.Lock()
	f.lastOpts = opts
	f.mu.Unlock()

	if f.err != nil {
		return nil, f.err
	}

	return &plan.MunicipalPlan{Municipality: plan.Municipality{Code: code}, Disclaimer: plan.Disclaimer}, nil
}

func (f *fakePlanner) DepartmentPlan(ctx context.Context, code string, _ plan.Options) (*plan.DepartmentPlan, error) {
	if f.entered != nil {
		f.entered <- struct{}{}
	}

	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &plan.DepartmentPlan{Department: plan.Municipality{Code: code}}, nil
}

func testDirectory() *geo.Index {
	return geo.NewIndex(
		[]geo.Municipality{{Code: "M1", Name: "Uno", DepartmentCode: "D1", Geometry: orb.MultiPolygon{{{{-88.7, 14.2}, {-88.5, 14.2}, {-88.5, 14.4}, {-88.7, 14.4}, {-88.7, 14.2}}}}}},
		[]geo.Department{{Code: "D1", Name: "Lempira"}},
	)
}

func newTestAPI(t *testing.T, planner *fakePlanner, cfgEdit func(*Config)) (*API, http.Handler) {
	t.Helper()

	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		MunicipalTimeout: time.Second, DepartmentTimeout: time.Second, ReadinessTimeout: time.Second,
		MunicipalRate: RateRule{PerSecond: 100, Burst: 100}, DepartmentRate: RateRule{PerSecond: 100, Burst: 100},
		HeavyMaxInFlight: 1, CORSOrigins: []string{"https://app.example"},
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
	}
	if cfgEdit != nil {
		cfgEdit(&cfg)
	}

	dir := testDirectory()
	api := New(cfg, planner, func() (Directory, bool) { return dir, true }, cat, func(context.Context) error { return nil }, slog.New(slog.DiscardHandler))

	return api, api.Handler()
}

func do(h http.Handler, target string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	req.RemoteAddr = "203.0.113.7:4000"

	for k, v := range header {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) Problem {
	t.Helper()

	if ct := rec.Header().Get("Content-Type"); ct != problemContentType {
		t.Fatalf("content type = %q, want %q", ct, problemContentType)
	}

	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem: %v", err)
	}

	return p
}

func TestPlanByLocation_ResolvesMunicipalityAndParsesOptions(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{}
	_, h := newTestAPI(t, planner, nil)

	rec := do(h, "/v1/plan?lat=14.3&lon=-88.6&irrigation=true&priority=soil&currentRotation=maize,beans", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}

	var p plan.MunicipalPlan
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || p.Municipality.Code != "M1" {
		t.Fatalf("plan = %+v, %v", p, err)
	}

	planner.mu.Lock()
	defer planner.mu.Unlock()

	if !planner.lastOpts.Irrigated || len(planner.lastOpts.CurrentRotation) != 2 {
		t.Errorf("options = %+v", planner.lastOpts)
	}

	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("missing X-Request-Id")
	}
}

func TestPlanByLocation_ValidationAndGeography(t *testing.T) {
	t.Parallel()

	_, h := newTestAPI(t, &fakePlanner{}, nil)

	rec := do(h, "/v1/plan?lat=abc&lon=-88.6&priority=fast", nil)
	if p := decodeProblem(t, rec); rec.Code != http.StatusBadRequest || len(p.Errors) != 2 {
		t.Errorf("status %d errors %v, want 400 with lat and priority errors", rec.Code, p.Errors)
	}

	rec = do(h, "/v1/plan?lat=40.4&lon=-3.7", nil)
	if p := decodeProblem(t, rec); rec.Code != http.StatusUnprocessableEntity || !strings.HasSuffix(p.Type, "outside-honduras") {
		t.Errorf("status %d type %q, want 422 outside-honduras", rec.Code, p.Type)
	}
}

func TestErrorsMapToProblemDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err        error
		status     int
		retryAfter bool
	}{
		{plan.ErrMunicipalityNotFound, http.StatusNotFound, false},
		{plan.ErrNotReady, http.StatusServiceUnavailable, true},
		{errors.Join(plan.ErrDataUnavailable, errors.New("redis down")), http.StatusServiceUnavailable, true},
		{&plan.InputError{Field: "currentCrop", Message: "unknown"}, http.StatusBadRequest, false},
		{errors.New("surprise"), http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		_, h := newTestAPI(t, &fakePlanner{err: tt.err}, nil)
		rec := do(h, "/v1/municipalities/M1/plan", nil)
		p := decodeProblem(t, rec)

		if rec.Code != tt.status || p.Status != tt.status {
			t.Errorf("%v: status %d/%d, want %d", tt.err, rec.Code, p.Status, tt.status)
		}

		if got := rec.Header().Get("Retry-After") != ""; got != tt.retryAfter {
			t.Errorf("%v: Retry-After present = %v, want %v", tt.err, got, tt.retryAfter)
		}

		if strings.Contains(rec.Body.String(), "redis down") || strings.Contains(rec.Body.String(), "surprise") {
			t.Errorf("%v: internal detail leaked to the client", tt.err)
		}
	}
}

func TestRateLimit_Returns429WithRetryAfter(t *testing.T) {
	t.Parallel()

	_, h := newTestAPI(t, &fakePlanner{}, func(c *Config) { c.MunicipalRate = RateRule{PerSecond: 0.5, Burst: 2} })

	for i := range 2 {
		if rec := do(h, "/v1/municipalities/M1/plan", nil); rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d", i, rec.Code)
		}
	}

	rec := do(h, "/v1/municipalities/M1/plan", nil)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "2" {
		t.Errorf("status %d Retry-After %q, want 429 and 2", rec.Code, rec.Header().Get("Retry-After"))
	}

	other := do(h, "/v1/municipalities/M1/plan", map[string]string{"X-Forwarded-For": "198.51.100.1"})
	if other.Code != http.StatusTooManyRequests {
		t.Errorf("untrusted proxy header bypassed the limit: status %d", other.Code)
	}
}

func TestClientIP_TrustsOnlyConfiguredProxies(t *testing.T) {
	t.Parallel()

	api, _ := newTestAPI(t, &fakePlanner{}, nil)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.1.2.3:5000"
	req.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.5")

	if ip := api.clientIP(req); ip != "198.51.100.9" {
		t.Errorf("client IP behind trusted proxy = %s, want 198.51.100.9", ip)
	}
}

func TestLoadShedding_RejectsWhenHeavySlotsAreFull(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{block: make(chan struct{}), entered: make(chan struct{})}
	_, h := newTestAPI(t, planner, nil)

	first := make(chan int)

	go func() { first <- do(h, "/v1/departments/D1/plan", nil).Code }()

	<-planner.entered

	rec := do(h, "/v1/departments/D1/plan", nil)

	close(planner.block)

	if code := <-first; code != http.StatusOK {
		t.Errorf("first request status = %d, want 200", code)
	}

	if p := decodeProblem(t, rec); rec.Code != http.StatusServiceUnavailable || !strings.HasSuffix(p.Type, "overloaded") || rec.Header().Get("Retry-After") == "" {
		t.Errorf("second request status = %d type %q, want 503 overloaded with Retry-After", rec.Code, p.Type)
	}
}

func TestRequestTimeout_Returns503(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{block: make(chan struct{})}
	defer close(planner.block)

	_, h := newTestAPI(t, planner, func(c *Config) { c.MunicipalTimeout = 20 * time.Millisecond })

	rec := do(h, "/v1/municipalities/M1/plan", nil)
	if p := decodeProblem(t, rec); rec.Code != http.StatusServiceUnavailable || !strings.HasSuffix(p.Type, "timeout") {
		t.Errorf("status %d type %q, want 503 timeout", rec.Code, p.Type)
	}
}

func TestRecover_TurnsPanicsInto500(t *testing.T) {
	t.Parallel()

	_, h := newTestAPI(t, &fakePlanner{panicNow: true}, nil)

	rec := do(h, "/v1/municipalities/M1/plan", nil)
	if p := decodeProblem(t, rec); rec.Code != http.StatusInternalServerError || p.Status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 problem", rec.Code)
	}
}

func TestHealthAndShutdown(t *testing.T) {
	t.Parallel()

	api, h := newTestAPI(t, &fakePlanner{}, nil)

	if rec := do(h, "/readyz", nil); rec.Code != http.StatusOK {
		t.Fatalf("readyz = %d", rec.Code)
	}

	api.StartShutdown()

	if rec := do(h, "/readyz", nil); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("readyz while shutting down = %d, want 503", rec.Code)
	}

	if rec := do(h, "/livez", nil); rec.Code != http.StatusOK {
		t.Errorf("livez = %d, want 200", rec.Code)
	}
}

func TestReadyz_DoesNotExposeDependencyError(t *testing.T) {
	t.Parallel()

	api, _ := newTestAPI(t, &fakePlanner{}, nil)
	api.ready = func(context.Context) error {
		return errors.New("dial tcp 172.18.0.2:6379: connect: connection refused")
	}

	rec := do(api.Handler(), "/readyz", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz = %d, want 503", rec.Code)
	}

	if body := rec.Body.String(); strings.Contains(body, "172.18.0.2") || strings.Contains(body, "6379") {
		t.Errorf("readyz body exposes the dependency error: %s", body)
	}
}

func TestCORSAndCatalog(t *testing.T) {
	t.Parallel()

	_, h := newTestAPI(t, &fakePlanner{}, nil)

	rec := do(h, "/v1/crops", map[string]string{"Origin": "https://app.example"})
	if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Errorf("status %d CORS %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}

	rec = do(h, "/v1/crops", map[string]string{"Origin": "https://evil.example"})
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("unlisted origin received CORS headers")
	}

	if rec := do(h, "/nope", nil); rec.Code != http.StatusNotFound {
		t.Errorf("unknown route = %d, want 404", rec.Code)
	}
}

func TestRunBackground_StopsWithContext(t *testing.T) {
	t.Parallel()

	api, _ := newTestAPI(t, &fakePlanner{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		_ = api.RunBackground(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunBackground did not stop after cancel")
	}
}

func TestOpenAPI_ServedSpecMatchesSourceFile(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if string(source) != openAPISpec {
		t.Fatal("internal/httpapi/openapi_data.go is stale: run node tools/genopenapi/gen.js > internal/httpapi/openapi_data.go")
	}

	_, h := newTestAPI(t, &fakePlanner{}, nil)

	rec := do(h, "/openapi.yaml", nil)
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "openapi: 3.1.0") {
		t.Errorf("openapi.yaml status %d", rec.Code)
	}

	rec = do(h, "/docs", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "swagger-ui-dist@"+swaggerUIVersion) {
		t.Errorf("docs status %d", rec.Code)
	}

	for _, sri := range []string{swaggerCSSSRI, swaggerBundleSRI} {
		if !strings.Contains(rec.Body.String(), `integrity="`+sri+`" crossorigin="anonymous"`) {
			t.Errorf("docs page is missing subresource integrity %s", sri)
		}
	}

	for _, route := range []string{"/v1/plan", "/v1/municipalities/{code}/plan", "/v1/departments/{code}/plan", "/v1/municipalities", "/v1/departments", "/v1/crops", "/livez", "/readyz"} {
		if !strings.Contains(openAPISpec, "  "+route+":") {
			t.Errorf("route %s missing from the OpenAPI spec", route)
		}
	}
}

func TestMunicipalities_IncludeAPointInsideEachMunicipality(t *testing.T) {
	t.Parallel()

	_, h := newTestAPI(t, &fakePlanner{}, nil)

	rec := do(h, "/v1/municipalities?department=D1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var body struct {
		Municipalities []municipalityItem `json:"municipalities"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if len(body.Municipalities) != 1 {
		t.Fatalf("municipalities = %+v, want 1", body.Municipalities)
	}

	m := body.Municipalities[0]
	if m.Latitude != 14.3 || m.Longitude != -88.6 {
		t.Errorf("point = (%v, %v), want the square's center (14.3, -88.6)", m.Latitude, m.Longitude)
	}
}
