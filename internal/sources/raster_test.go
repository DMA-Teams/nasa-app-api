package sources

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/paulmach/orb"
)

func writeENVI(t *testing.T, dir string, width, height int, values []int16) string {
	t.Helper()

	header := "ENVI\nsamples = 4\nlines = 3\nbands = 1\ndata type = 2\nbyte order = 0\n" +
		"map info = {Geographic Lat/Lon, 1, 1, -89.0, 15.0, 0.25, 0.25, WGS-84}\ndata ignore value = -32768\n"
	if err := os.WriteFile(filepath.Join(dir, "clay.hdr"), []byte(header), 0o600); err != nil {
		t.Fatal(err)
	}

	data := make([]byte, 0, width*height*2)
	for _, v := range values {
		data = binary.LittleEndian.AppendUint16(data, uint16(v))
	}

	path := filepath.Join(dir, "clay.bin")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestReadENVI_ValuesInPolygon(t *testing.T) {
	t.Parallel()

	path := writeENVI(t, t.TempDir(), 4, 3, []int16{
		100, 200, 300, 400,
		500, -32768, 700, 800,
		900, 1000, 1100, 1200,
	})

	r, err := ReadENVI(path)
	if err != nil {
		t.Fatalf("ReadENVI: %v", err)
	}

	poly := orb.MultiPolygon{{{{-89.0, 14.5}, {-88.5, 14.5}, {-88.5, 15.0}, {-89.0, 15.0}, {-89.0, 14.5}}}}
	got := r.ValuesIn(poly, 10)

	if len(got) != 3 {
		t.Fatalf("values = %v, want 3 pixels (one is nodata)", got)
	}

	if got[0] != 10 || got[1] != 20 || got[2] != 50 {
		t.Errorf("values = %v, want [10 20 50]", got)
	}
}
