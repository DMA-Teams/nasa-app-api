package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/dma-teams/nasa-app-api/internal/agro"
	"github.com/dma-teams/nasa-app-api/internal/geo"
)

const (
	powerBaseURL    = "https://power.larc.nasa.gov/api/temporal/daily/point"
	powerParameters = "PRECTOTCORR,T2M,T2M_MAX,T2M_MIN,RH2M,T2MDEW,WS2M,ALLSKY_SFC_SW_DWN,PS"
	powerDateLayout = "20060102"
)

var ErrUnexpectedPayload = errors.New("unexpected payload from source")

type PowerClient struct {
	client  *Client
	baseURL string
}

func NewPowerClient(c *Client) *PowerClient {
	return &PowerClient{client: c, baseURL: powerBaseURL}
}

type powerResponse struct {
	Geometry struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		Parameter map[string]map[string]float64 `json:"parameter"`
	} `json:"properties"`
	Header struct {
		FillValue float64  `json:"fill_value"`
		Sources   []string `json:"sources"`
	} `json:"header"`
}

type PowerSeries struct {
	Series    *agro.CellSeries
	Elevation agro.Meters
	Sources   []string
}

func (p *PowerClient) Daily(ctx context.Context, cell geo.Cell, from, to time.Time) (PowerSeries, error) {
	q := url.Values{}
	q.Set("parameters", powerParameters)
	q.Set("community", "AG")
	q.Set("latitude", fmt.Sprintf("%.4f", cell.Lat))
	q.Set("longitude", fmt.Sprintf("%.4f", cell.Lon))
	q.Set("start", from.Format(powerDateLayout))
	q.Set("end", to.Format(powerDateLayout))
	q.Set("format", "JSON")
	q.Set("time-standard", "LST")

	body, err := p.client.Get(ctx, p.baseURL+"?"+q.Encode())
	if err != nil {
		return PowerSeries{}, err
	}

	return ParsePowerDaily(body, from, to)
}

func ParsePowerDaily(body []byte, from, to time.Time) (PowerSeries, error) {
	var resp powerResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return PowerSeries{}, fmt.Errorf("decode POWER response: %w", err)
	}

	params := resp.Properties.Parameter
	if len(params) == 0 || len(resp.Geometry.Coordinates) < 3 {
		return PowerSeries{}, fmt.Errorf("POWER response without parameters or elevation: %w", ErrUnexpectedPayload)
	}

	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	days := int(to.Sub(from).Hours()/24) + 1
	s := &agro.CellSeries{Start: start}
	fill := resp.Header.FillValue

	read := func(name string) []float64 {
		values := make([]float64, days)
		series := params[name]

		for i := range days {
			v, ok := series[start.AddDate(0, 0, i).Format(powerDateLayout)]
			if !ok || v == fill || v <= -998 {
				v = math.NaN()
			}

			values[i] = v
		}

		return values
	}

	s.Precipitation = convert[agro.Millimeters](read("PRECTOTCORR"))
	s.TMean = convert[agro.Celsius](read("T2M"))
	s.TMax = convert[agro.Celsius](read("T2M_MAX"))
	s.TMin = convert[agro.Celsius](read("T2M_MIN"))
	s.RelativeHumid = read("RH2M")
	s.DewPoint = convert[agro.Celsius](read("T2MDEW"))
	s.Wind = convert[agro.MetersPerSecond](read("WS2M"))
	s.SolarRadiation = convert[agro.MegajoulesPerSquareMeterDay](read("ALLSKY_SFC_SW_DWN"))
	s.Pressure = convert[agro.Kilopascals](read("PS"))

	return PowerSeries{Series: s, Elevation: agro.Meters(resp.Geometry.Coordinates[2]), Sources: resp.Header.Sources}, nil
}

func convert[T ~float64](values []float64) []T {
	out := make([]T, len(values))
	for i, v := range values {
		out[i] = T(v)
	}

	return out
}
