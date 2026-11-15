package etl

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"sync"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/geo"
	"github.com/dma-teams/nasa-app-api/internal/sources"
)

type SoilProvider interface {
	Summary(ctx context.Context, m geo.Municipality) (agro.SoilSummary, error)
}

type RESTSoil struct {
	Client *sources.SoilGridsClient
}

func (r RESTSoil) Summary(ctx context.Context, m geo.Municipality) (agro.SoilSummary, error) {
	p := geo.InteriorPoint(m.Geometry)

	point, err := r.Client.Point(ctx, p.Lat(), p.Lon())
	if err != nil {
		return agro.SoilSummary{}, fmt.Errorf("soil for %s: %w", m.Code, err)
	}

	return point.Summary(), nil
}

type RasterSoil struct {
	Dir string

	mu      sync.Mutex
	rasters map[string]*sources.Raster
}

const soilScale = 10.0

var soilScales = map[string]float64{
	"clay": soilScale, "sand": soilScale, "silt": soilScale, "soc": soilScale, "phh2o": soilScale,
	"nitrogen": 100, "cec": soilScale, "bdod": 100,
}

var topsoilDepthWeights = []struct {
	depth  string
	weight float64
}{{"0-5cm", 5}, {"5-15cm", 10}, {"15-30cm", 15}}

func (r *RasterSoil) raster(property, depth string) (*sources.Raster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.rasters == nil {
		r.rasters = map[string]*sources.Raster{}
	}

	key := property + "_" + depth
	if ras, ok := r.rasters[key]; ok {
		return ras, nil
	}

	ras, err := sources.ReadENVI(filepath.Join(r.Dir, key+"_mean.bin"))
	if err != nil {
		return nil, err
	}

	r.rasters[key] = ras

	return ras, nil
}

func (r *RasterSoil) topsoil(property string, m geo.Municipality) ([]float64, error) {
	var combined []float64

	total := 0.0

	for _, dw := range topsoilDepthWeights {
		ras, err := r.raster(property, dw.depth)
		if err != nil {
			return nil, err
		}

		values := ras.ValuesIn(m.Geometry, soilScales[property])
		if combined == nil {
			combined = make([]float64, len(values))
		}

		if len(values) != len(combined) {
			return nil, fmt.Errorf("%s layers are not aligned for %s: %w", property, m.Code, sources.ErrUnsupportedRaster)
		}

		for i, v := range values {
			combined[i] += dw.weight * v
		}

		total += dw.weight
	}

	for i := range combined {
		combined[i] /= total
	}

	return combined, nil
}

func (r *RasterSoil) Summary(_ context.Context, m geo.Municipality) (agro.SoilSummary, error) {
	values := map[string][]float64{}

	for _, p := range sources.SoilProperties {
		v, err := r.topsoil(p, m)
		if err != nil {
			return agro.SoilSummary{}, err
		}

		values[p] = v
	}

	if len(values["clay"]) == 0 {
		return agro.SoilSummary{}, fmt.Errorf("no soil pixels inside %s: %w", m.Code, sources.ErrUnsupportedRaster)
	}

	sub := map[string]float64{}

	for _, p := range []string{"clay", "sand", "soc"} {
		ras, err := r.raster(p, "30-60cm")
		if err != nil {
			return agro.SoilSummary{}, err
		}

		sub[p] = agro.Percentile(ras.ValuesIn(m.Geometry, soilScales[p]), 0.5)
	}

	median := func(p string) float64 { return agro.Percentile(values[p], 0.5) }

	return agro.SoilSummary{
		ClayPercent: median("clay"), SandPercent: median("sand"), SiltPercent: median("silt"),
		OrganicCarbonGPerKg: median("soc"), PH: median("phh2o"), NitrogenGPerKg: median("nitrogen"),
		CECCmolPerKg: median("cec"), BulkDensityKgPerDm3: median("bdod"),
		ClayP05: agro.Percentile(values["clay"], 0.05), ClayP95: agro.Percentile(values["clay"], 0.95),
		SandP05: agro.Percentile(values["sand"], 0.05), SandP95: agro.Percentile(values["sand"], 0.95),
		SubsoilClayPercent: sub["clay"], SubsoilSandPercent: sub["sand"], SubsoilOrganicCarbon: sub["soc"],
		TextureShares: textureShares(values["sand"], values["clay"]),
		Pixels:        len(values["clay"]),
		Method:        "SoilGrids 250 m: mediana de los píxeles del municipio; P05 y P95 describen la variación espacial",
	}, nil
}

func textureShares(sand, clay []float64) map[string]float64 {
	counts := map[string]float64{}

	for i := range sand {
		if !math.IsNaN(sand[i]) && !math.IsNaN(clay[i]) {
			counts[agro.TextureClass(sand[i], clay[i])]++
		}
	}

	total := 0.0
	for _, c := range counts {
		total += c
	}

	for k := range counts {
		counts[k] = math.Round(counts[k]/total*1000) / 1000
	}

	return counts
}
