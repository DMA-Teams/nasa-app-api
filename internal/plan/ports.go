package plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/geo"
)

var (
	ErrNotReady             = errors.New("dataset not loaded")
	ErrMunicipalityNotFound = errors.New("municipality not found")
	ErrDepartmentNotFound   = errors.New("department not found")
	ErrInvalidInput         = errors.New("invalid input")
	ErrDataUnavailable      = errors.New("data unavailable")
)

type InputError struct {
	Field   string
	Message string
}

func (e *InputError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (e *InputError) Unwrap() error {
	return ErrInvalidInput
}

type ProfileReader interface {
	Profile(ctx context.Context, dataset, code string) (*agro.SiteProfile, error)
}

type ClimateReader interface {
	CellSeries(ctx context.Context, dataset, cell string) (*agro.CellSeries, error)
}

type ForecastReader interface {
	Forecast(ctx context.Context, version, cell string) (agro.CellForecast, error)
}

type ResultCache interface {
	Result(ctx context.Context, hash string) ([]byte, bool, error)
	PutResult(ctx context.Context, hash string, data []byte, ttl time.Duration) error
}

type Versions struct {
	Dataset  string `json:"dataset"`
	Forecast string `json:"forecast"`
	Catalog  string `json:"catalog"`
}

type VersionSource interface {
	Versions() (Versions, bool)
}

type Gazetteer interface {
	Municipality(code string) (geo.Municipality, error)
	Department(code string) (geo.Department, error)
	MunicipalityCodesOf(departmentCode string) ([]string, error)
}
