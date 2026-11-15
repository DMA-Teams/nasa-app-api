package agro

import (
	"math"
	"time"
)

const (
	subsoilShareOfRootZone = 0.5
	percentToFraction      = 100.0
)

type SoilSummary struct {
	ClayPercent          float64            `json:"clayPercent"`
	SandPercent          float64            `json:"sandPercent"`
	SiltPercent          float64            `json:"siltPercent"`
	OrganicCarbonGPerKg  float64            `json:"organicCarbonGPerKg"`
	PH                   float64            `json:"ph"`
	NitrogenGPerKg       float64            `json:"nitrogenGPerKg"`
	CECCmolPerKg         float64            `json:"cecCmolPerKg"`
	BulkDensityKgPerDm3  float64            `json:"bulkDensityKgPerDm3"`
	ClayP05              float64            `json:"clayP05Percent"`
	ClayP95              float64            `json:"clayP95Percent"`
	SandP05              float64            `json:"sandP05Percent"`
	SandP95              float64            `json:"sandP95Percent"`
	SubsoilClayPercent   float64            `json:"subsoilClayPercent"`
	SubsoilSandPercent   float64            `json:"subsoilSandPercent"`
	SubsoilOrganicCarbon float64            `json:"subsoilOrganicCarbonGPerKg"`
	TextureShares        map[string]float64 `json:"textureShares,omitempty"`
	Pixels               int                `json:"pixels"`
	Method               string             `json:"method"`
}

func (s SoilSummary) AvailableWaterPerMeter() Millimeters {
	return availableWater(s.SandPercent, s.ClayPercent, s.OrganicCarbonGPerKg, s.SubsoilSandPercent, s.SubsoilClayPercent, s.SubsoilOrganicCarbon)
}

func (s SoilSummary) AvailableWaterRange() (Millimeters, Millimeters) {
	sandy := availableWater(s.SandP95, s.ClayP05, s.OrganicCarbonGPerKg, s.SandP95, s.ClayP05, s.SubsoilOrganicCarbon)
	clayey := availableWater(s.SandP05, s.ClayP95, s.OrganicCarbonGPerKg, s.SandP05, s.ClayP95, s.SubsoilOrganicCarbon)

	return min(sandy, clayey), max(sandy, clayey)
}

func availableWater(sand, clay, soc, subSand, subClay, subSoc float64) Millimeters {
	top := SaxtonRawls(SoilTexture{
		Sand: Fraction(sand / percentToFraction), Clay: Fraction(clay / percentToFraction),
		OrganicMatterPercent: OrganicMatterPercentFromCarbon(soc),
	}).AvailableWaterPerMeter()

	if subSand == 0 && subClay == 0 {
		return top
	}

	sub := SaxtonRawls(SoilTexture{
		Sand: Fraction(subSand / percentToFraction), Clay: Fraction(subClay / percentToFraction),
		OrganicMatterPercent: OrganicMatterPercentFromCarbon(subSoc),
	}).AvailableWaterPerMeter()

	return Millimeters((1-subsoilShareOfRootZone)*float64(top) + subsoilShareOfRootZone*float64(sub))
}

func TextureClass(sandPercent, clayPercent float64) string {
	silt := 100 - sandPercent - clayPercent

	switch {
	case clayPercent >= 40 && silt < 40 && sandPercent <= 45:
		return "clay"
	case clayPercent >= 40 && silt >= 40:
		return "silty_clay"
	case clayPercent >= 35 && sandPercent > 45:
		return "sandy_clay"
	case clayPercent >= 27 && sandPercent <= 20:
		return "silty_clay_loam"
	case clayPercent >= 27 && sandPercent <= 45:
		return "clay_loam"
	case clayPercent >= 20 && sandPercent > 45:
		return "sandy_clay_loam"
	case silt >= 80 && clayPercent < 12:
		return "silt"
	case silt >= 50:
		return "silt_loam"
	case sandPercent >= 85:
		return "sand"
	case sandPercent >= 70:
		return "loamy_sand"
	case sandPercent > 52:
		return "sandy_loam"
	default:
		return "loam"
	}
}

type CellWeight struct {
	CellID    string  `json:"cellId"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Weight    float64 `json:"weight"`
	Elevation Meters  `json:"elevationM"`
}

type SiteProfile struct {
	MunicipalityCode    string                              `json:"municipalityCode"`
	MunicipalityName    string                              `json:"municipalityName"`
	DepartmentCode      string                              `json:"departmentCode"`
	DepartmentName      string                              `json:"departmentName"`
	LatitudeDeg         float64                             `json:"latitude"`
	LongitudeDeg        float64                             `json:"longitude"`
	Cells               []CellWeight                        `json:"cells"`
	ElevationMedian     Meters                              `json:"elevationMedianM"`
	ElevationMin        Meters                              `json:"elevationMinM"`
	ElevationMax        Meters                              `json:"elevationMaxM"`
	ElevationSamples    int                                 `json:"elevationSamples"`
	Soil                SoilSummary                         `json:"soil"`
	Fortnights          [fortnightsPerYear]FortnightClimate `json:"fortnights"`
	MonthlyPrecipMedian [12]float64                         `json:"monthlyPrecipitationMedianMm"`
	OnsetMedianDOY      int                                 `json:"onsetMedianDayOfYear"`
	OnsetYears          int                                 `json:"onsetYears"`
	Canicula            DayWindow                           `json:"canicula"`
	CaniculaYears       int                                 `json:"caniculaYears"`
	Years               int                                 `json:"years"`
	ExcludedMonths      []YearMonth                         `json:"excludedMonths"`
	DatasetVersion      string                              `json:"datasetVersion"`
}

func (p *SiteProfile) Conditions(meanTemperature Celsius, irrigated bool) SiteConditions {
	return SiteConditions{
		MeanTemperature: meanTemperature,
		PH:              p.Soil.PH,
		ClayPercent:     p.Soil.ClayPercent,
		Elevation:       p.ElevationMedian,
		Irrigated:       irrigated,
	}
}

func (p *SiteProfile) FillClimate(s *SiteSeries) {
	p.Fortnights = FortnightClimatology(s)
	p.MonthlyPrecipMedian = MonthlyMedianPrecipitation(s)

	first, last := s.DateAt(0).Year(), s.DateAt(s.Len()-1).Year()
	p.Years = last - first + 1

	var onsets, starts, ends []float64

	for y := first; y <= last; y++ {
		if doy, ok := OnsetOfRains(s, y); ok {
			onsets = append(onsets, float64(doy))
		}

		if w, ok := Canicula(s, y); ok {
			starts = append(starts, float64(w.StartDOY))
			ends = append(ends, float64(w.EndDOY))
		}
	}

	p.OnsetYears = len(onsets)
	p.CaniculaYears = len(starts)
	p.OnsetMedianDOY = roundedPercentile(onsets)
	p.Canicula = DayWindow{StartDOY: roundedPercentile(starts), EndDOY: roundedPercentile(ends)}
}

func roundedPercentile(values []float64) int {
	v := Percentile(values, 0.5)
	if math.IsNaN(v) {
		return 0
	}

	return int(math.Round(v))
}

func ExcludedSet(months []YearMonth) map[YearMonth]bool {
	set := make(map[YearMonth]bool, len(months))
	for _, m := range months {
		set[m] = true
	}

	return set
}

func DayOfYearToDate(year, doy int) time.Time {
	return time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, doy-1)
}
