package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dma-teams/nasa-app-api/internal/catalog"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/plan"
)

type Planner interface {
	MunicipalPlan(ctx context.Context, code string, opts plan.Options) (*plan.MunicipalPlan, error)
	DepartmentPlan(ctx context.Context, code string, opts plan.Options) (*plan.DepartmentPlan, error)
}

type Directory interface {
	MunicipalityAt(lat, lon float64) (geo.Municipality, error)
	Municipalities() []geo.Municipality
	Departments() []geo.Department
}

type ReadinessCheck func(ctx context.Context) error

type Config struct {
	CORSOrigins           []string
	TrustedProxies        []netip.Prefix
	MunicipalTimeout      time.Duration
	DepartmentTimeout     time.Duration
	MunicipalRate         RateRule
	DepartmentRate        RateRule
	HeavyMaxInFlight      int
	ReadinessTimeout      time.Duration
	PlanCacheMaxAgeSecond int
}

type API struct {
	cfg          Config
	planner      Planner
	directory    func() (Directory, bool)
	catalog      *catalog.Catalog
	ready        ReadinessCheck
	logger       *slog.Logger
	shuttingDown atomic.Bool
	municipal    *ipLimiter
	department   *ipLimiter
	heavy        *loadShedder
}

func New(cfg Config, planner Planner, directory func() (Directory, bool), cat *catalog.Catalog, ready ReadinessCheck, logger *slog.Logger) *API {
	return &API{
		cfg: cfg, planner: planner, directory: directory, catalog: cat, ready: ready, logger: logger,
		municipal:  newIPLimiter(cfg.MunicipalRate),
		department: newIPLimiter(cfg.DepartmentRate),
		heavy:      newLoadShedder(cfg.HeavyMaxInFlight),
	}
}

func (a *API) RunBackground(ctx context.Context) error {
	var wg sync.WaitGroup

	wg.Go(func() { a.municipal.run(ctx) })
	wg.Go(func() { a.department.run(ctx) })
	wg.Wait()

	return nil
}

func (a *API) StartShutdown() {
	a.shuttingDown.Store(true)
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	light := func(h http.HandlerFunc) http.Handler {
		return a.limit(a.municipal, withTimeout(a.cfg.MunicipalTimeout, h))
	}
	heavy := func(h http.HandlerFunc) http.Handler {
		return a.limit(a.department, a.heavy.wrap(withTimeout(a.cfg.DepartmentTimeout, h)))
	}

	mux.HandleFunc("GET /livez", a.livez)
	mux.HandleFunc("GET /readyz", a.readyz)
	mux.Handle("GET /openapi.yaml", light(a.openAPI))
	mux.Handle("GET /docs", light(a.swaggerUI))
	mux.Handle("GET /v1/crops", light(a.crops))
	mux.Handle("GET /v1/municipalities", light(a.municipalities))
	mux.Handle("GET /v1/departments", light(a.departments))
	mux.Handle("GET /v1/plan", light(a.planByLocation))
	mux.Handle("GET /v1/municipalities/{code}/plan", light(a.municipalPlan))
	mux.Handle("GET /v1/departments/{code}/plan", heavy(a.departmentPlan))
	mux.HandleFunc("/", a.notFound)

	return chain(mux, a.recoverPanics, requestID, a.accessLog, a.cors)
}
