package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/paulmach/orb"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/store"
)

const (
	midSummerFromDOY = 182
	midSummerToDOY   = 243
	imergWetDayMm    = 1.0
)

var caniculaRatios = []float64{0.3, 0.4, 0.5, 0.6, 0.7}

var caniculaMinimumDays = []int{5, 10, 15}

type municipalitySeries struct {
	name  string
	power *agro.SiteSeries
	imerg *agro.SiteSeries
}

type caniculaYear struct {
	year            int
	imergRainMm     float64
	powerRainMm     float64
	imergDrySpell   float64
	powerDetected   float64
	imergDetected   float64
	powerLengthDays float64
	imergLengthDays float64
}

func loadMunicipalitySeries(testdata, boundaries string, grid *imergGrid) ([]municipalitySeries, error) {
	profiles, err := loadProfiles(filepath.Join(testdata, "profile"))
	if err != nil {
		return nil, err
	}

	shapes, err := loadShapes(boundaries)
	if err != nil {
		return nil, err
	}

	climate := map[string]*agro.CellSeries{}
	out := make([]municipalitySeries, 0, len(profiles))

	for _, p := range profiles {
		power, err := powerSeries(filepath.Join(testdata, "climate"), p, climate)
		if err != nil {
			return nil, err
		}

		mp, ok := shapes[p.MunicipalityCode]
		if !ok {
			return nil, fmt.Errorf("municipality %s without boundary", p.MunicipalityCode)
		}

		pixels := grid.pixelsIn(mp, orb.Point{p.LongitudeDeg, p.LatitudeDeg})
		out = append(out, municipalitySeries{name: p.MunicipalityName, power: power, imerg: withImerg(power, grid, pixels)})
	}

	slices.SortFunc(out, func(a, b municipalitySeries) int { return strings.Compare(a.name, b.name) })

	return out, nil
}

func loadProfiles(dir string) ([]*agro.SiteProfile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read profiles: %w", err)
	}

	out := make([]*agro.SiteProfile, 0, len(entries))

	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read profile: %w", err)
		}

		var p agro.SiteProfile
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("decode profile %s: %w", e.Name(), err)
		}

		out = append(out, &p)
	}

	return out, nil
}

func loadShapes(path string) (map[string]orb.MultiPolygon, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read boundaries: %w", err)
	}

	fc, err := geo.ParseFeatureCollection(data)
	if err != nil {
		return nil, err
	}

	out := make(map[string]orb.MultiPolygon, len(fc.Features))

	for _, f := range fc.Features {
		mp, err := f.Geometry.MultiPolygon()
		if err != nil {
			return nil, err
		}

		out[f.Properties["shapeID"]] = mp
	}

	return out, nil
}

func powerSeries(dir string, p *agro.SiteProfile, cache map[string]*agro.CellSeries) (*agro.SiteSeries, error) {
	cells := make([]agro.WeightedCell, 0, len(p.Cells))

	for _, c := range p.Cells {
		s, ok := cache[c.CellID]
		if !ok {
			var err error
			if s, err = readCell(filepath.Join(dir, c.CellID+".bin.gz")); err != nil {
				return nil, err
			}

			cache[c.CellID] = s
		}

		cells = append(cells, agro.WeightedCell{Series: s, Weight: c.Weight, Elevation: c.Elevation})
	}

	series, err := agro.BuildSiteSeries(cells, agro.SiteSpec{
		LatitudeDeg: p.LatitudeDeg, Elevation: p.ElevationMedian, Excluded: agro.ExcludedSet(p.ExcludedMonths),
	})
	if err != nil {
		return nil, fmt.Errorf("site series %s: %w", p.MunicipalityName, err)
	}

	return series, nil
}

func readCell(path string) (*agro.CellSeries, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read climate: %w", err)
	}

	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("gunzip %s: %w", path, err)
	}

	data, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("gunzip %s: %w", path, err)
	}

	return store.DecodeCellSeries(data)
}

func withImerg(power *agro.SiteSeries, grid *imergGrid, pixels []int) *agro.SiteSeries {
	out := *power
	out.Precipitation = make([]agro.Millimeters, power.Len())
	offset := power.IndexOf(grid.start)

	for i := range out.Precipitation {
		out.Precipitation[i] = agro.Millimeters(grid.mean(i-offset, pixels))
	}

	return &out
}

func caniculaReport(w io.Writer, series []municipalitySeries, from, to int) error {
	years := make([]caniculaYear, 0, to-from+1)
	for y := from; y <= to; y++ {
		years = append(years, summarizeYear(series, y))
	}

	p := &printer{w: w}
	p.line("# Canícula en Lempira: POWER contra IMERG (%d municipios, %d–%d)", len(series), from, to)
	p.line("")
	p.line("Lluvia del 1 de julio al 31 de agosto, promedio de los municipios. Racha seca IMERG: días seguidos con menos de %.0f mm.", imergWetDayMm)
	p.line("Detección con la regla actual (lluvia de 10 días < 0,5 × ET0 de 10 días, al menos 5 días seguidos).")
	p.line("")
	p.line("| Año | Lluvia IMERG (mm) | Lluvia POWER (mm) | Racha seca IMERG (días) | Detectada POWER | Detectada IMERG | Duración POWER | Duración IMERG |")
	p.line("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |")

	for _, y := range years {
		p.line("| %d | %.0f | %.0f | %.1f | %.0f %% | %.0f %% | %.0f | %.0f |", y.year, y.imergRainMm, y.powerRainMm, y.imergDrySpell,
			y.powerDetected*100, y.imergDetected*100, y.powerLengthDays, y.imergLengthDays)
	}

	p.line("")
	p.line("## Correlación de cada regla con la lluvia de julio y agosto según IMERG")
	p.line("")
	p.line("Spearman entre la fracción de municipios con canícula detectada y la lluvia IMERG de julio–agosto. Más negativo es mejor: más detección en los años más secos.")
	p.line("")
	p.line("| Cociente | Días mínimos | Spearman POWER | Spearman IMERG | Años con detección media POWER | Años con detección media IMERG |")
	p.line("| ---: | ---: | ---: | ---: | ---: | ---: |")

	rain := make([]float64, len(years))
	for i, y := range years {
		rain[i] = y.imergRainMm
	}

	for _, ratio := range caniculaRatios {
		for _, minDays := range caniculaMinimumDays {
			rule := agro.CaniculaRule{DeficitRatio: ratio, MinimumDays: minDays}
			pw, im := make([]float64, len(years)), make([]float64, len(years))

			for i, y := range years {
				pw[i], _ = detection(series, y.year, rule, func(m municipalitySeries) *agro.SiteSeries { return m.power })
				im[i], _ = detection(series, y.year, rule, func(m municipalitySeries) *agro.SiteSeries { return m.imerg })
			}

			p.line("| %.1f | %d | %.2f | %.2f | %d | %d |", ratio, minDays, spearman(pw, rain), spearman(im, rain), countAbove(pw, 0.5), countAbove(im, 0.5))
		}
	}

	return p.err
}

func summarizeYear(series []municipalitySeries, year int) caniculaYear {
	out := caniculaYear{year: year}
	n := float64(len(series))

	for _, m := range series {
		out.imergRainMm += rainBetween(m.imerg, year) / n
		out.powerRainMm += rainBetween(m.power, year) / n
		out.imergDrySpell += float64(longestDryRun(m.imerg, year)) / n
	}

	out.powerDetected, out.powerLengthDays = detection(series, year, agro.DefaultCaniculaRule, func(m municipalitySeries) *agro.SiteSeries { return m.power })
	out.imergDetected, out.imergLengthDays = detection(series, year, agro.DefaultCaniculaRule, func(m municipalitySeries) *agro.SiteSeries { return m.imerg })

	return out
}

func detection(series []municipalitySeries, year int, rule agro.CaniculaRule, pick func(municipalitySeries) *agro.SiteSeries) (fraction, meanLength float64) {
	detected := 0

	for _, m := range series {
		if w, ok := rule.Detect(pick(m), year); ok {
			detected++
			meanLength += float64(w.Length())
		}
	}

	if detected > 0 {
		meanLength /= float64(detected)
	}

	return float64(detected) / float64(len(series)), meanLength
}

func dayIndex(s *agro.SiteSeries, year, doy int) int {
	return s.IndexOf(time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)) + doy - 1
}

func rainBetween(s *agro.SiteSeries, year int) float64 {
	total := 0.0

	for i := dayIndex(s, year, midSummerFromDOY); i <= dayIndex(s, year, midSummerToDOY); i++ {
		if i < 0 || i >= s.Len() {
			return math.NaN()
		}

		total += float64(s.Precipitation[i])
	}

	return total
}

func longestDryRun(s *agro.SiteSeries, year int) int {
	longest, run := 0, 0

	for i := dayIndex(s, year, midSummerFromDOY); i <= dayIndex(s, year, midSummerToDOY) && i < s.Len(); i++ {
		if i >= 0 && float64(s.Precipitation[i]) < imergWetDayMm {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}

	return longest
}

func countAbove(values []float64, threshold float64) int {
	n := 0

	for _, v := range values {
		if v > threshold {
			n++
		}
	}

	return n
}

func spearman(a, b []float64) float64 {
	var ca, cb []float64

	for i := range a {
		if !math.IsNaN(a[i]) && !math.IsNaN(b[i]) {
			ca, cb = append(ca, a[i]), append(cb, b[i])
		}
	}

	ra, rb := ranks(ca), ranks(cb)
	ma, mb := meanOf(ra), meanOf(rb)

	var cov, va, vb float64
	for i := range ra {
		cov += (ra[i] - ma) * (rb[i] - mb)
		va += (ra[i] - ma) * (ra[i] - ma)
		vb += (rb[i] - mb) * (rb[i] - mb)
	}

	if va == 0 || vb == 0 {
		return math.NaN()
	}

	return cov / math.Sqrt(va*vb)
}

func ranks(values []float64) []float64 {
	idx := make([]int, len(values))
	for i := range idx {
		idx[i] = i
	}

	sort.SliceStable(idx, func(i, j int) bool { return values[idx[i]] < values[idx[j]] })

	out := make([]float64, len(values))

	for i := 0; i < len(idx); {
		j := i
		for j+1 < len(idx) && values[idx[j+1]] == values[idx[i]] {
			j++
		}

		for k := i; k <= j; k++ {
			out[idx[k]] = float64(i+j)/2 + 1
		}

		i = j + 1
	}

	return out
}

func meanOf(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

type printer struct {
	w   io.Writer
	err error
}

func (p *printer) line(format string, args ...any) {
	if p.err != nil {
		return
	}

	_, p.err = fmt.Fprintf(p.w, format+"\n", args...)
}
