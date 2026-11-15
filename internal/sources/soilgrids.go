package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"

	"github.com/dma-teams/nasa-app-api/internal/agro"
)

const soilGridsBaseURL = "https://rest.isric.org/soilgrids/v2.0/properties/query"

var (
	SoilProperties = []string{"clay", "sand", "silt", "soc", "phh2o", "nitrogen", "cec", "bdod"}
	SoilDepths     = []string{"0-5cm", "5-15cm", "15-30cm", "30-60cm"}
	topsoilWeights = map[string]float64{"0-5cm": 5, "5-15cm": 10, "15-30cm": 15}
)

type SoilValue struct {
	Mean float64
	Q05  float64
	Q95  float64
}

type SoilPoint map[string]map[string]SoilValue

type SoilGridsClient struct {
	client  *Client
	baseURL string
}

func NewSoilGridsClient(c *Client) *SoilGridsClient {
	return &SoilGridsClient{client: c, baseURL: soilGridsBaseURL}
}

func (s *SoilGridsClient) Point(ctx context.Context, lat, lon float64) (SoilPoint, error) {
	q := url.Values{}
	q.Set("lat", strconv.FormatFloat(lat, 'f', 4, 64))
	q.Set("lon", strconv.FormatFloat(lon, 'f', 4, 64))

	for _, p := range SoilProperties {
		q.Add("property", p)
	}

	for _, d := range SoilDepths {
		q.Add("depth", d)
	}

	for _, v := range []string{"mean", "Q0.05", "Q0.95"} {
		q.Add("value", v)
	}

	body, err := s.client.Get(ctx, s.baseURL+"?"+q.Encode())
	if err != nil {
		return nil, err
	}

	return ParseSoilGrids(body)
}

func ParseSoilGrids(body []byte) (SoilPoint, error) {
	var resp struct {
		Properties struct {
			Layers []struct {
				Name        string `json:"name"`
				UnitMeasure struct {
					DFactor float64 `json:"d_factor"`
				} `json:"unit_measure"`
				Depths []struct {
					Label  string              `json:"label"`
					Values map[string]*float64 `json:"values"`
				} `json:"depths"`
			} `json:"layers"`
		} `json:"properties"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode SoilGrids response: %w", err)
	}

	if len(resp.Properties.Layers) == 0 {
		return nil, fmt.Errorf("SoilGrids response without layers: %w", ErrUnexpectedPayload)
	}

	out := SoilPoint{}

	for _, layer := range resp.Properties.Layers {
		factor := layer.UnitMeasure.DFactor
		if factor == 0 {
			factor = 1
		}

		out[layer.Name] = map[string]SoilValue{}

		for _, d := range layer.Depths {
			out[layer.Name][d.Label] = SoilValue{
				Mean: scaled(d.Values["mean"], factor),
				Q05:  scaled(d.Values["Q0.05"], factor),
				Q95:  scaled(d.Values["Q0.95"], factor),
			}
		}
	}

	return out, nil
}

func scaled(v *float64, factor float64) float64 {
	if v == nil {
		return math.NaN()
	}

	return *v / factor
}

func (p SoilPoint) Topsoil(property string, pick func(SoilValue) float64) float64 {
	sum, weight := 0.0, 0.0

	for depth, w := range topsoilWeights {
		v := pick(p[property][depth])
		if math.IsNaN(v) {
			continue
		}

		sum += w * v
		weight += w
	}

	if weight == 0 {
		return math.NaN()
	}

	return sum / weight
}

func (p SoilPoint) Summary() agro.SoilSummary {
	mean := func(v SoilValue) float64 { return v.Mean }
	q05 := func(v SoilValue) float64 { return v.Q05 }
	q95 := func(v SoilValue) float64 { return v.Q95 }
	sub := p["clay"]["30-60cm"]

	return agro.SoilSummary{
		ClayPercent: p.Topsoil("clay", mean), SandPercent: p.Topsoil("sand", mean), SiltPercent: p.Topsoil("silt", mean),
		OrganicCarbonGPerKg: p.Topsoil("soc", mean), PH: p.Topsoil("phh2o", mean), NitrogenGPerKg: p.Topsoil("nitrogen", mean),
		CECCmolPerKg: p.Topsoil("cec", mean), BulkDensityKgPerDm3: p.Topsoil("bdod", mean),
		ClayP05: p.Topsoil("clay", q05), ClayP95: p.Topsoil("clay", q95),
		SandP05: p.Topsoil("sand", q05), SandP95: p.Topsoil("sand", q95),
		SubsoilClayPercent: sub.Mean, SubsoilSandPercent: p["sand"]["30-60cm"].Mean,
		SubsoilOrganicCarbon: p["soc"]["30-60cm"].Mean,
		Pixels:               1,
		Method:               "SoilGrids REST: punto representativo del municipio; P05 y P95 son cuantiles del modelo",
	}
}
