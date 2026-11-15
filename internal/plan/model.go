package plan

import (
	"github.com/dma-teams/nasa-app-api/internal/agro"
)

const Disclaimer = "Herramienta de exploración basada en probabilidades históricas y pronóstico estacional. No reemplaza al técnico agrónomo."

type Municipality struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	DepartmentCode string `json:"departmentCode"`
	DepartmentName string `json:"departmentName,omitempty"`
}

type DataSource struct {
	Name       string `json:"name"`
	Use        string `json:"use"`
	Resolution string `json:"resolution"`
	Coverage   string `json:"coverage"`
}

var Sources = []DataSource{
	{Name: "NASA POWER (MERRA-2)", Use: "Clima diario 1996-2025", Resolution: "0,5° × 0,625° (~55 × 67 km)", Coverage: "Global"},
	{Name: "SoilGrids 2.0 (ISRIC)", Use: "Propiedades del suelo 0-60 cm", Resolution: "250 m", Coverage: "Global; modelo estadístico, no análisis de laboratorio"},
	{Name: "ECMWF SEAS5 vía Open-Meteo", Use: "Pronóstico estacional de lluvia por terciles", Resolution: "~36 km", Coverage: "Global; hasta 7 meses"},
	{Name: "Copernicus DEM vía Open-Meteo", Use: "Altitud del municipio", Resolution: "90 m", Coverage: "Global"},
	{Name: "geoBoundaries (ICF Honduras)", Use: "Límites de municipios y departamentos", Resolution: "Vectorial simplificado", Coverage: "Honduras"},
}

type Site struct {
	ElevationMedianM      float64                      `json:"elevationMedianM"`
	ElevationMinM         float64                      `json:"elevationMinM"`
	ElevationMaxM         float64                      `json:"elevationMaxM"`
	Cells                 []agro.CellWeight            `json:"cells"`
	Soil                  agro.SoilSummary             `json:"soil"`
	AvailableWaterMmPerM  float64                      `json:"availableWaterMmPerM"`
	AvailableWaterRangeMm [2]float64                   `json:"availableWaterRangeMmPerM"`
	Fortnights            []agro.FortnightClimate      `json:"fortnights"`
	OnsetMedianDayOfYear  int                          `json:"onsetMedianDayOfYear"`
	Canicula              agro.DayWindow               `json:"canicula"`
	CaniculaYears         int                          `json:"caniculaYears"`
	Years                 int                          `json:"years"`
	ExcludedMonths        []agro.YearMonth             `json:"excludedMonths"`
	Forecast              []*agro.TercileProbabilities `json:"forecastByPlanMonth,omitempty"`
}

type MunicipalPlan struct {
	Municipality Municipality    `json:"municipality"`
	GeneratedAt  string          `json:"generatedAt"`
	Site         Site            `json:"site"`
	Evaluation   agro.Evaluation `json:"evaluation"`
	Versions     Versions        `json:"versions"`
	Warnings     []string        `json:"warnings"`
	Sources      []DataSource    `json:"sources"`
	Disclaimer   string          `json:"disclaimer"`
}

type Range struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type Group struct {
	Signature          string         `json:"signature"`
	RotationName       string         `json:"rotationName"`
	Crops              []string       `json:"crops"`
	Municipalities     []Municipality `json:"municipalities"`
	TotalScore         Range          `json:"totalScore"`
	MaxLossProbability Range          `json:"maxLossProbability"`
}

type DepartmentPlan struct {
	Department     Municipality `json:"department"`
	GeneratedAt    string       `json:"generatedAt"`
	Municipalities int          `json:"municipalities"`
	Groups         []Group      `json:"groups"`
	Versions       Versions     `json:"versions"`
	Warnings       []string     `json:"warnings"`
	Disclaimer     string       `json:"disclaimer"`
}
