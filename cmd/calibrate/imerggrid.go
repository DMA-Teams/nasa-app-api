package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
)

var errBadImergFile = errors.New("unexpected IMERG CSV layout")

type imergGrid struct {
	start time.Time
	lons  []float64
	lats  []float64
	days  [][]float64
}

func parseImergDay(data []byte) (lons, lats []float64, values []float64, err error) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)

	for sc.Scan() {
		name, rest, ok := strings.Cut(sc.Text(), ",")
		if !ok {
			continue
		}

		nums, err := parseFloats(rest)
		if err != nil {
			return nil, nil, nil, err
		}

		switch {
		case name == "/lon":
			lons = nums
		case name == "/lat":
			lats = nums
		case strings.HasPrefix(name, "/precipitation[0]["):
			values = append(values, nums...)
		}
	}

	if err := sc.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("scan IMERG CSV: %w", err)
	}

	if len(lons) == 0 || len(lats) == 0 || len(values) != len(lons)*len(lats) {
		return nil, nil, nil, fmt.Errorf("%w: %d lon, %d lat, %d values", errBadImergFile, len(lons), len(lats), len(values))
	}

	return lons, lats, values, nil
}

func parseFloats(s string) ([]float64, error) {
	fields := strings.Split(s, ",")
	out := make([]float64, 0, len(fields))

	for _, f := range fields {
		v, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %q", errBadImergFile, f)
		}

		out = append(out, v)
	}

	return out, nil
}

func loadImergGrid(dir string, start, end time.Time) (*imergGrid, error) {
	g := &imergGrid{start: start}

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		path := filepath.Join(dir, d.Format("2006"), d.Format("20060102")+".csv")

		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			g.days = append(g.days, nil)

			continue
		}

		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		lons, lats, values, err := parseImergDay(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		if g.lons == nil {
			g.lons, g.lats = lons, lats
		}

		g.days = append(g.days, values)
	}

	return g, nil
}

func (g *imergGrid) pixelsIn(mp orb.MultiPolygon, centroid orb.Point) []int {
	var inside []int

	nearest, best := 0, math.Inf(1)

	for i, lon := range g.lons {
		for j, lat := range g.lats {
			p := orb.Point{lon, lat}
			if planar.MultiPolygonContains(mp, p) {
				inside = append(inside, i*len(g.lats)+j)
			}

			if d := planar.DistanceSquared(p, centroid); d < best {
				nearest, best = i*len(g.lats)+j, d
			}
		}
	}

	if len(inside) == 0 {
		return []int{nearest}
	}

	return inside
}

func (g *imergGrid) mean(day int, pixels []int) float64 {
	if day < 0 || day >= len(g.days) || g.days[day] == nil {
		return math.NaN()
	}

	sum := 0.0

	for _, p := range pixels {
		v := g.days[day][p]
		if v < 0 {
			return math.NaN()
		}

		sum += v
	}

	return sum / float64(len(pixels))
}
