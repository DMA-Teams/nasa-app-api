package geo

import (
	"errors"
	"math"
	"testing"

	"github.com/paulmach/orb"
)

func square(lon, lat, size float64) orb.MultiPolygon {
	return orb.MultiPolygon{{{
		{lon, lat}, {lon + size, lat}, {lon + size, lat + size}, {lon, lat + size}, {lon, lat},
	}}}
}

func testIndex() *Index {
	return NewIndex(
		[]Municipality{
			{Code: "M1", Name: "Uno", DepartmentCode: "D1", Geometry: square(-88.7, 14.2, 0.2)},
			{Code: "M2", Name: "Dos", DepartmentCode: "D1", Geometry: square(-88.5, 14.2, 0.2)},
			{Code: "M3", Name: "Tres", DepartmentCode: "D2", Geometry: square(-87.0, 15.5, 0.2)},
		},
		[]Department{{Code: "D1", Name: "Uno"}, {Code: "D2", Name: "Dos"}},
	)
}

func TestPowerCellOf_MatchesObservedPowerGrid(t *testing.T) {
	t.Parallel()

	a, b, c := PowerCellOf(14.30, -88.60), PowerCellOf(14.45, -88.60), PowerCellOf(14.30, -88.40)

	if a != b {
		t.Errorf("14.30 and 14.45 fall in %v and %v, want the same cell", a, b)
	}

	if a == c {
		t.Errorf("-88.60 and -88.40 fall in the same cell %v, want neighbors", a)
	}

	if a.Lat != 14.5 || math.Abs(a.Lon-(-88.75)) > 1e-9 {
		t.Errorf("cell = %v, want 14.5, -88.75", a)
	}
}

func TestIndex_MunicipalityAt(t *testing.T) {
	t.Parallel()

	ix := testIndex()

	m, err := ix.MunicipalityAt(14.3, -88.6)
	if err != nil || m.Code != "M1" {
		t.Errorf("MunicipalityAt = %v, %v; want M1", m.Code, err)
	}

	if _, err := ix.MunicipalityAt(40.4, -3.7); !errors.Is(err, ErrOutsideHonduras) {
		t.Errorf("Madrid err = %v, want ErrOutsideHonduras", err)
	}

	codes, err := ix.MunicipalityCodesOf("D1")
	if err != nil || len(codes) != 2 {
		t.Errorf("MunicipalityCodesOf(D1) = %v, %v; want two codes", codes, err)
	}
}

func TestIndexWithUnavailable_HidesMunicipalitiesWithoutData(t *testing.T) {
	t.Parallel()

	ix := NewIndex(
		[]Municipality{
			{Code: "M1", Name: "Uno", DepartmentCode: "D1", Geometry: square(-88.7, 14.2, 0.2)},
			{Code: "M2", Name: "Dos", DepartmentCode: "D1", Geometry: square(-88.5, 14.2, 0.2)},
			{Code: "M3", Name: "Tres", DepartmentCode: "D2", Geometry: square(-87.5, 14.2, 0.2)},
		},
		[]Department{{Code: "D1", Name: "Uno"}, {Code: "D2", Name: "Dos"}},
	).WithUnavailable([]string{"M2", "M3"})

	if got := ix.Municipalities(); len(got) != 1 || got[0].Code != "M1" {
		t.Errorf("Municipalities = %v, want only M1", got)
	}

	if got := ix.Departments(); len(got) != 1 || got[0].Code != "D1" {
		t.Errorf("Departments = %v, want only D1", got)
	}

	codes, err := ix.MunicipalityCodesOf("D1")
	if err != nil || len(codes) != 1 || codes[0] != "M1" {
		t.Errorf("MunicipalityCodesOf(D1) = %v, %v; want [M1]", codes, err)
	}

	if n := ix.UnavailableCountOf("D1"); n != 1 {
		t.Errorf("UnavailableCountOf(D1) = %d, want 1", n)
	}

	if m, err := ix.Municipality("M2"); err != nil || m.Code != "M2" {
		t.Errorf("Municipality(M2) = %v, %v; lookup by code must still work", m.Code, err)
	}

	if m, err := ix.MunicipalityAt(14.3, -88.4); err != nil || m.Code != "M2" {
		t.Errorf("MunicipalityAt inside M2 = %v, %v; lookup by location must still work", m.Code, err)
	}
}

func TestCellShares_SumToOneAndSplitAcrossCells(t *testing.T) {
	t.Parallel()

	shares := CellShares(square(-88.5, 14.0, 0.5))
	if len(shares) < 2 {
		t.Fatalf("shares = %v, want the square split across cells", shares)
	}

	sum := 0.0
	for _, s := range shares {
		sum += s.Share
	}

	if math.Abs(sum-1) > 1e-9 {
		t.Errorf("shares sum = %.6f, want 1", sum)
	}
}

func TestSamplePoints_ReturnsPointsInsidePolygon(t *testing.T) {
	t.Parallel()

	mp := square(-88.7, 14.2, 0.2)
	points := SamplePoints(mp, 0.045, 9)

	if len(points) < 9 {
		t.Fatalf("got %d points, want >= 9", len(points))
	}

	tiny := square(-88.7, 14.2, 0.001)
	if got := SamplePoints(tiny, 0.045, 9); len(got) == 0 {
		t.Error("tiny polygon must still produce a sample point")
	}
}

func TestBoundariesRoundTrip(t *testing.T) {
	t.Parallel()

	ix := testIndex()

	data, err := EncodeBoundaries(ix.Municipalities(), ix.Departments())
	if err != nil {
		t.Fatalf("EncodeBoundaries: %v", err)
	}

	back, err := DecodeBoundaries(data)
	if err != nil {
		t.Fatalf("DecodeBoundaries: %v", err)
	}

	m, err := back.MunicipalityAt(15.6, -86.9)
	if err != nil || m.Code != "M3" || m.DepartmentCode != "D2" {
		t.Errorf("decoded lookup = %+v, %v; want M3 in D2", m, err)
	}
}
