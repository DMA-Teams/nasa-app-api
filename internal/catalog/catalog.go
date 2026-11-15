package catalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dma-teams/nasa-app-api/internal/agro"
)

var ErrInvalidCatalog = errors.New("invalid crop catalog")

type Confidence string

const (
	ConfidenceA Confidence = "A"
	ConfidenceB Confidence = "B"
	ConfidenceC Confidence = "C"
)

type Field[T any] struct {
	Value      T          `json:"value"`
	Source     string     `json:"source"`
	Confidence Confidence `json:"confidence"`
}

type CropEntry struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Family          string             `json:"family"`
	Role            string             `json:"role"`
	Stages          Field[[4]int]      `json:"stages"`
	Kc              Field[[3]float64]  `json:"kc"`
	RootDepthM      Field[float64]     `json:"rootDepthM"`
	Depletion       Field[float64]     `json:"depletion"`
	KyTotal         Field[float64]     `json:"kyTotal"`
	KyByPhase       *Field[[4]float64] `json:"kyByPhase,omitempty"`
	Temperature     Field[[5]float64]  `json:"temperatureC"`
	HeatStressTMax  Field[float64]     `json:"heatStressTMaxC"`
	PH              Field[[4]float64]  `json:"ph"`
	AltitudeM       Field[[2]float64]  `json:"altitudeM"`
	Irrigation      Field[string]      `json:"irrigation"`
	Waterlogging    Field[string]      `json:"waterlogging"`
	Nitrogen        Field[[2]float64]  `json:"nitrogenKgHa"`
	ReferenceYield  Field[float64]     `json:"referenceYieldTHa"`
	CoverFactor     Field[[4]float64]  `json:"coverFactor"`
	Relay           *Field[RelaySpec]  `json:"relay,omitempty"`
	SeasonsHonduras []string           `json:"seasonsHonduras"`
	Notes           string             `json:"notes,omitempty"`
}

type TemplateEntry struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Crops []string `json:"crops"`
	Relay []bool   `json:"relay"`
	Mulch bool     `json:"mulch"`
}

type Document struct {
	Version   string          `json:"version"`
	Crops     []CropEntry     `json:"crops"`
	Templates []TemplateEntry `json:"templates"`
}

type Catalog struct {
	Version   string
	Hash      string
	Document  Document
	Crops     []*agro.Crop
	Templates []agro.Template
}

func Load() (*Catalog, error) {
	return Parse([]byte(cropsJSON))
}

func Parse(data []byte) (*Catalog, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var doc Document
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode catalog: %w", errors.Join(ErrInvalidCatalog, err))
	}

	if err := Validate(doc); err != nil {
		return nil, err
	}

	sum := sha256.Sum256(data)
	c := &Catalog{Version: doc.Version, Hash: hex.EncodeToString(sum[:6]), Document: doc}

	for _, e := range doc.Crops {
		c.Crops = append(c.Crops, toCrop(e))
	}

	for _, t := range doc.Templates {
		c.Templates = append(c.Templates, agro.Template{ID: t.ID, Name: t.Name, Crops: t.Crops, Relay: t.Relay, Mulch: t.Mulch})
	}

	return c, nil
}

func toCrop(e CropEntry) *agro.Crop {
	t := e.Temperature.Value
	c := &agro.Crop{
		ID: e.ID, Name: e.Name, Family: e.Family, Role: roles[e.Role],
		StageDays: e.Stages.Value,
		KcIni:     e.Kc.Value[0], KcMid: e.Kc.Value[1], KcEnd: e.Kc.Value[2],
		MaxRootDepth: agro.Meters(e.RootDepthM.Value), DepletionFraction: agro.Fraction(e.Depletion.Value),
		KyTotal: e.KyTotal.Value,
		Temperature: agro.TemperatureRange{
			Base: agro.Celsius(t[0]), AbsoluteLo: agro.Celsius(t[1]), OptimalLo: agro.Celsius(t[2]),
			OptimalHi: agro.Celsius(t[3]), AbsoluteHi: agro.Celsius(t[4]),
		},
		HeatStressTMax: agro.Celsius(e.HeatStressTMax.Value),
		PH:             agro.PHRange{AbsoluteLo: e.PH.Value[0], OptimalLo: e.PH.Value[1], OptimalHi: e.PH.Value[2], AbsoluteHi: e.PH.Value[3]},
		AltitudeMin:    agro.Meters(e.AltitudeM.Value[0]), AltitudeMax: agro.Meters(e.AltitudeM.Value[1]),
		Irrigation:              irrigationNeeds[e.Irrigation.Value],
		Waterlogging:            waterlogging[e.Waterlogging.Value],
		NitrogenFixationKgHa:    e.Nitrogen.Value[0],
		NitrogenRemovalKgPerTon: e.Nitrogen.Value[1],
		ReferenceYieldTonHa:     e.ReferenceYield.Value,
		CoverFactorByStage:      e.CoverFactor.Value,
		LowConfidenceFields:     lowConfidence(e),
	}

	if e.KyByPhase != nil {
		c.KyByPhase, c.HasKyByPhase = e.KyByPhase.Value, true
	}

	if e.Relay != nil {
		c.RelayGuest = relayKinds[e.Relay.Value.Guest]
		for _, h := range e.Relay.Value.Hosts {
			c.RelayHosts = append(c.RelayHosts, relayKinds[h])
		}
	}

	return c
}

type RelaySpec struct {
	Guest string   `json:"guest,omitempty"`
	Hosts []string `json:"hosts,omitempty"`
}

var relayKinds = map[string]agro.RelayKind{"cover": agro.RelayCover, "grain": agro.RelayGrain}

var roles = map[string]agro.CropRole{"harvest": agro.RoleHarvest, "cover": agro.RoleCover}

var irrigationNeeds = map[string]agro.IrrigationNeed{"optional": agro.IrrigationOptional, "required": agro.IrrigationRequired}

var waterlogging = map[string]agro.WaterloggingTolerance{
	"sensitive": agro.WaterloggingSensitive, "moderate": agro.WaterloggingModerate, "tolerant": agro.WaterloggingTolerant,
}

var criticalFields = map[string]bool{"stages": true, "kc": true, "rootDepthM": true, "depletion": true, "kyTotal": true, "temperatureC": true}

func lowConfidence(e CropEntry) []string {
	var out []string

	for name, conf := range confidences(e) {
		if conf == ConfidenceC && criticalFields[name] {
			out = append(out, name)
		}
	}

	sortStrings(out)

	return out
}

func confidences(e CropEntry) map[string]Confidence {
	out := map[string]Confidence{
		"stages": e.Stages.Confidence, "kc": e.Kc.Confidence, "rootDepthM": e.RootDepthM.Confidence,
		"depletion": e.Depletion.Confidence, "kyTotal": e.KyTotal.Confidence, "temperatureC": e.Temperature.Confidence,
		"heatStressTMaxC": e.HeatStressTMax.Confidence, "ph": e.PH.Confidence, "altitudeM": e.AltitudeM.Confidence,
		"irrigation": e.Irrigation.Confidence, "waterlogging": e.Waterlogging.Confidence, "nitrogenKgHa": e.Nitrogen.Confidence,
		"referenceYieldTHa": e.ReferenceYield.Confidence, "coverFactor": e.CoverFactor.Confidence,
	}

	if e.KyByPhase != nil {
		out["kyByPhase"] = e.KyByPhase.Confidence
	}

	if e.Relay != nil {
		out["relay"] = e.Relay.Confidence
	}

	return out
}

func sources(e CropEntry) map[string]string {
	out := map[string]string{
		"stages": e.Stages.Source, "kc": e.Kc.Source, "rootDepthM": e.RootDepthM.Source, "depletion": e.Depletion.Source,
		"kyTotal": e.KyTotal.Source, "temperatureC": e.Temperature.Source, "heatStressTMaxC": e.HeatStressTMax.Source,
		"ph": e.PH.Source, "altitudeM": e.AltitudeM.Source, "irrigation": e.Irrigation.Source, "waterlogging": e.Waterlogging.Source,
		"nitrogenKgHa": e.Nitrogen.Source, "referenceYieldTHa": e.ReferenceYield.Source, "coverFactor": e.CoverFactor.Source,
	}

	if e.KyByPhase != nil {
		out["kyByPhase"] = e.KyByPhase.Source
	}

	if e.Relay != nil {
		out["relay"] = e.Relay.Source
	}

	return out
}
