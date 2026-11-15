package sources

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/paulmach/orb"
)

const (
	elevationBaseURL     = "https://api.open-meteo.com/v1/elevation"
	seasonalBaseURL      = "https://seasonal-api.open-meteo.com/v1/seasonal"
	elevationBatchSize   = 100
	seasonalForecastDays = 215
	SeasonalModel        = "ECMWF SEAS5 (Open-Meteo)"
)

type OpenMeteoClient struct {
	client       *Client
	elevationURL string
	seasonalURL  string
}

func NewOpenMeteoClient(c *Client) *OpenMeteoClient {
	return &OpenMeteoClient{client: c, elevationURL: elevationBaseURL, seasonalURL: seasonalBaseURL}
}

func (o *OpenMeteoClient) Elevations(ctx context.Context, points []orb.Point) ([]float64, error) {
	out := make([]float64, 0, len(points))

	for start := 0; start < len(points); start += elevationBatchSize {
		batch := points[start:min(start+elevationBatchSize, len(points))]
		lats, lons := make([]string, len(batch)), make([]string, len(batch))

		for i, p := range batch {
			lats[i] = strconv.FormatFloat(p.Lat(), 'f', 4, 64)
			lons[i] = strconv.FormatFloat(p.Lon(), 'f', 4, 64)
		}

		var resp struct {
			Elevation []float64 `json:"elevation"`
		}

		u := o.elevationURL + "?latitude=" + strings.Join(lats, ",") + "&longitude=" + strings.Join(lons, ",")
		if err := o.client.GetJSON(ctx, u, &resp); err != nil {
			return nil, err
		}

		if len(resp.Elevation) != len(batch) {
			return nil, fmt.Errorf("elevation batch returned %d values for %d points: %w", len(resp.Elevation), len(batch), ErrUnexpectedPayload)
		}

		out = append(out, resp.Elevation...)
	}

	return out, nil
}

type SeasonalForecast struct {
	Dates          []time.Time
	Members        [][]float64
	MonthlyAnomaly map[[2]int]float64
}

func (o *OpenMeteoClient) Seasonal(ctx context.Context, lat, lon float64) (SeasonalForecast, error) {
	base := url.Values{}
	base.Set("latitude", strconv.FormatFloat(lat, 'f', 4, 64))
	base.Set("longitude", strconv.FormatFloat(lon, 'f', 4, 64))
	base.Set("forecast_days", strconv.Itoa(seasonalForecastDays))

	daily := cloneValues(base)
	daily.Set("daily", "precipitation_sum")

	var dailyResp struct {
		Daily map[string]any `json:"daily"`
	}

	if err := o.client.GetJSON(ctx, o.seasonalURL+"?"+daily.Encode(), &dailyResp); err != nil {
		return SeasonalForecast{}, err
	}

	monthly := cloneValues(base)
	monthly.Set("monthly", "precipitation_anomaly")

	var monthlyResp struct {
		Monthly struct {
			Time    []string   `json:"time"`
			Anomaly []*float64 `json:"precipitation_anomaly"`
		} `json:"monthly"`
	}

	if err := o.client.GetJSON(ctx, o.seasonalURL+"?"+monthly.Encode(), &monthlyResp); err != nil {
		return SeasonalForecast{}, err
	}

	return parseSeasonal(dailyResp.Daily, monthlyResp.Monthly.Time, monthlyResp.Monthly.Anomaly)
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vs := range v {
		out[k] = append([]string(nil), vs...)
	}

	return out
}

func parseSeasonal(daily map[string]any, months []string, anomalies []*float64) (SeasonalForecast, error) {
	rawTimes, ok := daily["time"].([]any)
	if !ok || len(rawTimes) == 0 {
		return SeasonalForecast{}, fmt.Errorf("seasonal response without daily time: %w", ErrUnexpectedPayload)
	}

	f := SeasonalForecast{MonthlyAnomaly: map[[2]int]float64{}}

	for _, rt := range rawTimes {
		s, _ := rt.(string)

		t, err := time.Parse(time.DateOnly, s)
		if err != nil {
			return SeasonalForecast{}, fmt.Errorf("seasonal date %q: %w", s, err)
		}

		f.Dates = append(f.Dates, t)
	}

	for key, raw := range daily {
		if !strings.HasPrefix(key, "precipitation_sum_member") {
			continue
		}

		values, ok := raw.([]any)
		if !ok || len(values) != len(f.Dates) {
			continue
		}

		member := make([]float64, len(values))
		for i, v := range values {
			member[i] = math.NaN()
			if n, ok := v.(float64); ok {
				member[i] = n
			}
		}

		f.Members = append(f.Members, member)
	}

	for i, m := range months {
		t, err := time.Parse(time.DateOnly, m)
		if err != nil || i >= len(anomalies) || anomalies[i] == nil {
			continue
		}

		f.MonthlyAnomaly[[2]int{t.Year(), int(t.Month())}] = *anomalies[i]
	}

	return f, nil
}

func (f SeasonalForecast) FullMonths() [][2]int {
	if len(f.Dates) == 0 {
		return nil
	}

	var out [][2]int

	first, last := f.Dates[0], f.Dates[len(f.Dates)-1]
	for m := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC); !m.After(last); m = m.AddDate(0, 1, 0) {
		end := m.AddDate(0, 1, -1)
		if !m.Before(first) && !end.After(last) {
			out = append(out, [2]int{m.Year(), int(m.Month())})
		}
	}

	return out
}

func (f SeasonalForecast) MemberAnomalies(year, month int) ([]float64, bool) {
	anomaly, ok := f.MonthlyAnomaly[[2]int{year, month}]
	if !ok || len(f.Members) == 0 {
		return nil, false
	}

	totals := make([]float64, len(f.Members))
	ensembleMean := 0.0

	for m, member := range f.Members {
		for i, d := range f.Dates {
			if d.Year() == year && int(d.Month()) == month {
				totals[m] += member[i]
			}
		}

		ensembleMean += totals[m]
	}

	ensembleMean /= float64(len(f.Members))
	climatology := ensembleMean - anomaly

	out := make([]float64, len(totals))
	for i, t := range totals {
		out[i] = t - climatology
	}

	return out, true
}
