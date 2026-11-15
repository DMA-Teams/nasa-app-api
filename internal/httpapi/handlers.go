package httpapi

import (
	"context"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/plan"
)

const (
	maxRotationCrops = 4
	listCacheSeconds = 3600
)

func (a *API) livez(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}, 0)
}

func (a *API) readyz(w http.ResponseWriter, r *http.Request) {
	if a.shuttingDown.Load() {
		writeProblem(w, r, Problem{Type: problemType("shutting-down"), Title: "El servidor se está apagando", Status: http.StatusServiceUnavailable}, 5)

		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), a.cfg.ReadinessTimeout)
	defer cancel()

	if err := a.ready(ctx); err != nil {
		writeProblem(w, r, Problem{Type: problemType("not-ready"), Title: "No listo", Status: http.StatusServiceUnavailable, Detail: err.Error()}, 5)

		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"}, 0)
}

func (a *API) notFound(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, Problem{Type: problemType("route-not-found"), Title: "Ruta no encontrada", Status: http.StatusNotFound}, 0)
}

func (a *API) crops(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version": a.catalog.Version, "hash": a.catalog.Hash,
		"crops": a.catalog.Document.Crops, "templates": a.catalog.Document.Templates,
	}, listCacheSeconds)
}

type municipalityItem struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	DepartmentCode string  `json:"departmentCode"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
}

const coordinateDecimals = 1e4

func roundCoordinate(v float64) float64 {
	return math.Round(v*coordinateDecimals) / coordinateDecimals
}

func (a *API) municipalities(w http.ResponseWriter, r *http.Request) {
	dir, ok := a.directory()
	if !ok {
		a.writeError(w, r, plan.ErrNotReady)

		return
	}

	department := r.URL.Query().Get("department")
	items := []municipalityItem{}

	for _, m := range dir.Municipalities() {
		if department == "" || m.DepartmentCode == department {
			p := geo.InteriorPoint(m.Geometry)
			items = append(items, municipalityItem{
				Code: m.Code, Name: m.Name, DepartmentCode: m.DepartmentCode,
				Latitude: roundCoordinate(p.Lat()), Longitude: roundCoordinate(p.Lon()),
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"municipalities": items}, listCacheSeconds)
}

func (a *API) departments(w http.ResponseWriter, r *http.Request) {
	dir, ok := a.directory()
	if !ok {
		a.writeError(w, r, plan.ErrNotReady)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"departments": dir.Departments()}, listCacheSeconds)
}

func (a *API) planByLocation(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lat, latErr := strconv.ParseFloat(q.Get("lat"), 64)
	lon, lonErr := strconv.ParseFloat(q.Get("lon"), 64)

	var errs []FieldError
	if latErr != nil || lat < -90 || lat > 90 {
		errs = append(errs, FieldError{Field: "lat", Message: "latitud decimal entre -90 y 90"})
	}

	if lonErr != nil || lon < -180 || lon > 180 {
		errs = append(errs, FieldError{Field: "lon", Message: "longitud decimal entre -180 y 180"})
	}

	opts, optErrs := a.parseOptions(r)
	if errs = append(errs, optErrs...); len(errs) > 0 {
		writeProblem(w, r, badRequest(errs), 0)

		return
	}

	dir, ok := a.directory()
	if !ok {
		a.writeError(w, r, plan.ErrNotReady)

		return
	}

	m, err := dir.MunicipalityAt(lat, lon)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	a.respondMunicipal(w, r, m.Code, opts)
}

func (a *API) municipalPlan(w http.ResponseWriter, r *http.Request) {
	opts, errs := a.parseOptions(r)
	if len(errs) > 0 {
		writeProblem(w, r, badRequest(errs), 0)

		return
	}

	a.respondMunicipal(w, r, r.PathValue("code"), opts)
}

func (a *API) respondMunicipal(w http.ResponseWriter, r *http.Request, code string, opts plan.Options) {
	p, err := a.planner.MunicipalPlan(r.Context(), code, opts)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, p, a.cfg.PlanCacheMaxAgeSecond)
}

func (a *API) departmentPlan(w http.ResponseWriter, r *http.Request) {
	opts, errs := a.parseOptions(r)
	if len(errs) > 0 {
		writeProblem(w, r, badRequest(errs), 0)

		return
	}

	p, err := a.planner.DepartmentPlan(r.Context(), r.PathValue("code"), opts)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, p, a.cfg.PlanCacheMaxAgeSecond)
}

func (a *API) parseOptions(r *http.Request) (plan.Options, []FieldError) {
	q := r.URL.Query()

	var (
		opts plan.Options
		errs []FieldError
	)

	if v := q.Get("irrigation"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, FieldError{Field: "irrigation", Message: "true o false"})
		}

		opts.Irrigated = b
	}

	priority, ok := agro.ParsePriority(q.Get("priority"))
	if !ok {
		errs = append(errs, FieldError{Field: "priority", Message: "balanced, water o soil"})
	}

	opts.Priority = priority

	if v := q.Get("currentRotation"); v != "" {
		crops := strings.Split(v, ",")
		if len(crops) > maxRotationCrops || slices.Contains(crops, "") {
			errs = append(errs, FieldError{Field: "currentRotation", Message: "entre 1 y 4 cultivos separados por coma"})
		}

		opts.CurrentRotation = crops
	}

	opts.CurrentCrop = q.Get("currentCrop")

	if v := q.Get("fieldFreeFrom"); v != "" {
		t, err := time.Parse(time.DateOnly, v)
		if err != nil {
			errs = append(errs, FieldError{Field: "fieldFreeFrom", Message: "fecha AAAA-MM-DD"})
		}

		opts.FieldFreeFrom = t
	}

	return opts, errs
}
