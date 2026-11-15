package main

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulmach/orb"

	"github.com/dma-teams/nasa-app-api/internal/agro"
)

const sampleImergDay = `Dataset: 3B-DAY.MS.MRG.3IMERG.20150701-S000000-E235959.V07B.nc4
/lon, -88.95, -88.85
/lat, 13.85, 13.95, 14.05
/precipitation[0][0], 1, 2, 3
/precipitation[0][1], 4, 5, 6
`

func TestParseImergDay_ReadsLonMajorGrid(t *testing.T) {
	lons, lats, values, err := parseImergDay([]byte(sampleImergDay))
	if err != nil {
		t.Fatal(err)
	}

	if len(lons) != 2 || len(lats) != 3 {
		t.Fatalf("grid %dx%d, want 2x3", len(lons), len(lats))
	}

	if values[1*len(lats)+2] != 6 {
		t.Errorf("value at lon 1, lat 2 = %v, want 6", values[1*len(lats)+2])
	}
}

func TestParseImergDay_RejectsIncompleteGrid(t *testing.T) {
	if _, _, _, err := parseImergDay([]byte("/lon, 1, 2\n/lat, 3\n/precipitation[0][0], 1\n")); err == nil {
		t.Fatal("want error for missing precipitation row")
	}
}

func TestImergGrid_PixelsAndMeanWithGaps(t *testing.T) {
	dir := t.TempDir()
	day := time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC)

	if err := writeAtomic(filepath.Join(dir, "2015", "20150701.csv"), []byte(sampleImergDay)); err != nil {
		t.Fatal(err)
	}

	g, err := loadImergGrid(dir, day, day.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}

	square := orb.MultiPolygon{{{{-89, 13.9}, {-88.8, 13.9}, {-88.8, 14.1}, {-89, 14.1}, {-89, 13.9}}}}

	pixels := g.pixelsIn(square, orb.Point{-88.9, 14})
	if len(pixels) != 4 {
		t.Fatalf("pixels inside = %v, want 4", pixels)
	}

	if got := g.mean(0, pixels); got != (2+3+5+6)/4.0 {
		t.Errorf("mean = %v, want 4", got)
	}

	if got := g.mean(1, pixels); !math.IsNaN(got) {
		t.Errorf("missing day mean = %v, want NaN", got)
	}

	far := orb.MultiPolygon{{{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0, 0}}}}
	if got := g.pixelsIn(far, orb.Point{-88.95, 13.85}); len(got) != 1 || got[0] != 0 {
		t.Errorf("fallback pixel = %v, want nearest [0]", got)
	}
}

func TestWithImerg_AlignsDatesAndLeavesPowerUntouched(t *testing.T) {
	start := time.Date(2015, time.June, 30, 0, 0, 0, 0, time.UTC)
	power := &agro.SiteSeries{Start: start, Precipitation: []agro.Millimeters{9, 9, 9}, ET0: []agro.Millimeters{4, 4, 4}}
	g := &imergGrid{start: start.AddDate(0, 0, 1), days: [][]float64{{2}, {-9999.9}}}

	got := withImerg(power, g, []int{0})

	if !math.IsNaN(float64(got.Precipitation[0])) || got.Precipitation[1] != 2 || !math.IsNaN(float64(got.Precipitation[2])) {
		t.Errorf("imerg precipitation = %v, want [NaN 2 NaN]", got.Precipitation)
	}

	if power.Precipitation[1] != 9 {
		t.Error("power series modified")
	}
}

func TestSpearman(t *testing.T) {
	tests := []struct {
		name string
		a, b []float64
		want float64
	}{
		{"perfect", []float64{1, 2, 3, 4}, []float64{10, 20, 30, 40}, 1},
		{"inverse", []float64{1, 2, 3, 4}, []float64{40, 30, 20, 10}, -1},
		{"ties", []float64{1, 1, 2, 2}, []float64{1, 1, 2, 2}, 1},
		{"skips missing pairs", []float64{1, 2, 3, math.NaN()}, []float64{10, 20, 30, 0}, 1},
	}

	for _, tt := range tests {
		if got := spearman(tt.a, tt.b); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("%s: spearman = %v, want %v", tt.name, got, tt.want)
		}
	}

	if got := spearman([]float64{1, 1}, []float64{1, 2}); !math.IsNaN(got) {
		t.Errorf("constant input = %v, want NaN", got)
	}
}
