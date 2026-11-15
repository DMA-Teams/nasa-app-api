package catalog

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

func Validate(doc Document) error {
	var errs []error

	if strings.TrimSpace(doc.Version) == "" {
		errs = append(errs, errors.New("catalog version is required"))
	}

	ids := map[string]bool{}

	for _, c := range doc.Crops {
		if ids[c.ID] {
			errs = append(errs, fmt.Errorf("crop %s: duplicated id", c.ID))
		}

		ids[c.ID] = true
		errs = append(errs, validateCrop(c)...)
	}

	for _, t := range doc.Templates {
		errs = append(errs, validateTemplate(t, ids)...)
	}

	if err := errors.Join(errs...); err != nil {
		return errors.Join(ErrInvalidCatalog, err)
	}

	return nil
}

func validateCrop(c CropEntry) []error {
	var errs []error

	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("crop %s: "+format, append([]any{c.ID}, args...)...))
	}

	if c.ID == "" || c.Name == "" || c.Family == "" {
		fail("id, name and family are required")
	}

	if _, ok := roles[c.Role]; !ok {
		fail("unknown role %q", c.Role)
	}

	validateProvenance(c, fail)
	validateRanges(c, fail)

	return errs
}

func validateProvenance(c CropEntry, fail func(string, ...any)) {
	for name, src := range sources(c) {
		if strings.TrimSpace(src) == "" {
			fail("field %s has no source", name)
		}
	}

	for name, conf := range confidences(c) {
		if conf != ConfidenceA && conf != ConfidenceB && conf != ConfidenceC {
			fail("field %s has invalid confidence %q", name, conf)
		}
	}
}

func validateRanges(c CropEntry, fail func(string, ...any)) {
	for _, d := range c.Stages.Value {
		if d <= 0 {
			fail("stage lengths must be positive")
		}
	}

	if c.Kc.Value[0] <= 0 || c.Kc.Value[1] <= 0 || c.Kc.Value[2] <= 0 {
		fail("kc values must be positive")
	}

	if c.RootDepthM.Value <= 0 || c.Depletion.Value <= 0 || c.Depletion.Value >= 1 {
		fail("root depth must be positive and depletion within (0, 1)")
	}

	if !ascending(c.Temperature.Value[1:]) || !ascending(c.PH.Value[:]) || !ascending(c.AltitudeM.Value[:]) {
		fail("temperature, pH and altitude ranges must be ascending")
	}

	if _, ok := irrigationNeeds[c.Irrigation.Value]; !ok {
		fail("unknown irrigation need %q", c.Irrigation.Value)
	}

	if _, ok := waterlogging[c.Waterlogging.Value]; !ok {
		fail("unknown waterlogging tolerance %q", c.Waterlogging.Value)
	}

	for _, f := range c.CoverFactor.Value {
		if f < 0 || f > 1 {
			fail("cover factors must be within [0, 1]")
		}
	}

	if c.Relay != nil {
		validateRelay(c.Relay.Value, fail)
	}
}

func validateRelay(r RelaySpec, fail func(string, ...any)) {
	if _, ok := relayKinds[r.Guest]; r.Guest != "" && !ok {
		fail("unknown relay guest kind %q", r.Guest)
	}

	for _, h := range r.Hosts {
		if _, ok := relayKinds[h]; !ok {
			fail("unknown relay host kind %q", h)
		}
	}

	if r.Guest == "" && len(r.Hosts) == 0 {
		fail("relay needs a guest kind or host kinds")
	}
}

func validateTemplate(t TemplateEntry, crops map[string]bool) []error {
	var errs []error

	if len(t.Crops) == 0 || len(t.Relay) != len(t.Crops) {
		errs = append(errs, fmt.Errorf("template %s: needs crops and one relay flag per crop", t.ID))
	}

	for _, id := range t.Crops {
		if !crops[id] {
			errs = append(errs, fmt.Errorf("template %s: unknown crop %s", t.ID, id))
		}
	}

	return errs
}

func ascending(values []float64) bool {
	return slices.IsSorted(values)
}

func sortStrings(s []string) {
	slices.Sort(s)
}
