package thermo

import (
	"fmt"
	"math"
	"strings"
)

// SaltModel identifies how solution ions are reduced to the monovalent value
// consumed by the current nearest-neighbor entropy correction.
type SaltModel string

const (
	// SaltModelMonovalent uses only the supplied monovalent cation concentration.
	SaltModelMonovalent SaltModel = "monovalent"

	// SaltModelOwczarzyLite applies the historical Mg-to-Na-equivalent heuristic:
	// Na_eff = Na + 3.8*sqrt(Mg). This is an approximation, not a full mixed-salt model.
	SaltModelOwczarzyLite SaltModel = "owczarzy-lite"

	// SaltModelOwczarzy08 applies the mixed monovalent/divalent salt correction
	// form from Owczarzy et al. 2008 using free Mg after dNTP chelation.
	SaltModelOwczarzy08 SaltModel = "owczarzy08"
)

// Conditions collects wet-lab inputs used by thermodynamic calculations.
type Conditions struct {
	AnnealC           float64
	NaM               float64
	MgM               float64
	DntpM             float64
	PrimerTotalM      float64
	SaltModel         SaltModel
	SelfComplementary bool
}

// DefaultConditions returns the ipcr-thermo CLI defaults in mol/L.
func DefaultConditions() Conditions {
	return Conditions{
		AnnealC:      60,
		NaM:          0.05,
		MgM:          0.003,
		DntpM:        0,
		PrimerTotalM: 2.5e-7,
		SaltModel:    SaltModelMonovalent,
	}
}

func (m SaltModel) String() string {
	if m == "" {
		return string(SaltModelMonovalent)
	}
	return string(m)
}

// ParseSaltModel validates and normalizes a salt model name.
func ParseSaltModel(raw string) (SaltModel, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return SaltModelMonovalent, nil
	}
	switch SaltModel(s) {
	case SaltModelMonovalent, SaltModelOwczarzyLite, SaltModelOwczarzy08:
		return SaltModel(s), nil
	default:
		return "", fmt.Errorf("unknown salt model %q; expected one of: %s", raw, KnownSaltModels())
	}
}

// KnownSaltModels returns CLI help text for salt model choices.
func KnownSaltModels() string {
	return strings.Join([]string{SaltModelMonovalent.String(), SaltModelOwczarzyLite.String(), SaltModelOwczarzy08.String()}, " | ")
}

// WithDefaults fills missing condition fields with CLI defaults. MgM is not
// defaulted here because zero magnesium is a valid explicit experimental state;
// callers that want the CLI default should start from DefaultConditions().
func (c Conditions) WithDefaults() Conditions {
	d := DefaultConditions()
	if c.AnnealC == 0 {
		c.AnnealC = d.AnnealC
	}
	if c.NaM == 0 {
		c.NaM = d.NaM
	}
	if c.PrimerTotalM == 0 {
		c.PrimerTotalM = d.PrimerTotalM
	}
	if c.SaltModel == "" {
		c.SaltModel = d.SaltModel
	}
	return c
}

// EffectiveNaM returns the monovalent concentration consumed by the current NN
// salt correction under the selected salt model.
func (c Conditions) EffectiveNaM() float64 {
	c = c.WithDefaults()
	return EffectiveMonovalent(c.NaM, c.MgM, c.DntpM, c.SaltModel)
}

// FreeMgM returns free Mg2+ after a simple dNTP chelation approximation. The
// input DntpM is interpreted as total dNTP concentration in mol/L.
func (c Conditions) FreeMgM() float64 {
	c = c.WithDefaults()
	return FreeMagnesium(c.MgM, c.DntpM)
}

// TmInput builds the core nearest-neighbor Tm input from these conditions.
func (c Conditions) TmInput() TmInput {
	c = c.WithDefaults()
	x := 4
	if c.SelfComplementary {
		x = 1
	}
	return TmInput{
		CT:        c.PrimerTotalM,
		Na:        c.EffectiveNaM(),
		Mg:        c.MgM,
		Dntp:      c.DntpM,
		SaltModel: c.SaltModel,
		X:         x,
	}
}

// ParseConc parses common molar strings such as "50mM", "250nM", "3uM",
// "3µM" (micro sign), and "3μM" (Greek mu) into mol/L.
func ParseConc(s string) (float64, error) {
	raw := strings.TrimSpace(s)
	norm := strings.ToLower(raw)
	norm = strings.ReplaceAll(norm, "µ", "u")
	norm = strings.ReplaceAll(norm, "μ", "u")

	unit := ""
	val := 0.0
	_, err := fmt.Sscanf(norm, "%f%s", &val, &unit)
	if err != nil {
		return 0, fmt.Errorf("invalid conc %q: %w", raw, err)
	}
	if val < 0 {
		return 0, fmt.Errorf("invalid conc %q: concentration must be non-negative", raw)
	}
	switch unit {
	case "m", "":
		return val, nil
	case "mm":
		return val * 1e-3, nil
	case "um":
		return val * 1e-6, nil
	case "nm":
		return val * 1e-9, nil
	default:
		return 0, fmt.Errorf("unknown unit %q in %q", unit, raw)
	}
}

// EffectiveMonovalent returns the effective monovalent concentration under an
// explicit salt model.
func EffectiveMonovalent(naM, mgM, dntpM float64, model SaltModel) float64 {
	if model == "" {
		model = SaltModelMonovalent
	}
	if model == SaltModelOwczarzyLite && mgM > 0 {
		return naM + 3.8*math.Sqrt(FreeMagnesium(mgM, dntpM))
	}
	return naM
}

// FreeMagnesium returns free Mg2+ after dNTP chelation. The equilibrium form
// follows the common Owczarzy/Primer3-style Mg:dNTP association approximation
// with Ka=3e4 M^-1.
func FreeMagnesium(mgM, dntpM float64) float64 {
	if mgM <= 0 {
		return 0
	}
	if dntpM <= 0 {
		return mgM
	}
	const ka = 3e4
	b := ka*dntpM - ka*mgM + 1.0
	disc := b*b + 4.0*ka*mgM
	if disc < 0 || math.IsNaN(disc) || math.IsInf(disc, 0) {
		return 0
	}
	free := (-b + math.Sqrt(disc)) / (2.0 * ka)
	if free < 0 || math.IsNaN(free) || math.IsInf(free, 0) {
		return 0
	}
	return free
}
