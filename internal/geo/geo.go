package geo

import (
	"errors"
	"slices"
	"strings"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
)

var (
	ErrOutsideHonduras = errors.New("location outside Honduras")
	ErrNotFound        = errors.New("administrative unit not found")
)

var HondurasBound = orb.Bound{Min: orb.Point{-89.40, 12.95}, Max: orb.Point{-83.10, 16.55}}

type Department struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Municipality struct {
	Code           string           `json:"code"`
	Name           string           `json:"name"`
	DepartmentCode string           `json:"departmentCode"`
	Geometry       orb.MultiPolygon `json:"-"`
	Bound          orb.Bound        `json:"-"`
}

type Index struct {
	municipalities []Municipality
	byCode         map[string]int
	departments    []Department
	byDepartment   map[string][]string
	depByCode      map[string]Department
	unavailable    map[string]bool
}

func NewIndex(municipalities []Municipality, departments []Department) *Index {
	ix := &Index{
		municipalities: slices.Clone(municipalities),
		byCode:         make(map[string]int, len(municipalities)),
		departments:    slices.Clone(departments),
		byDepartment:   map[string][]string{},
		depByCode:      make(map[string]Department, len(departments)),
	}

	slices.SortFunc(ix.municipalities, func(a, b Municipality) int { return strings.Compare(a.Code, b.Code) })
	slices.SortFunc(ix.departments, func(a, b Department) int { return strings.Compare(a.Code, b.Code) })

	for i := range ix.municipalities {
		m := &ix.municipalities[i]
		if m.Bound.IsZero() {
			m.Bound = m.Geometry.Bound()
		}

		ix.byCode[m.Code] = i
		ix.byDepartment[m.DepartmentCode] = append(ix.byDepartment[m.DepartmentCode], m.Code)
	}

	for _, d := range ix.departments {
		ix.depByCode[d.Code] = d
	}

	return ix
}

func InHondurasBound(lat, lon float64) bool {
	return HondurasBound.Contains(orb.Point{lon, lat})
}

func (ix *Index) MunicipalityAt(lat, lon float64) (Municipality, error) {
	if !InHondurasBound(lat, lon) {
		return Municipality{}, ErrOutsideHonduras
	}

	p := orb.Point{lon, lat}
	for _, m := range ix.municipalities {
		if m.Bound.Contains(p) && planar.MultiPolygonContains(m.Geometry, p) {
			return m, nil
		}
	}

	return Municipality{}, ErrOutsideHonduras
}

func (ix *Index) Municipality(code string) (Municipality, error) {
	i, ok := ix.byCode[code]
	if !ok {
		return Municipality{}, ErrNotFound
	}

	return ix.municipalities[i], nil
}

func (ix *Index) Department(code string) (Department, error) {
	d, ok := ix.depByCode[code]
	if !ok {
		return Department{}, ErrNotFound
	}

	return d, nil
}

func (ix *Index) WithUnavailable(codes []string) *Index {
	c := *ix
	c.unavailable = make(map[string]bool, len(codes))

	for _, code := range codes {
		c.unavailable[code] = true
	}

	return &c
}

func (ix *Index) Departments() []Department {
	return slices.DeleteFunc(slices.Clone(ix.departments), func(d Department) bool {
		return !slices.ContainsFunc(ix.byDepartment[d.Code], ix.available)
	})
}

func (ix *Index) Municipalities() []Municipality {
	return slices.DeleteFunc(slices.Clone(ix.municipalities), func(m Municipality) bool {
		return !ix.available(m.Code)
	})
}

func (ix *Index) MunicipalityCodesOf(departmentCode string) ([]string, error) {
	codes, ok := ix.byDepartment[departmentCode]
	if !ok {
		return nil, ErrNotFound
	}

	return slices.DeleteFunc(slices.Clone(codes), func(code string) bool { return !ix.available(code) }), nil
}

func (ix *Index) UnavailableCountOf(departmentCode string) int {
	n := 0

	for _, code := range ix.byDepartment[departmentCode] {
		if !ix.available(code) {
			n++
		}
	}

	return n
}

func (ix *Index) available(code string) bool {
	return !ix.unavailable[code]
}
