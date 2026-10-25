# Product milestones

Updated 2026-10-25. This is the single priority/status index, not a release schedule
or authorization to implement every stage. The next delivery focus is analyzer
extraction and installation. Additional language/relationship, architecture and
test-analysis goals are an incremental direction, not one promised short sprint.
Existing implementation gaps remain in [technical debt](TECH-DEBT.md).

## Done — current baseline

- [x] **Done** — ~~Separate the Go/Cobra public CLI, repository/snapshot lifecycle and shared reporting from a Kotlin/JVM Java worker.~~ Java still lives in this repository.
- [x] **Done** — ~~Define a versioned analyzer request/response boundary with identity validation, cancellation and explicit partial/unknown evidence.~~ See [current contracts](contracts/README.md); this is not an installable plugin system.
- [x] **Done (Java)** — ~~Collect static declarations/source signals, build-model dependency graphs and optional remote-version comparisons through Maven/Gradle adapters.~~
- [x] **Done (Java, bounded)** — ~~Compare immutable before/after snapshots, Java methods/symbols, partial caller/indirect impact and bounded literal reflection.~~
- [x] **Done (Java)** — ~~Run default/custom target test lifecycles, retain fresh JUnit/JaCoCo evidence and match independent datasets to affected methods with explicit unknowns.~~ This does not prove assertion quality or complete behavioral coverage.
- [x] **Done** — ~~Support isolated working-tree after snapshots, explicit build-tool selection and target/analyzer JVM separation.~~
- [x] **Done** — ~~Render shared terminal/JSON/Markdown reports and standalone interactive HTML analysis reports.~~ Java scan file export currently writes Markdown; output support is command-specific.
- [x] **Done** — ~~Normalize workspace dependency CSVs to JSON/HTML and generate suggested phases/cycle diagnostics and a draft consumer-plan YAML.~~ No Mod CLI or JVM is needed for CSV processing.
- [x] **Done** — ~~Build/test Go, JVM and browser graph code in CI; package slim and Java-21-runtime distributions with extracted-package smoke checks.~~ Workflow artifacts are not published releases.

## Priority 1 — M1: Independently installed analyzers

See [decision 001](decisions/001-analyzer-distribution.md). Implement in runnable
slices, retaining Java behavior throughout the split:

1. [ ] Establish an external analyzer package/Go-wrapper boundary that does not
   depend on importing this repository's `internal/` packages. Decide the minimal
   public contract/SDK and compatibility/capability handshake from the working adapter.
2. [ ] Extract the Java analyzer into its own repository, with a Go entry point
   wrapping the Kotlin/JVM worker. Kotlin still owns Java-specific analysis; Go
   packaging does not mean rewriting that analysis in Go. No repository is created
   by this documentation change.
3. [ ] Add plugin installation/discovery through a command such as
   `scryer install java`, independent analyzer versions and platform-aware package
   selection. Define distribution source, integrity checks, local installation,
   replacement/rollback and missing/incompatible-analyzer diagnostics.
4. [ ] Make Java analysis run through the installed analyzer rather than a required
   in-tree implementation. Retain standalone Scryer use and language-neutral reports;
   record tool/analyzer/schema versions so consumers can interpret historical facts.
5. [ ] Split builds/releases and move Java-specific tests with their owner while
   preserving host/adapter contract and end-to-end package checks. Define how the
   optional Java 21 runtime ships with the independently installed analyzer.

The existing Go `Analyzer` interface and Java process adapter are the starting
point, not evidence that install/discovery, a plugin SDK or Go dynamic-module
loading already exists. No external Go ABI/plugin mechanism is selected.

## Priority 2 — M2: More stacks and relationship facts

Order individual stacks by real fixtures/use cases; no per-language order is
agreed yet. Deliver static facts first, then richer resolution/evidence where the
stack supports it. Unsupported capabilities must stay explicit.

- [ ] Markdown: document structure and local/external link/reference relationships,
  with unresolved links and scope limits rather than treating documents as Java code.
- [ ] C++: project/build structure, code/symbol observations and declared/resolved
  library dependencies, with compiler/build-system and generated-code boundaries.
- [ ] Rust: crate/workspace/module relationships, code facts and library dependency
  graphs, retaining requested/resolved versions and configuration where available.
- [ ] Tauri and mixed-language repositories: compose relevant analyzers, represent
  frontend/backend boundaries and supported command/event/IPC bindings. Tauri is
  not a separate source language; do not require a Java-shaped project model.
- [ ] IPC discovery, including supported named-pipe/message/command patterns, and
  REST API provider/consumer relationships. Retain endpoint/protocol and source
  evidence where available; distinguish declaration, static inference and observed
  execution. Dynamic routes/dispatch and external consumers may remain unknown.
- [ ] Common library/package dependency facts across supported stacks, with stable
  identities, provenance, direct/transitive context and unresolved edges. Do not
  merge package dependencies, calls, IPC and REST into one untyped arrow meaning.

Keep reports consumable as structured JSON and human-readable reports; adapters
must not depend on Orrery. Existing CSV workspace facts remain supported rather
than requiring the future discovery engine.

## Priority 3 — M3: Architecture facts across supported stacks

- [ ] Build a language-neutral, evidence-backed architecture model: projects,
  components/modules, optional layer assignments and typed relationships, with
  source locations, analyzer versions and partial/unknown coverage.
- [ ] Support Markdown, C++, Rust and Tauri/mixed repositories incrementally using
  their M2 capabilities. User configuration may supply component/layer mappings;
  do not present folder nesting as proof of clean-architecture intent.
- [ ] Export comparable architecture snapshots/differences with stable entity
  identity or explicit uncertain matching, source revision/dirty-state context and
  provenance. Preserve removed entities/relationships and distinguish missing
  analysis from observed removal.

**Consumer boundary:** Orrery may retain and compare snapshots, draw interactive
Excalidraw-style/layered/concentric architecture views, apply editable display
names and show architecture-evolution stages. Scryer owns facts and bounded
analysis, not Orrery's workspace history, UI templates or pipeline execution.
Display-name overrides do not rewrite the facts or historical observed names.

## Priority 4 — M4: Cross-stack test analysis and strategy diagrams

- [ ] Extend test discovery, framework/lifecycle facts and source/test roles to the
  supported stacks. Markdown document/link checks are distinct from compiled-code
  unit/integration/system tests; unsupported categories remain not-applicable/unknown.
- [ ] Add stack-specific execution/evidence adapters where supported, with explicit
  request/side effects, command results, artifacts and snapshot provenance. Preserve
  the separation between static candidates and verified execution observations.
- [ ] Analyze test effectiveness with explainable, bounded evidence: affected code,
  execution gaps and available assertions/behavioral evidence. Passing tests and
  aggregate coverage alone do not prove effective assertions or refactoring safety;
  do not invent a universal confidence score or per-test causal attribution.
- [ ] Classify observed automated-test layers with configurable conventions and
  evidence/uncertainty, and generate test-layer/strategy diagrams (for example
  pyramid/trophy-style views where appropriate). Distinguish observed inventory
  from a proposed strategy; no universal layer ratio or recommended mix is agreed.
  Select export/interactive formats when implementing the renderer; images must
  derive from the same structured data, not become a second analysis source.

Java's existing test/evidence matching is the baseline to preserve and extend,
not a claim that these cross-stack or strategy-diagram capabilities exist today.

## Deferred — architecture guard DSL and richer evolution analysis

- [ ] Define an architecture-rule DSL for Scryer: constraints/tests over observed
  components/layers/dependencies/IPC/API relationships, compared against scan facts.
  Record rule version, inputs, evidence and result; insufficient scan coverage is
  unknown/unsupported, not a passing assertion. Begin with concrete rule cases;
  no grammar, parser library or new CLI command is approved yet.
- [ ] Provide reproducible architecture comparisons usable by Orrery's interactive
  evolution timelines and major-change stages. Derived stage boundaries must be
  explainable; consumers can annotate them. Static topology alone does not reveal
  the original author's intent or prove architectural improvement.

These are future analysis capabilities. Scryer does not become a plan executor,
agent manager or mandatory backend for an Orrery-only visualization system.
