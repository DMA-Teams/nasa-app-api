package store

import (
	"context"
	"errors"
	"math"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/dma-teams/nasa-app-api/internal/agro"
)

func sampleSeries(days int) *agro.CellSeries {
	s := newCellSeries(time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC), days)
	for i := range days {
		s.Precipitation[i] = agro.Millimeters(float64(i%7) * 1.5)
		s.TMean[i], s.TMax[i], s.TMin[i] = 22, 30, 15
		s.RelativeHumid[i], s.DewPoint[i], s.Wind[i] = 75, 17, 2
		s.SolarRadiation[i], s.Pressure[i] = 18.5, 89.9
	}

	s.Precipitation[3] = agro.Millimeters(math.NaN())

	return s
}

func TestCellSeriesCodec_RoundTrip(t *testing.T) {
	t.Parallel()

	in := sampleSeries(10958)

	data, err := EncodeCellSeries(in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if want := seriesHeaderSize + seriesParameters*10958*4; len(data) != want {
		t.Errorf("encoded size = %d, want %d", len(data), want)
	}

	out, err := DecodeCellSeries(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !out.Start.Equal(in.Start) || out.Len() != in.Len() {
		t.Fatalf("decoded start %v len %d, want %v %d", out.Start, out.Len(), in.Start, in.Len())
	}

	if !math.IsNaN(float64(out.Precipitation[3])) || out.Precipitation[6] != in.Precipitation[6] || out.SolarRadiation[100] != 18.5 {
		t.Error("decoded values differ from the original")
	}
}

func TestDecodeCellSeries_RejectsCorruptData(t *testing.T) {
	t.Parallel()

	data, err := EncodeCellSeries(sampleSeries(30))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	cases := map[string][]byte{
		"truncated": data[:len(data)-3],
		"bad magic": append([]byte("XXXX"), data[4:]...),
		"empty":     nil,
	}

	for name, payload := range cases {
		if _, err := DecodeCellSeries(payload); !errors.Is(err, ErrCorruptSeries) {
			t.Errorf("%s: err = %v, want ErrCorruptSeries", name, err)
		}
	}
}

type fakeVersions struct {
	v   Versions
	err error
}

func (f *fakeVersions) CurrentVersions(context.Context) (Versions, error) {
	return f.v, f.err
}

func TestWatcher_RefreshCallsOnChangeOnlyWhenVersionsChange(t *testing.T) {
	t.Parallel()

	reader := &fakeVersions{v: Versions{Dataset: "d1", Forecast: "f1"}}
	calls := 0
	w := NewWatcher(reader, time.Minute, time.Second, func(context.Context, Versions) error {
		calls++

		return nil
	}, nil)

	for range 3 {
		if err := w.Refresh(context.Background()); err != nil {
			t.Fatalf("Refresh: %v", err)
		}
	}

	reader.v.Forecast = "f2"
	if err := w.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if v, ok := w.Current(); !ok || v.Forecast != "f2" || calls != 2 {
		t.Errorf("current %+v ok %v calls %d, want f2, true, 2", v, ok, calls)
	}
}

func TestStore_RedisIntegration(t *testing.T) {
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("REDIS_TEST_ADDR not set; integration test skipped")
	}

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("REDIS_TEST_PASSWORD"), ContextTimeoutEnabled: true})
	t.Cleanup(func() { _ = rdb.Close() })

	s := New(rdb)
	dataset := "test-" + time.Now().UTC().Format("20060102150405.000")

	if err := s.PutCellSeries(ctx, dataset, "14.50_-88.750", sampleSeries(400)); err != nil {
		t.Fatalf("PutCellSeries: %v", err)
	}

	got, err := s.CellSeries(ctx, dataset, "14.50_-88.750")
	if err != nil || got.Len() != 400 {
		t.Fatalf("CellSeries = %v, %v", got, err)
	}

	if _, err := s.Profile(ctx, dataset, "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing profile err = %v, want ErrNotFound", err)
	}

	if err := s.PutProfile(ctx, dataset, &agro.SiteProfile{MunicipalityCode: "present"}); err != nil {
		t.Fatalf("PutProfile: %v", err)
	}

	missing, err := s.MissingProfiles(ctx, dataset, []string{"present", "missing"})
	if err != nil || len(missing) != 1 || missing[0] != "missing" {
		t.Errorf("MissingProfiles = %v, %v; want [missing]", missing, err)
	}

	t.Cleanup(func() { rdb.Del(ctx, ClimateKey(dataset, "14.50_-88.750"), ProfileKey(dataset, "present")) })
}
