package etl

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"

	"github.com/dma-teams/nasa-app-api/internal/geo"
)

func BuildBoundaries(adm1, adm2 geo.FeatureCollection) ([]geo.Municipality, []geo.Department, error) {
	departments := make([]geo.Department, 0, len(adm1.Features))
	depGeoms := make(map[string]orb.MultiPolygon, len(adm1.Features))

	for _, f := range adm1.Features {
		mp, err := f.Geometry.MultiPolygon()
		if err != nil {
			return nil, nil, fmt.Errorf("department %s: %w", f.Properties["shapeName"], err)
		}

		code := cmp.Or(f.Properties["shapeISO"], f.Properties["shapeID"])
		departments = append(departments, geo.Department{Code: code, Name: f.Properties["shapeName"]})
		depGeoms[code] = mp
	}

	municipalities := make([]geo.Municipality, 0, len(adm2.Features))

	for _, f := range adm2.Features {
		mp, err := f.Geometry.MultiPolygon()
		if err != nil {
			return nil, nil, fmt.Errorf("municipality %s: %w", f.Properties["shapeName"], err)
		}

		depCode := departmentOf(geo.InteriorPoint(mp), depGeoms)
		if depCode == "" {
			return nil, nil, fmt.Errorf("municipality %s is outside every department: %w", f.Properties["shapeName"], geo.ErrInvalidBoundaries)
		}

		municipalities = append(municipalities, geo.Municipality{
			Code: f.Properties["shapeID"], Name: f.Properties["shapeName"], DepartmentCode: depCode, Geometry: mp,
		})
	}

	slices.SortFunc(departments, func(a, b geo.Department) int { return cmp.Compare(a.Code, b.Code) })

	return municipalities, departments, nil
}

func departmentOf(p orb.Point, departments map[string]orb.MultiPolygon) string {
	for code, mp := range departments {
		if mp.Bound().Contains(p) && planar.MultiPolygonContains(mp, p) {
			return code
		}
	}

	return ""
}
