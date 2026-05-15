<!-- ./docs/ARCHITECTURE.md -->

# ipcr Architecture (layers & rules)

**Goal:** keep the engine fast/clean, keep presentation optional, and make adding new tools easy.

## Layers (top → bottom)

1. **cmd/** — tiny binaries (signal handling, exit code).
2. **internal/app, internal/probeapp, internal/multiplexapp, internal/nestedapp** — parse CLI and call the shared harness.
3. **internal/appcore** — one harness for all tools: chunking, engine, pipeline, visitor, writer.
4. **internal/writers, internal/visitors** — extension points for output and filtering.
5. **internal/pipeline** — FASTA chunking, dedupe, stream products.
6. **internal/engine, internal/primer, internal/probe, internal/oligo** — domain logic.
7. **internal/fasta** — IO for FASTA streams.
8. **internal/output, internal/probeoutput, internal/nestedoutput, internal/pretty** — concrete formats & ASCII rendering.
9. **internal/common, internal/runutil, internal/cli\*, internal/version** — leaf utilities.

## Allowed imports (arrows)

- `cmd/*` → `internal/app*` only.
- `internal/app*` → `appcore`, `cli*/probecli/nestedcli`, `visitors`, `writers`, `runutil`, `version`, `primer`.
- `appcore` → `cmdutil`, `engine`, `pipeline`, `primer`, `visitors`, `writers`, `runutil`.
- `writers` → `output/probeoutput/nestedoutput`, `pretty`, `engine`, `common`.
- `pipeline` → `engine`, `fasta`, `primer`, `common`.
- `engine` → `primer` (and stdlib).
- `output/probeoutput/nestedoutput/pretty` → may import `engine` types, but **must not** import `app*`, `appcore`, `pipeline`, `cli*`.

## Key invariants

- **Only writers know about “pretty”.**
- **Engine never depends upward.** (no imports of app, pipeline, writers, cli, output)

## Thermodynamic modeling boundary

`ipcr-thermo` is implemented as a ranking layer over the core amplicon engine.
The engine still finds candidate products; the thermo visitor annotates and
ranks those products using model/profile metadata.

Keep these boundaries explicit:

- Thermodynamic modes such as `nn-duplex-v1` and `nn-structure-v1` describe how
  primer/probe/structure terms are calculated.
- Score profiles such as `binding`, `pcr`, and `gel` describe how those terms are
  combined for ranking.
- Fallback policies, IUPAC expansion status, salt model, and probe score mode are
  output metadata, not presentation-only details.
- Empirical profiles must not be documented as full PCR kinetics.

Detailed release-claim guidance is in [`docs/THERMO_MODELS.md`](./THERMO_MODELS.md). Release/smoke-test guidance is in [`docs/THERMO_RELEASE_CHECKLIST.md`](./THERMO_RELEASE_CHECKLIST.md).

## Future split

If you ever need external reuse: lift `engine`, `primer`, `probe`, `oligo`, `fasta` into a separate module (`ipcr-core`) and keep `app`, `appcore`, `writers`, `output`, `pretty` in this repo.
