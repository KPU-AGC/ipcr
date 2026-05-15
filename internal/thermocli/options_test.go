package thermocli

import (
	"flag"
	"io"
	"ipcr-core/thermo"
	"ipcr/internal/thermomodel"
	"strings"
	"testing"
)

func parseArgsForTest(args ...string) (Options, error) {
	fs := NewFlagSet("ipcr-thermo")
	fs.SetOutput(io.Discard)
	return ParseArgs(fs, args)
}

func minimalArgs() []string {
	return []string{"--forward", "ACGT", "--reverse", "ACGT", "--sequences", "ref.fa"}
}

func TestParseArgs_DefaultThermoModelIsLegacyHeuristic(t *testing.T) {
	opts, err := parseArgsForTest(minimalArgs()...)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.ThermoModel != thermomodel.LegacyHeuristic.String() {
		t.Fatalf("got model %q, want %q", opts.ThermoModel, thermomodel.LegacyHeuristic)
	}
}

func TestParseArgs_ExplicitLegacyThermoModel(t *testing.T) {
	args := append(minimalArgs(), "--thermo-model", "legacy-heuristic")
	opts, err := parseArgsForTest(args...)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.ThermoModel != thermomodel.LegacyHeuristic.String() {
		t.Fatalf("got model %q, want %q", opts.ThermoModel, thermomodel.LegacyHeuristic)
	}
}

func TestParseArgs_UnknownThermoModelRejected(t *testing.T) {
	args := append(minimalArgs(), "--thermo-model", "bogus")
	_, err := parseArgsForTest(args...)
	if err == nil {
		t.Fatal("expected unknown model error")
	}
	if !strings.Contains(err.Error(), "unknown thermo model") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseArgs_NNDuplexThermoModelAccepted(t *testing.T) {
	args := append(minimalArgs(), "--thermo-model", thermomodel.NNDuplexV1.String())
	opts, err := parseArgsForTest(args...)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.ThermoModel != thermomodel.NNDuplexV1.String() {
		t.Fatalf("got model %q, want %q", opts.ThermoModel, thermomodel.NNDuplexV1)
	}
}

func TestParseArgs_NNStructureThermoModelAccepted(t *testing.T) {
	args := append(minimalArgs(), "--thermo-model", thermomodel.NNStructureV1.String())
	opts, err := parseArgsForTest(args...)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.ThermoModel != thermomodel.NNStructureV1.String() {
		t.Fatalf("got model %q, want %q", opts.ThermoModel, thermomodel.NNStructureV1)
	}
}

func TestParseArgs_HelpShowsThermoModelFlag(t *testing.T) {
	fs := NewFlagSet("ipcr-thermo")
	fs.SetOutput(io.Discard)
	if _, err := ParseArgs(fs, []string{"-h"}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
	found := false
	fs.VisitAll(func(f *flag.Flag) {
		if f.Name == "thermo-model" {
			found = true
		}
	})
	if !found {
		t.Fatal("expected --thermo-model flag to be registered")
	}
}

func TestParseArgs_DefaultSaltModelIsMonovalent(t *testing.T) {
	opts, err := parseArgsForTest(minimalArgs()...)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.SaltModel != thermo.SaltModelMonovalent.String() {
		t.Fatalf("got salt model %q, want %q", opts.SaltModel, thermo.SaltModelMonovalent)
	}
}

func TestParseArgs_UnknownSaltModelRejected(t *testing.T) {
	args := append(minimalArgs(), "--salt-model", "hidden-env")
	_, err := parseArgsForTest(args...)
	if err == nil {
		t.Fatal("expected unknown salt model error")
	}
	if !strings.Contains(err.Error(), "unknown salt model") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseArgs_ThermoDetailsFlag(t *testing.T) {
	args := append(minimalArgs(), "--thermo-details")
	opts, err := parseArgsForTest(args...)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if !opts.ThermoDetails {
		t.Fatal("expected --thermo-details to be enabled")
	}
}

func TestParseArgs_Owczarzy08SaltModelAccepted(t *testing.T) {
	args := append(minimalArgs(), "--salt-model", thermo.SaltModelOwczarzy08.String())
	opts, err := parseArgsForTest(args...)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.SaltModel != thermo.SaltModelOwczarzy08.String() {
		t.Fatalf("got salt model %q, want %q", opts.SaltModel, thermo.SaltModelOwczarzy08)
	}
}

func TestParseArgs_DNTPFlag(t *testing.T) {
	args := append(minimalArgs(), "--dntp", "800uM")
	opts, err := parseArgsForTest(args...)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.DntpSpec != "800uM" {
		t.Fatalf("got dNTP spec %q, want 800uM", opts.DntpSpec)
	}
}
