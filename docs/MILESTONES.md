# Product milestones

Updated 2026-10-31. This is the single priority/status index, not authorization to
implement all stages. [Product direction](PRODUCT-DIRECTION.md) defines scope;
[decisions](decisions/002-scan-scopes-and-fact-boundary.md) record agreements;
[technical debt](TECH-DEBT.md) records existing limitations. Planned commands and
configuration below are not current capabilities.

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

## Priority 1 — CLI, explicit scopes and portable configuration

- [ ] Agree concrete project/workspace journeys and command grammar before changing
  commands. Workspace becomes a first-class scope rather than `analyze workspace`.
  Classify existing scan/analyze behavior, evidence matching and phase/plan
  suggestions against the facts-only boundary; preserve existing behavior until
  a deliberate migration is implemented.
- [ ] Define handwritten workspace/project `.scryer/` configuration: retained,
  explicit membership using workspace-relative paths; project-level scan options;
  defaults for standalone project scans; root project `.` support. No directory-based
  membership guessing or reverse membership duplicated in project configuration.
- [ ] Define JSON results for every command, default JSON delivery, stdout/file
  formats, stderr diagnostics/progress, exit codes and partial-result semantics.
  Include provenance and distinguish missing engine, incompatible engine,
  unsupported capability and execution failure. JSON mode stdout contains only JSON.
- [ ] Keep source/symbol extraction usable without target build/run success.
  Separate optional resolution and execution capabilities and report coverage gaps.
- [ ] Write meaningful command/configuration contracts and migration examples.
  Current CLI remains documented in [CLI](CLI.md) until implementation changes it.

## Priority 2 — Independent Java engine installation and extraction

See [decision 001](decisions/001-analyzer-distribution.md). Work in runnable slices:

1. [ ] Finish the logical external boundary from the current Go/Java adapter.
   External engines must not import Core's `internal/` packages. Define the minimal
   public protocol/contract, identity/version and capability handshake.
2. [ ] Verify install/uninstall/list using a minimal local engine package and an
   actual request/response before moving the Java implementation. Commands such as
   `scryer engine install`, `uninstall` and `list` are provisional. A failed install
   must not appear as a usable engine. Public download/update infrastructure can wait.
3. [ ] Move Java into the user-created `scryer-java-engine` repository, with a Go
   entry point around Kotlin/JVM. Preserve Java capabilities and evidence; Java
   runtime discovery/optional runtime packaging belong with that engine.
4. [ ] Invoke installed Java through ordinary Core commands; keep non-Java Core
   operations usable without a JVM. Split tests/builds/releases by ownership and
   retain host/engine compatibility, cancellation and package smoke checks.

No dynamic Go ABI or general plugin framework is selected. Engine naming, package
schema and exact install flags remain open. macOS is the primary acceptance target;
Windows migration is deferred until both Scryer and Orrery are stable.

### 2.1 — Update Orrery after Java extraction

- [ ] Adapt Orrery's Scryer provider and relevant scan-plan bindings to the new
  command/configuration/engine contract after Priority 2. Publish a new provider
  version instead of changing published YAML. Validate actual invocation, JSON,
  artifacts and partial/unsupported outcomes through Orrery, while retaining
  standalone CLI use.

## Priority 3 — Additional languages and mixed-project facts

Choose the first language after Java extraction and the Orrery adaptation. Do not
preselect an implementation order or promise complete semantic engines together.

- [ ] C++: source/build/symbol facts and declared/resolved library dependencies,
  including compiler flags, macros, generated sources and incomplete-model limits.
- [ ] Rust: crate/module/symbol facts and dependency relationships, preserving build
  configurations and declared/resolved identities.
- [ ] TypeScript: project/module/symbol and dependency facts, with explicit supported
  frontend/bridge patterns and unresolved boundaries.
- [ ] Go: module/package/build/symbol and dependency facts.
- [ ] Markdown: document structure and link/reference facts, with unresolved links.
- [ ] Mixed-language projects: compose installed engines for Scriptorium Ink,
  Orrery and Scryer; report per-engine capability/coverage before richer cross-language
  links. Build-system modules are not automatically separate registered projects.
- [ ] Expose typed architecture entities/relations, source locations, structural
  signals and explicit user mappings. Folder nesting is not proof of intended
  architectural layers. Deliver useful project facts before every language is ready.
- [ ] Extend test inventory/execution evidence where supported; preserve independent
  datasets, freshness, provenance and unknowns. Test adequacy and strategy judgments
  belong to consumers rather than a new Scryer assessment engine.

## Priority 4 — Workspace-local cross-repository relationships

Implement bounded patterns in this order; retain useful existing CSV input support:

1. [ ] Library/package dependencies: match consumers to providers within the explicit
   workspace, preserving identity/version/configuration and unresolved/candidate
   matches instead of equating package-name equality with a confirmed local edge.
2. [ ] IPC: supported shared-memory, named-pipe and macOS communication patterns,
   with evidence for endpoints and matching. Ubuntu-specific mechanisms are outside
   current scope; dynamic communication may remain unknown.
3. [ ] REST API declarations/calls and supported provider/consumer matching; lower
   priority than library relationships and IPC.

Cross-repository edges retain their relation type and evidence. Dependencies, calls,
IPC and REST do not become one undifferentiated arrow. Observed absence must remain
distinct from an engine's missing capability or incomplete scan coverage.

## Deferred integration and consumer responsibilities

- [ ] Consider Orrery provider operations for editing Scryer configuration and an
  explicit integration action that runs a default plan to generate project entries.
  Name/UX are undecided. Show differences and preserve handwritten scan options;
  no automatic synchronization or silent overwriting in the initial scope.
- [ ] Revisit Windows development/distribution for the input-method project after
  Scryer and Orrery stabilize; do not add Windows acceptance work now.

Architecture-rule evaluation, Hunch-test comparison, refactoring recommendations,
test-effectiveness/strategy judgments and retained evolution timelines belong to
consumers. This supersedes earlier Scryer roadmap entries for an architecture-guard
DSL and judgment engine. Scryer supplies versioned facts, evidence and explicit
before/after observations without owning the persistent history or verdict.
