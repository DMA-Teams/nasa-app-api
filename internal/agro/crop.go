package agro

import (
	"math"
	"slices"
)

type CropRole int

const (
	RoleUnknown CropRole = iota
	RoleHarvest
	RoleCover
)

type IrrigationNeed int

const (
	IrrigationUnknown IrrigationNeed = iota
	IrrigationOptional
	IrrigationRequired
)

type WaterloggingTolerance int

const (
	WaterloggingUnknown WaterloggingTolerance = iota
	WaterloggingSensitive
	WaterloggingModerate
	WaterloggingTolerant
)

type RelayKind int

const (
	RelayNone RelayKind = iota
	RelayCover
	RelayGrain
)

type Stage int

const (
	StageInitial Stage = iota
	StageDevelopment
	StageMid
	StageLate
)

type Phase int

const (
	PhaseVegetative Phase = iota
	PhaseFlowering
	PhaseYieldFormation
	PhaseRipening
)

const (
	floweringShareOfMidStage   = 0.4
	referenceStageTemperatureC = 24.0
	minimumStageScale          = 0.7
	maximumStageScale          = 1.8
)

type TemperatureRange struct {
	Base       Celsius
	AbsoluteLo Celsius
	OptimalLo  Celsius
	OptimalHi  Celsius
	AbsoluteHi Celsius
}

type PHRange struct {
	AbsoluteLo float64
	OptimalLo  float64
	OptimalHi  float64
	AbsoluteHi float64
}

type Crop struct {
	ID                      string
	Name                    string
	Family                  string
	Role                    CropRole
	StageDays               [4]int
	KcIni                   float64
	KcMid                   float64
	KcEnd                   float64
	MaxRootDepth            Meters
	DepletionFraction       Fraction
	KyTotal                 float64
	KyByPhase               [4]float64
	HasKyByPhase            bool
	Temperature             TemperatureRange
	HeatStressTMax          Celsius
	PH                      PHRange
	AltitudeMin             Meters
	AltitudeMax             Meters
	Irrigation              IrrigationNeed
	Waterlogging            WaterloggingTolerance
	NitrogenFixationKgHa    float64
	NitrogenRemovalKgPerTon float64
	ReferenceYieldTonHa     float64
	CoverFactorByStage      [4]float64
	RelayGuest              RelayKind
	RelayHosts              []RelayKind
	LowConfidenceFields     []string
}

func (c *Crop) IsLegume() bool {
	return c.Family == "Fabaceae"
}

func (c *Crop) AcceptsRelay(guest *Crop) bool {
	return guest.RelayGuest != RelayNone && slices.Contains(c.RelayHosts, guest.RelayGuest)
}

func (c *Crop) RelayWindow(guest *Crop, stages [4]int) (from, to int) {
	cycle := stages[0] + stages[1] + stages[2] + stages[3]

	switch guest.RelayGuest {
	case RelayCover:
		return relayMinimumDays, cycle
	case RelayGrain:
		return cycle - stages[3], cycle
	case RelayNone:
		return 0, 0
	default:
		return 0, 0
	}
}

func (c *Crop) CycleDays() int {
	return c.StageDays[0] + c.StageDays[1] + c.StageDays[2] + c.StageDays[3]
}

func (c *Crop) ScaledStages(scale float64) [4]int {
	var out [4]int

	for i, d := range c.StageDays {
		out[i] = max(1, int(math.Round(float64(d)*scale)))
	}

	return out
}

func StageScaleForTemperature(meanTemperature, base Celsius) float64 {
	effective := float64(meanTemperature - base)
	if effective <= 0 {
		return maximumStageScale
	}

	return Clamp((referenceStageTemperatureC-float64(base))/effective, minimumStageScale, maximumStageScale)
}

func StageOnDay(stages [4]int, day int) Stage {
	limit := 0

	for i, d := range stages {
		limit += d
		if day < limit {
			return Stage(i)
		}
	}

	return StageLate
}

func PhaseOnDay(stages [4]int, day int) Phase {
	vegetativeEnd := stages[0] + stages[1]
	floweringEnd := vegetativeEnd + int(math.Ceil(float64(stages[2])*floweringShareOfMidStage))
	yieldEnd := vegetativeEnd + stages[2]

	switch {
	case day < vegetativeEnd:
		return PhaseVegetative
	case day < floweringEnd:
		return PhaseFlowering
	case day < yieldEnd:
		return PhaseYieldFormation
	default:
		return PhaseRipening
	}
}

func (c *Crop) KcOnDay(stages [4]int, day int) float64 {
	ini, dev, mid := stages[0], stages[1], stages[2]

	switch {
	case day < ini:
		return c.KcIni
	case day < ini+dev:
		return c.KcIni + (c.KcMid-c.KcIni)*float64(day-ini+1)/float64(dev)
	case day < ini+dev+mid:
		return c.KcMid
	default:
		late := max(stages[3], 1)
		progress := Clamp(float64(day-ini-dev-mid+1)/float64(late), 0, 1)

		return c.KcMid + (c.KcEnd-c.KcMid)*progress
	}
}
