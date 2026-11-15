package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/geo"
)

const (
	KeyDatasetCurrent  = "dataset:current"
	KeyForecastCurrent = "forecast:current"
	forecastTTL        = 120 * 24 * time.Hour
)

var ErrNotFound = errors.New("store: key not found")

type Versions struct {
	Dataset  string `json:"dataset"`
	Forecast string `json:"forecast"`
}

type CellMeta struct {
	Cell      geo.Cell         `json:"cell"`
	Elevation agro.Meters      `json:"elevationM"`
	Excluded  []agro.YearMonth `json:"excludedMonths"`
	Source    string           `json:"source"`
}

type Store struct {
	rdb redis.UniversalClient
}

func New(rdb redis.UniversalClient) *Store {
	return &Store{rdb: rdb}
}

func climateKey(dataset, cell string) string  { return dataset + ":climate:" + cell }
func cellMetaKey(dataset, cell string) string { return dataset + ":cellmeta:" + cell }
func profileKey(dataset, code string) string  { return dataset + ":profile:" + code }
func boundariesKey(dataset string) string     { return dataset + ":boundaries" }
func forecastKey(version, cell string) string { return version + ":forecast:" + cell }
func resultKey(hash string) string            { return "result:" + hash }
func ProfileKey(dataset, code string) string  { return profileKey(dataset, code) }
func ClimateKey(dataset, cell string) string  { return climateKey(dataset, cell) }
func ForecastKey(version, cell string) string { return forecastKey(version, cell) }
func BoundariesKey(dataset string) string     { return boundariesKey(dataset) }
func CellMetaKey(dataset, cell string) string { return cellMetaKey(dataset, cell) }

func (s *Store) Ping(ctx context.Context) error {
	if err := s.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}

	return nil
}

func (s *Store) CurrentVersions(ctx context.Context) (Versions, error) {
	values, err := s.rdb.MGet(ctx, KeyDatasetCurrent, KeyForecastCurrent).Result()
	if err != nil {
		return Versions{}, fmt.Errorf("read versions: %w", err)
	}

	v := Versions{}
	if d, ok := values[0].(string); ok {
		v.Dataset = d
	}

	if f, ok := values[1].(string); ok {
		v.Forecast = f
	}

	if v.Dataset == "" {
		return v, fmt.Errorf("dataset pointer: %w", ErrNotFound)
	}

	return v, nil
}

func (s *Store) SetCurrentDataset(ctx context.Context, version string) error {
	return s.set(ctx, KeyDatasetCurrent, version, 0)
}

func (s *Store) SetCurrentForecast(ctx context.Context, version string) error {
	return s.set(ctx, KeyForecastCurrent, version, 0)
}

func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	n, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("exists %s: %w", key, err)
	}

	return n == 1, nil
}

func (s *Store) PutCellSeries(ctx context.Context, dataset, cell string, series *agro.CellSeries) error {
	data, err := EncodeCellSeries(series)
	if err != nil {
		return err
	}

	return s.set(ctx, climateKey(dataset, cell), data, 0)
}

func (s *Store) CellSeries(ctx context.Context, dataset, cell string) (*agro.CellSeries, error) {
	data, err := s.get(ctx, climateKey(dataset, cell))
	if err != nil {
		return nil, err
	}

	return DecodeCellSeries(data)
}

func (s *Store) PutCellMeta(ctx context.Context, dataset string, meta CellMeta) error {
	return s.putJSON(ctx, cellMetaKey(dataset, meta.Cell.ID()), meta, 0)
}

func (s *Store) CellMeta(ctx context.Context, dataset, cell string) (CellMeta, error) {
	var meta CellMeta

	return meta, s.getJSON(ctx, cellMetaKey(dataset, cell), &meta)
}

func (s *Store) PutBoundaries(ctx context.Context, dataset string, data []byte) error {
	return s.set(ctx, boundariesKey(dataset), data, 0)
}

func (s *Store) Boundaries(ctx context.Context, dataset string) ([]byte, error) {
	return s.get(ctx, boundariesKey(dataset))
}

func (s *Store) PutProfile(ctx context.Context, dataset string, p *agro.SiteProfile) error {
	return s.putJSON(ctx, profileKey(dataset, p.MunicipalityCode), p, 0)
}

func (s *Store) Profile(ctx context.Context, dataset, code string) (*agro.SiteProfile, error) {
	var p agro.SiteProfile
	if err := s.getJSON(ctx, profileKey(dataset, code), &p); err != nil {
		return nil, err
	}

	return &p, nil
}

func (s *Store) PutForecast(ctx context.Context, version, cell string, f agro.CellForecast) error {
	return s.putJSON(ctx, forecastKey(version, cell), f, forecastTTL)
}

func (s *Store) Forecast(ctx context.Context, version, cell string) (agro.CellForecast, error) {
	var f agro.CellForecast

	return f, s.getJSON(ctx, forecastKey(version, cell), &f)
}

func (s *Store) Result(ctx context.Context, hash string) ([]byte, bool, error) {
	data, err := s.get(ctx, resultKey(hash))
	if errors.Is(err, ErrNotFound) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	return data, true, nil
}

func (s *Store) PutResult(ctx context.Context, hash string, data []byte, ttl time.Duration) error {
	return s.set(ctx, resultKey(hash), data, ttl)
}

func (s *Store) set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := s.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}

	return nil
}

func (s *Store) get(ctx context.Context, key string) ([]byte, error) {
	data, err := s.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("%s: %w", key, ErrNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("get %s: %w", key, err)
	}

	return data, nil
}

func (s *Store) putJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode %s: %w", key, err)
	}

	return s.set(ctx, key, data, ttl)
}

func (s *Store) getJSON(ctx context.Context, key string, v any) error {
	data, err := s.get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("decode %s: %w", key, err)
	}

	return nil
}
