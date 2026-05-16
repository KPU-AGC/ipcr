# Thermodynamic release checklist

Use this checklist before releasing or advertising changes to `ipcr-thermo`. It
keeps the public claim aligned with the actual model: thermodynamically informed
ranking with explicit approximation metadata.

## Required automated checks

Run these from the repository root:

```bash
go test ./... -count=1
(cd core && go test ./... -count=1)
go test -tags thermo ./... -count=1
golangci-lint run
make build
preflight
```

If `golangci-lint` is unavailable in a local environment, do not mark the release
ready until CI or another machine has run it.

## Required smoke tests

Run at least one representative command for each row. Use real FASTA fixtures
when available and keep the command/output in release notes or test artifacts.

| Scenario              | Example options                                               | Expected check                                                                                                                |
| --------------------- | ------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Legacy compatibility  | default `ipcr-thermo` invocation                              | Existing historical rows still appear.                                                                                        |
| NN duplex             | `--thermo-model nn-duplex-v1 --thermo-details`                | Output includes NN model, salt model, margins, mismatch metadata, and dangling-end fields when template flanks are available. |
| NN structure          | `--thermo-model nn-structure-v1 --thermo-details`             | Output includes structure policy/model and dimer/hairpin fields.                                                              |
| Score profiles        | `--score-profile binding`, `pcr`, `gel`                       | Product order changes only for documented profile reasons.                                                                    |
| IUPAC thermo          | `--iupac-thermo-policy worst` with a degenerate primer        | Output includes expansion count, capped status, effective variant.                                                            |
| Mixed salt/dNTP       | `--salt-model owczarzy08 --mg 3mM --dntp 0.8mM`               | Output includes raw/effective ion fields.                                                                                     |
| Probe annotate        | `--probe ... --probe-thermo --probe-score-mode annotate`      | Probe fields populate without filtering the product.                                                                          |
| Probe gate            | `--probe ... --probe-thermo --probe-score-mode gate`          | Failing probes are filtered/penalized in a documented way.                                                                    |
| Modified probe caveat | MGB/LNA style assay with `annotate` or `--probe-thermo=false` | Documentation says unmodified-DNA thermo is not calibrated for modified probes.                                               |

## Output metadata checklist

A release example that uses thermodynamics should expose enough metadata to audit
the score. Prefer JSON/JSONL or `--thermo-details` TSV. Check for these fields
when the corresponding layer is enabled:

- `thermo_model` / model label
- `score_profile`
- salt model and ionic concentrations (`na_m`, `mg_m`, `dntp_m`, free/effective ions)
- IUPAC policy, expansion count, cap status, and effective variant
- mismatch policy, fallback counts, source labels, parameter sets/citations, and mismatch penalties
- terminal mismatch and table-backed dangling-end fields when present
- structure policy/model and component penalties
- probe score mode, probe margin, gate penalty, and probe IUPAC metadata

## Release wording checklist

Acceptable:

- “thermodynamically informed ranking”
- “nearest-neighbor based primer/probe scoring”
- “explicit fallback and approximation metadata”
- “empirical PCR/gel score profiles”

Avoid unless a separately validated kinetics model is added:

- “fully thermodynamically faithful PCR simulation”
- “absolute amplification yield prediction”
- “quantitative gel-intensity prediction”
- “MGB/LNA probe thermodynamics” without a named calibrated modifier model

## Model-change checklist

When adding or changing a model term:

1. Add a stable model/policy label.
2. Expose the term in JSON/JSONL and `--thermo-details` where practical.
3. Add or update a unit test for monotonic behavior and a regression test for a
   representative fixture.
4. Document whether the term is literature-parameterized, heuristic, or empirical.
5. Confirm scores from old and new profiles are not described as interchangeable.

## Known release caveats

Document these until they are replaced with calibrated/literature-backed models:

- Curated mismatch triplets cover isolated internal single-base A/C/G/T DNA/DNA
  mismatches.
- Template-adjacent terminal dangling ends next to Watson-Crick closing pairs use
  SantaLucia-Hicks 2004 Table 3 when flanking bases are available.
- Fallback mismatch terms remain part of the model for terminal, tandem/clustered,
  target-`N`, degenerate-edge, and modified-probe contexts.
- `nn-stem-loop-v2` is a bounded structure approximation, not a full partition
  function or dynamic-programming thermodynamic structure engine.
- `owczarzy08` and related salt handling are practical approximations that should
  be checked against goldens.
- `pcr` and `gel` are empirical ranking profiles.
- Modified probes such as MGB probes require explicit calibration before gate or
  blend mode should be trusted.
