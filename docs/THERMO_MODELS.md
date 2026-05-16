# Thermodynamic models, score profiles, and release claims

`ipcr-thermo` is a thermodynamically informed ranking tool. It is not a full
PCR kinetics simulator. The implementation combines nearest-neighbor duplex
terms, explicit fallback labels, secondary-structure competition terms, and
empirical PCR/gel score profiles so users can rank candidate amplicons and see
why a product was favored or filtered.

The practical release claim is:

> `ipcr-thermo` provides nearest-neighbor-informed in-silico PCR ranking with
> explicit approximation metadata and deterministic outputs.

Do not describe the current implementation as fully thermodynamically faithful
PCR amplification modeling. PCR yield, gel brightness, polymerase kinetics,
modified probes, and complete loop/mismatch parameter tables still require
calibration or additional chemistry-specific parameters.

## Thermodynamic implementation modes

| Mode               | Purpose                                                               | Notes                                                                                                                    |
| ------------------ | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `legacy-heuristic` | Historical score path                                                 | Maintained for backward compatibility. Scores are not directly comparable with NN modes.                                 |
| `nn-duplex-v1`     | Primer-template nearest-neighbor duplex ranking                       | Uses runtime conditions, salt model, primer concentration, IUPAC thermo policy, and explicit mismatch fallback metadata. |
| `nn-structure-v1`  | `nn-duplex-v1` plus primer hairpin/self-dimer/cross-dimer competition | Uses the current secondary-structure evaluator and reports structure policy/model metadata.                              |

## Structure model labels

| Label                   | Meaning                                                                                                                            |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `nn-contiguous-stem-v1` | Contiguous Watson-Crick stem model. Preserved as the simplest secondary-structure baseline.                                        |
| `nn-stem-loop-v2`       | Bounded gapped-stem model with bulges, small internal loops, asymmetric-loop penalties, and structure dangling-end approximations. |

`nn-stem-loop-v2` is a bounded approximation. It does not replace a full dynamic
programming secondary-structure engine with complete loop, bulge, dangling-end,
and coaxial-stacking parameter tables.

## Score profiles

| Profile   | Intended question                                                      | Formula sketch                                            |
| --------- | ---------------------------------------------------------------------- | --------------------------------------------------------- |
| `binding` | Which primer pair binds best under the configured thermodynamic model? | Primer-template score, minus enabled structure penalties. |
| `pcr`     | Which product is expected to amplify efficiently?                      | `binding + extension_bonus - length_penalty`.             |
| `gel`     | Which product is expected to look strongest on an agarose gel?         | `pcr + band_mass_bonus`.                                  |

`pcr` and `gel` are empirical ranking profiles. They are useful for reproducing
observed product dominance, but they are not calibrated polymerase kinetics or a
quantitative fluorescence/gel-intensity model.

### Choosing a profile

| Use case                                      | Recommended profile        | Rationale                                                                       |
| --------------------------------------------- | -------------------------- | ------------------------------------------------------------------------------- |
| Primer-design triage                          | `binding`                  | Least empirical; closest to primer-template/structure thermodynamics.           |
| Multiplex product prioritization              | `pcr`                      | Adds extension and long-product penalties without treating band mass as signal. |
| Comparing against agarose-gel band prominence | `gel`                      | Adds a band-mass proxy so short products are not automatically favored.         |
| Debugging model changes                       | `binding --thermo-details` | Keeps the score closest to the underlying terms and exposes metadata.           |

Scores from different profiles should be treated as different ranking scales. A
`gel` score is not directly comparable with a `binding` score even when the same
amplicon and conditions are used.

## Salt and concentration models

| Salt model      | Meaning                                                                        |
| --------------- | ------------------------------------------------------------------------------ |
| `monovalent`    | Monovalent nearest-neighbor salt correction.                                   |
| `owczarzy-lite` | Mg-to-effective-Na approximation for compatibility and continuity.             |
| `owczarzy08`    | Mixed monovalent/divalent correction with dNTP-adjusted free Mg approximation. |

When reporting results, keep the raw and effective ionic conditions visible:
`na_m`, `mg_m`, `dntp_m`, `effective_na_m`, and `free_mg_m`.

## IUPAC thermodynamics policy

Thermodynamic scoring supports explicit IUPAC expansion policies:

| Policy      | Behavior                                                                                        |
| ----------- | ----------------------------------------------------------------------------------------------- |
| `strict`    | Reject non-ACGT primers/probes in NN thermodynamics.                                            |
| `worst`     | Expand concrete variants and use the weakest score. Recommended for conservative assay ranking. |
| `best`      | Use the strongest concrete variant.                                                             |
| `mean`      | Average concrete variants.                                                                      |
| `enumerate` | Emit per-expansion diagnostics where supported.                                                 |

Always report the IUPAC metadata when present:
`iupac_thermo_policy`, `iupac_expansion_count`, `iupac_expansion_capped`, and
`iupac_effective_variant`.

## Probe thermodynamics

Probe thermodynamics reuses the primer-template NN machinery for unmodified DNA
probe/site duplexes. The current modes are:

| Mode       | Behavior                                                                    |
| ---------- | --------------------------------------------------------------------------- |
| `annotate` | Compute and report probe thermodynamics without changing product score.     |
| `gate`     | Penalize or suppress products that fail probe presence/margin requirements. |
| `blend`    | Blend probe margin into the product score using `--probe-weight`.           |

Modified probes are not fully modeled. In particular, MGB, LNA, molecular beacon,
quencher, and dye effects are not automatically calibrated. For MGB assays, use
`--probe-score-mode annotate` or `--probe-thermo=false` unless a calibrated probe
modifier model is added.

## Fallback metadata that should remain visible

Thermodynamic outputs should expose when approximate paths were used. The most
important labels are:

- mismatch policy and fallback counts,
- structure policy/model,
- salt model and free/effective ion concentrations,
- IUPAC policy and expansion/capping status,
- score profile,
- probe score mode and gate penalty.

These fields are part of the release story: a result can be useful even when it
uses approximations, as long as the approximation is visible.

## Output comparability rules

- Compare scores only within the same `thermo_model`, `score_profile`, salt model,
  IUPAC policy, probe score mode, and annealing conditions.
- Treat `legacy-heuristic` scores as historical compatibility scores, not as the
  same unit scale as NN modes.
- When reporting a ranked panel, include the model labels and conditions used to
  generate the ranking.
- Prefer JSON/JSONL or `--thermo-details` TSV for release examples because scalar
  scores alone hide fallback and approximation metadata.

## Known remaining limitations

The current thermodynamic implementation is intentionally transparent about these
limits:

1. Curated mismatch triplet tables are still incomplete; fallback mismatch terms
   remain possible and must remain reported.
2. `owczarzy08` uses a practical mixed-salt/free-Mg approximation, not a full
   activity-coefficient chemistry model.
3. `nn-stem-loop-v2` is not a complete secondary-structure dynamic-programming
   engine.
4. PCR and gel score profiles are empirical rankers, not full amplification
   kinetics.
5. Modified probe chemistries such as MGB require opt-in calibration.
6. Scores from different thermo modes or score profiles should not be compared
   as if they were on one universal physical scale.

## Release checklist

The operational checklist lives in [THERMO_RELEASE_CHECKLIST.md](./THERMO_RELEASE_CHECKLIST.md).

Before advertising a release as thermodynamically informed, verify:

```bash
go test ./... -count=1
(cd core && go test ./... -count=1)
go test -tags thermo ./... -count=1
golangci-lint run
make build
```

Also check representative CLI output with:

```bash
bin/ipcr-thermo --examples
bin/ipcr-thermo --help
```

and at least one `--thermo-details` run for each profile:

```bash
--score-profile binding
--score-profile pcr
--score-profile gel
```
