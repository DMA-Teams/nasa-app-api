package geo

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/paulmach/orb"
)

var ErrInvalidBoundaries = errors.New("invalid boundaries document")

type Geometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type Feature struct {
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties"`
	Geometry   Geometry          `json:"geometry"`
}

type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

type boundariesDocument struct {
	Departments    []Department      `json:"departments"`
	Municipalities FeatureCollection `json:"municipalities"`
}

func (g Geometry) MultiPolygon() (orb.MultiPolygon, error) {
	switch g.Type {
	case "Polygon":
		var p [][][2]float64
		if err := json.Unmarshal(g.Coordinates, &p); err != nil {
			return nil, fmt.Errorf("polygon coordinates: %w", err)
		}

		return orb.MultiPolygon{toPolygon(p)}, nil
	case "MultiPolygon":
		var mp [][][][2]float64
		if err := json.Unmarshal(g.Coordinates, &mp); err != nil {
			return nil, fmt.Errorf("multipolygon coordinates: %w", err)
		}

		out := make(orb.MultiPolygon, len(mp))
		for i, p := range mp {
			out[i] = toPolygon(p)
		}

		return out, nil
	default:
		return nil, fmt.Errorf("geometry type %q: %w", g.Type, ErrInvalidBoundaries)
	}
}

func toPolygon(rings [][][2]float64) orb.Polygon {
	p := make(orb.Polygon, len(rings))

	for i, r := range rings {
		ring := make(orb.Ring, len(r))
		for j, pt := range r {
			ring[j] = orb.Point{pt[0], pt[1]}
		}

		p[i] = ring
	}

	return p
}

func multiPolygonGeometry(mp orb.MultiPolygon) (Geometry, error) {
	coords := make([][][][2]float64, len(mp))

	for i, p := range mp {
		coords[i] = make([][][2]float64, len(p))
		for j, r := range p {
			coords[i][j] = make([][2]float64, len(r))
			for k, pt := range r {
				coords[i][j][k] = [2]float64{pt[0], pt[1]}
			}
		}
	}

	raw, err := json.Marshal(coords)
	if err != nil {
		return Geometry{}, fmt.Errorf("encode coordinates: %w", err)
	}

	return Geometry{Type: "MultiPolygon", Coordinates: raw}, nil
}

func EncodeBoundaries(municipalities []Municipality, departments []Department) ([]byte, error) {
	fc := FeatureCollection{Type: "FeatureCollection", Features: make([]Feature, 0, len(municipalities))}

	for _, m := range municipalities {
		g, err := multiPolygonGeometry(m.Geometry)
		if err != nil {
			return nil, err
		}

		fc.Features = append(fc.Features, Feature{
			Type:       "Feature",
			Properties: map[string]string{"code": m.Code, "name": m.Name, "departmentCode": m.DepartmentCode},
			Geometry:   g,
		})
	}

	data, err := json.Marshal(boundariesDocument{Departments: departments, Municipalities: fc})
	if err != nil {
		return nil, fmt.Errorf("encode boundaries: %w", err)
	}

	return data, nil
}

func DecodeBoundaries(data []byte) (*Index, error) {
	var doc boundariesDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode boundaries: %w", errors.Join(ErrInvalidBoundaries, err))
	}

	if len(doc.Municipalities.Features) == 0 || len(doc.Departments) == 0 {
		return nil, fmt.Errorf("boundaries without municipalities or departments: %w", ErrInvalidBoundaries)
	}

	municipalities := make([]Municipality, 0, len(doc.Municipalities.Features))

	for _, f := range doc.Municipalities.Features {
		mp, err := f.Geometry.MultiPolygon()
		if err != nil {
			return nil, fmt.Errorf("municipality %s: %w", f.Properties["code"], err)
		}

		municipalities = append(municipalities, Municipality{
			Code: f.Properties["code"], Name: f.Properties["name"], DepartmentCode: f.Properties["departmentCode"], Geometry: mp,
		})
	}

	return NewIndex(municipalities, doc.Departments), nil
}

func ParseFeatureCollection(data []byte) (FeatureCollection, error) {
	var raw struct {
		Features []struct {
			Properties map[string]any `json:"properties"`
			Geometry   Geometry       `json:"geometry"`
		} `json:"features"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return FeatureCollection{}, fmt.Errorf("decode feature collection: %w", errors.Join(ErrInvalidBoundaries, err))
	}

	fc := FeatureCollection{Type: "FeatureCollection", Features: make([]Feature, 0, len(raw.Features))}

	for _, f := range raw.Features {
		props := make(map[string]string, len(f.Properties))
		for k, v := range f.Properties {
			if s, ok := v.(string); ok {
				props[k] = s
			}
		}

		fc.Features = append(fc.Features, Feature{Type: "Feature", Properties: props, Geometry: f.Geometry})
	}

	return fc, nil
}
