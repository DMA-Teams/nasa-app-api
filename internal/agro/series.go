package agro

import (
	"errors"
	"fmt"
	"math"
	"time"
)

const (
	lapseRateCelsiusPerMeter = 0.0065
	hoursPerDay              = 24
)

var (
	ErrEmptySeries      = errors.New("empty series")
	ErrMismatchedSeries = errors.New("series do not share start date and length")
)

type CellSeries struct {
	Start          time.Time
	Precipitation  []Millimeters
	TMean          []Celsius
	TMax           []Celsius
	TMin           []Celsius
	RelativeHumid  []float64
	DewPoint       []Celsius
	Wind           []MetersPerSecond
	SolarRadiation []MegajoulesPerSquareMeterDay
	Pressure       []Kilopascals
}

func (s *CellSeries) Len() int {
	return len(s.Precipitation)
}

func (s *CellSeries) Validate() error {
	n := s.Len()
	if n == 0 {
		return ErrEmptySeries
	}

	lengths := []int{len(s.TMean), len(s.TMax), len(s.TMin), len(s.RelativeHumid), len(s.DewPoint), len(s.Wind), len(s.SolarRadiation), len(s.Pressure)}
	for _, l := range lengths {
		if l != n {
			return fmt.Errorf("cell series parameter has %d days, want %d: %w", l, n, ErrMismatchedSeries)
		}
	}

	return nil
}

type WeightedCell struct {
	Series    *CellSeries
	Weight    float64
	Elevation Meters
}

type SiteSeries struct {
	Start         time.Time
	Precipitation []Millimeters
	ET0           []Millimeters
	TMean         []Celsius
	TMax          []Celsius
	TMin          []Celsius
}

func (s *SiteSeries) Len() int {
	return len(s.Precipitation)
}

func (s *SiteSeries) IndexOf(t time.Time) int {
	return int(dateOnly(t).Sub(dateOnly(s.Start)).Hours() / hoursPerDay)
}

func (s *SiteSeries) DateAt(i int) time.Time {
	return dateOnly(s.Start).AddDate(0, 0, i)
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

type SiteSpec struct {
	LatitudeDeg float64
	Elevation   Meters
	Excluded    map[YearMonth]bool
}

type YearMonth struct {
	Year  int
	Month time.Month
}

func BuildSiteSeries(cells []WeightedCell, spec SiteSpec) (*SiteSeries, error) {
	if len(cells) == 0 {
		return nil, ErrEmptySeries
	}

	ref := cells[0].Series
	for _, c := range cells {
		if err := c.Series.Validate(); err != nil {
			return nil, err
		}

		if !c.Series.Start.Equal(ref.Start) || c.Series.Len() != ref.Len() {
			return nil, ErrMismatchedSeries
		}
	}

	n := ref.Len()
	site := &SiteSeries{
		Start:         dateOnly(ref.Start),
		Precipitation: make([]Millimeters, n),
		ET0:           make([]Millimeters, n),
		TMean:         make([]Celsius, n),
		TMax:          make([]Celsius, n),
		TMin:          make([]Celsius, n),
	}

	totalWeight := 0.0
	for _, c := range cells {
		totalWeight += c.Weight
	}

	loc := Location{LatitudeDeg: spec.LatitudeDeg, Elevation: spec.Elevation}
	pressure := AtmosphericPressure(spec.Elevation)

	for i := range n {
		day := blendDay(cells, i, totalWeight, spec.Elevation)
		date := site.DateAt(i)

		if spec.Excluded[YearMonth{Year: date.Year(), Month: date.Month()}] {
			day.precipitation = Millimeters(math.NaN())
		}

		site.Precipitation[i] = day.precipitation
		site.TMean[i] = day.tMean
		site.TMax[i] = day.tMax
		site.TMin[i] = day.tMin
		site.ET0[i] = dailyET0(day, loc, pressure, date.YearDay())
	}

	return site, nil
}

type blendedDay struct {
	precipitation Millimeters
	tMean         Celsius
	tMax          Celsius
	tMin          Celsius
	dewPoint      Celsius
	wind          MetersPerSecond
	solar         MegajoulesPerSquareMeterDay
}

func blendDay(cells []WeightedCell, i int, totalWeight float64, siteElevation Meters) blendedDay {
	var d blendedDay

	for _, c := range cells {
		w := c.Weight / totalWeight
		shift := Celsius(-lapseRateCelsiusPerMeter * float64(siteElevation-c.Elevation))
		s := c.Series
		d.precipitation += Millimeters(w) * s.Precipitation[i]
		d.tMean += Celsius(w) * (s.TMean[i] + shift)
		d.tMax += Celsius(w) * (s.TMax[i] + shift)
		d.tMin += Celsius(w) * (s.TMin[i] + shift)
		d.dewPoint += Celsius(w) * s.DewPoint[i]
		d.wind += MetersPerSecond(w) * s.Wind[i]
		d.solar += MegajoulesPerSquareMeterDay(w) * s.SolarRadiation[i]
	}

	return d
}

func dailyET0(d blendedDay, loc Location, pressure Kilopascals, dayOfYear int) Millimeters {
	meteo := DailyMeteo{
		TMax:                d.tMax,
		TMin:                d.tMin,
		ActualVaporPressure: VaporPressureFromDewPoint(min(d.dewPoint, d.tMax)),
		WindAt2m:            d.wind,
		SolarRadiation:      d.solar,
		Pressure:            pressure,
	}

	et0 := PenmanMonteith(meteo, loc, dayOfYear)
	if !math.IsNaN(float64(et0)) {
		return et0
	}

	return Hargreaves(d.tMax, d.tMin, ExtraterrestrialRadiation(loc.LatitudeDeg, dayOfYear))
}
