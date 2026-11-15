package agro

const (
	SourceClimate  = "NASA POWER 1996-2025 (MERRA-2)"
	SourceWater    = "FAO-56 balance hídrico; FAO-33 respuesta del rendimiento al agua"
	SourceForecast = "ECMWF SEAS5 vía Open-Meteo"
	SourceSoil     = "SoilGrids 2.0 (ISRIC); Saxton y Rawls (2006)"
	SourceErosion  = "RUSLE relativa con índice de Fournier modificado"
	SourceCatalog  = "Ficha de cultivos del proyecto"
)

type ReasonCode string

const (
	ReasonLossProbability   ReasonCode = "loss_probability"
	ReasonExpectedYield     ReasonCode = "expected_relative_yield"
	ReasonCaniculaFlowering ReasonCode = "canicula_during_flowering"
	ReasonHeatStress        ReasonCode = "heat_stress_days"
	ReasonIrrigation        ReasonCode = "irrigation_required_mm"
	ReasonUnsuitable        ReasonCode = "low_suitability"
	ReasonLowConfidence     ReasonCode = "low_confidence_parameters"
	ReasonBareSoil          ReasonCode = "bare_soil_rainy_months"
	ReasonErosion           ReasonCode = "relative_erosion"
	ReasonNitrogen          ReasonCode = "nitrogen_balance_kg_ha"
	ReasonFamilyRepeated    ReasonCode = "family_repeated"
	ReasonForecastWeighted  ReasonCode = "forecast_weighted"
	ReasonInfeasible        ReasonCode = "no_viable_sowing_window"
)

type Reason struct {
	Code   ReasonCode `json:"code"`
	Crop   string     `json:"crop,omitempty"`
	Value  float64    `json:"value"`
	Detail string     `json:"detail,omitempty"`
	Source string     `json:"source"`
}

func (ev *evaluator) reasons(plantings []*plantingStats, soil SoilIndicators, feasible bool) []Reason {
	out := make([]Reason, 0, len(plantings)*4+5)

	if !feasible {
		out = append(out, Reason{Code: ReasonInfeasible, Value: float64(len(plantings)), Source: SourceWater})
	}

	forecastUsed := false

	for i, p := range plantings {
		out = append(out, plantingReasons(p)...)

		if i > 0 && plantings[i-1].crop.Family == p.crop.Family {
			out = append(out, Reason{Code: ReasonFamilyRepeated, Crop: p.crop.ID, Value: 1, Detail: p.crop.Family, Source: SourceCatalog})
		}

		for _, k := range p.months {
			forecastUsed = forecastUsed || ev.forecastMask[k]
		}
	}

	out = append(out,
		Reason{Code: ReasonBareSoil, Value: float64(soil.BareRainyMonths), Source: SourceClimate},
		Reason{Code: ReasonErosion, Value: soil.RelativeErosion, Source: SourceErosion},
		Reason{Code: ReasonNitrogen, Value: soil.NitrogenBalanceKgHa, Source: SourceCatalog},
	)

	if forecastUsed {
		out = append(out, Reason{Code: ReasonForecastWeighted, Value: float64(ev.forecastMonths), Source: SourceForecast})
	}

	return out
}

func plantingReasons(p *plantingStats) []Reason {
	id := p.crop.ID
	out := []Reason{
		{Code: ReasonLossProbability, Crop: id, Value: round3(p.lossProbability), Source: SourceClimate + "; " + SourceWater},
		{Code: ReasonExpectedYield, Crop: id, Value: round3(p.expectedYield), Source: SourceWater},
	}

	if p.caniculaShare >= caniculaReasonShare {
		out = append(out, Reason{Code: ReasonCaniculaFlowering, Crop: id, Value: round3(p.caniculaShare), Source: SourceClimate})
	}

	if p.heatDays >= heatReasonDays {
		out = append(out, Reason{Code: ReasonHeatStress, Crop: id, Value: round1(p.heatDays), Source: SourceClimate})
	}

	if p.irrigationMm > 0 {
		out = append(out, Reason{Code: ReasonIrrigation, Crop: id, Value: round1(p.irrigationMm), Source: SourceWater})
	}

	if float64(p.suitability) < unsuitableReasonShare {
		out = append(out, Reason{Code: ReasonUnsuitable, Crop: id, Value: round3(float64(p.suitability)), Detail: string(p.limit), Source: SourceSoil + "; " + SourceCatalog})
	}

	if n := len(p.crop.LowConfidenceFields); n > 0 {
		out = append(out, Reason{Code: ReasonLowConfidence, Crop: id, Value: float64(n), Source: SourceCatalog})
	}

	return out
}
