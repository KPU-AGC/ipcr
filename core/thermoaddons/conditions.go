// core/thermoaddons/conditions.go
package thermoaddons

import (
	"os"
	"strings"

	"ipcr-core/thermo"
)

// Conditions is kept as a compatibility alias. New thermodynamic code should
// use thermo.Conditions directly.
type Conditions = thermo.Conditions

// ParseConc delegates to the canonical concentration parser in core/thermo.
func ParseConc(s string) (float64, error) { return thermo.ParseConc(s) }

// EffectiveMonovalent is a legacy compatibility wrapper. New code should choose
// an explicit thermo.SaltModel and call thermo.EffectiveMonovalent instead.
func EffectiveMonovalent(naM, mgM float64) float64 {
	model := thermo.SaltModelMonovalent
	mode := strings.TrimSpace(strings.ToLower(os.Getenv("IPCR_MG_EQ")))
	if mode == thermo.SaltModelOwczarzyLite.String() {
		model = thermo.SaltModelOwczarzyLite
	}
	return thermo.EffectiveMonovalent(naM, mgM, 0, model)
}
