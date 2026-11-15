package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/plan"
)

const problemContentType = "application/problem+json"

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Problem struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

func problemType(slug string) string {
	return "urn:nasa-app-api:problem:" + slug
}

func writeProblem(w http.ResponseWriter, r *http.Request, p Problem, retryAfterSeconds int) {
	if p.Instance == "" {
		p.Instance = RequestIDFrom(r.Context())
	}

	if retryAfterSeconds > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	}

	w.Header().Set("Content-Type", problemContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func writeJSON(w http.ResponseWriter, status int, v any, cacheSeconds int) {
	w.Header().Set("Content-Type", "application/json")

	if cacheSeconds > 0 {
		w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(cacheSeconds))
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}

	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func badRequest(fieldErrors []FieldError) Problem {
	return Problem{Type: problemType("invalid-parameters"), Title: "Parámetros inválidos", Status: http.StatusBadRequest, Errors: fieldErrors}
}

func (a *API) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var input *plan.InputError

	switch {
	case errors.As(err, &input):
		writeProblem(w, r, badRequest([]FieldError{{Field: input.Field, Message: input.Message}}), 0)
	case errors.Is(err, plan.ErrMunicipalityNotFound), errors.Is(err, plan.ErrDepartmentNotFound), errors.Is(err, geo.ErrNotFound):
		writeProblem(w, r, Problem{Type: problemType("not-found"), Title: "Unidad administrativa no encontrada", Status: http.StatusNotFound}, 0)
	case errors.Is(err, geo.ErrOutsideHonduras):
		writeProblem(w, r, Problem{Type: problemType("outside-honduras"), Title: "La ubicación está fuera de Honduras", Status: http.StatusUnprocessableEntity}, 0)
	case errors.Is(err, plan.ErrNotReady):
		writeProblem(w, r, Problem{Type: problemType("not-ready"), Title: "Los datos todavía no están cargados", Status: http.StatusServiceUnavailable}, 30)
	case errors.Is(err, plan.ErrDataUnavailable), errors.Is(err, agro.ErrInsufficientHistory):
		a.logger.ErrorContext(r.Context(), "data unavailable", slog.String("error", err.Error()))
		writeProblem(w, r, Problem{Type: problemType("data-unavailable"), Title: "Datos no disponibles temporalmente", Status: http.StatusServiceUnavailable}, 30)
	case errors.Is(err, context.DeadlineExceeded):
		writeProblem(w, r, Problem{Type: problemType("timeout"), Title: "El cálculo excedió el tiempo permitido", Status: http.StatusServiceUnavailable}, 5)
	case errors.Is(err, context.Canceled):
		return
	default:
		a.logger.ErrorContext(r.Context(), "unexpected error", slog.String("error", err.Error()))
		writeProblem(w, r, Problem{Type: problemType("internal"), Title: "Error interno", Status: http.StatusInternalServerError}, 0)
	}
}
