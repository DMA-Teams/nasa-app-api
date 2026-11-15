package catalog

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/dma-teams/nasa-app-api/internal/agro"
)

func TestLoad_EmbeddedCatalogIsValid(t *testing.T) {
	t.Parallel()

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(c.Crops) < 20 || len(c.Templates) != 4 || c.Hash == "" {
		t.Fatalf("crops=%d templates=%d hash=%q, want >= 20 crops, 4 templates and a hash", len(c.Crops), len(c.Templates), c.Hash)
	}

	if _, err := agro.NewEngine(c.Crops, c.Templates); err != nil {
		t.Fatalf("NewEngine with catalog: %v", err)
	}
}

func TestLoad_LowConfidenceFieldsAreFlagged(t *testing.T) {
	t.Parallel()

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, crop := range c.Crops {
		if crop.ID == "mucuna" && len(crop.LowConfidenceFields) == 0 {
			t.Error("mucuna uses proxy values and must report low-confidence fields")
		}

		if crop.ID == "maize" && !slices.Equal(crop.LowConfidenceFields, []string{"stages", "temperatureC"}) {
			t.Errorf("maize low-confidence fields = %v, want the provisional stages and the temperature range", crop.LowConfidenceFields)
		}
	}
}

func TestParse_RejectsMissingSource(t *testing.T) {
	t.Parallel()

	data := strings.Replace(cropsJSON, `"source": "FAO-56 Tabla 22"`, `"source": ""`, 1)

	_, err := Parse([]byte(data))
	if !errors.Is(err, ErrInvalidCatalog) || !strings.Contains(err.Error(), "has no source") {
		t.Errorf("err = %v, want missing source error", err)
	}
}

func TestParse_RejectsUnknownFields(t *testing.T) {
	t.Parallel()

	data := strings.Replace(cropsJSON, `"version"`, `"unexpected": 1, "version"`, 1)

	if _, err := Parse([]byte(data)); !errors.Is(err, ErrInvalidCatalog) {
		t.Errorf("err = %v, want ErrInvalidCatalog", err)
	}
}

func TestParse_RejectsUnknownTemplateCrop(t *testing.T) {
	t.Parallel()

	data := strings.Replace(cropsJSON, `"crops": [
        "maize",
        "sorghum"`, `"crops": [
        "maize",
        "teff"`, 1)

	if _, err := Parse([]byte(data)); !errors.Is(err, ErrInvalidCatalog) {
		t.Errorf("err = %v, want ErrInvalidCatalog", err)
	}
}
