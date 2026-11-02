# Product milestones

Updated 2026-11-02. This is the single priority/status index, not authorization to
implement all stages. [Product direction](PRODUCT-DIRECTION.md) defines scope;
[current design decision](decisions/004-multilanguage-scopes-cli-and-engine-config.md) records agreements;
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
- [x] **Done** — ~~Normalize workspace dependency CSVs to JSON/HTML.~~ No Mod CLI or JVM is needed for CSV processing. This adapter is temporary under TD-006; former plan generation has been removed.
- [x] **Done** — ~~Build/test Go, JVM and browser graph code in CI; package slim and Java-21-runtime distributions with extracted-package smoke checks.~~ Workflow artifacts are not published releases.

## Immediate cleanup and design checkpoint

- [x] **Done** — ~~Remove refactor-plan generation.~~ See [TD-005](TECH-DEBT.md#td-005-remove-refactor-plan-generation-from-scryer); accepted by DevOps / Daily Development; run recorded in TD-005.
- [x] **Done (design)** — ~~Agree multilanguage scopes, command skeleton, scan/analyze
  separation and engine-config ownership.~~ See decision 004; implementation is pending.
- [x] **Done (design)** — ~~Agree minimal workspace/per-engine YAML ownership and
  JSON-only engine results with Core-owned modular MD/HTML rendering.~~ Decision 004
  records the pre-stability no-compatibility policy; no implementation is claimed.
- [x] **Done** — ~~Reorganize current command entries while preserving Java behavior.~~ Do not
  wait for future engine/workspace schemas. Define the external engine contract during
  extraction and native workspace aggregation contracts when that capability is built.
- [ ] Retain the Mod-fed workspace CSV adapter temporarily under
  [TD-006](TECH-DEBT.md#td-006-retire-csv-independencies-out-workspace-adaptation);
  migrate its entry into the new workspace command shape, then remove the adapter
  once native workspace dependency scanning and consumer migration work.

## Priority 1 — Reorganize existing commands

- [x] **Done** — ~~Add project scan/analyze with optional
  paths defaulting to the current directory, preserving Java behavior.~~
- [x] **Done** — ~~Add workspace import for external CSV
  facts.~~ Hidden legacy entries preserve consumers until their planned migration.
- [x] **Done** — ~~Keep Java flags/default behavior during this first slice.~~ No new
  stack/LOC, workspace-native scanning or engine installation support is claimed.

Acceptance: **Daily Development** run `20261102T144247407Z-58cd8ba1` succeeded in all
eight steps. Installed-CLI checks also verified default-path project scan with the
real Java worker, scope help, workspace JSON/HTML, and matching legacy CSV JSON.

## Priority 2 — Independent Java engine installation and extraction

See [decision 001](decisions/001-analyzer-distribution.md). Work in runnable slices:

1. [ ] Add engine list/install/uninstall with mocks and a minimal worker fixture.
   Defer remote-list; no install-all or stack-driven auto-install.
2. [ ] Verify install/list/invoke/uninstall against the user-created Java-engine
   repository. Start with local packages/source directories and owned install data.
3. [ ] Define the minimal external JSON/capability/template contract, extract Java
   into its Go wrapper plus Kotlin worker, and verify existing features end to end.
   No external engine imports Core's internal packages.
4. [ ] Package/release the independent engine with bundled Java 21 runtime on GitHub.
   Install compatible stable assets; also support source builds through an engine-owned
   build entry producing the same package layout. Source builds require dev toolchains.
5. [ ] Provide automatic global config/data/cache defaults and optional SCRYER_HOME;
   users must not initialize a global home. Core installs independently with go install.
   Split Core/engine CI and plans, and compose Core-plus-engine acceptance with isolated
   homes, known artifacts and serialized dependent operations.

No dynamic Go ABI or general plugin framework is selected. Engine naming, package
schema and exact install flags remain open. Java is the validation engine, not a
requirement that every future capability be added to Java. macOS is the primary
acceptance target;
Windows migration is deferred until both Scryer and Orrery are stable.

### 2.1 — Update Orrery after Java extraction

- [ ] Adapt Orrery's Scryer provider and relevant scan-plan bindings to the new
  command/configuration/engine contract after Priority 2. Publish a new provider
  version instead of changing published YAML. Validate actual invocation, JSON,
  artifacts and partial/unsupported outcomes through Orrery, while retaining
  standalone CLI use.

## After extraction and 2.1 — Core baseline and configuration

Default stack/LOC scanning is implemented after Java extraction to validate that
Core works without any installed engine or Java. This work is deliberately deferred
from the initial command-only slice. Follow decision 004 for the behavior below.

- [ ] Implement workspace init/add/remove and explicit membership. Initialize
  `.scryer/engines/<engine-id>.yaml` from engine templates under projects, without
  project.yaml. Existing files are not automatically overwritten. Workspace has only
  workspace.yaml; engine config content is engine-owned and need not be backward
  compatible before stability.
- [ ] Implement shared `--facts` semantics: default stack/LOC, architecture instead of
  separate symbol/call user commands, workspace forwarding/aggregation, explicit
  unavailable/partial results. Decide filter, static test-inventory and LOC policies.
- [ ] Include engine-independent project/workspace technology-stack discovery in
  CLI design: retain evidence for file/manifest/build signals, support mixed stacks
  and use configured workspace membership without automatically registering projects.
- [ ] Define JSON results for every command, default JSON delivery, stdout/file
  formats, stderr diagnostics/progress, exit codes and partial-result semantics.
  Include provenance and distinguish missing engine, incompatible engine,
  unsupported capability and execution failure. JSON mode stdout contains only JSON.
- [ ] Implement distinct engine/project/workspace structured results; engines emit
  JSON only. Core report modules reuse common components, stack-specific presentation
  and separate scope views. Preserve unknown extensions in JSON-detail fallbacks;
  no report plugin framework. Internal child scans need not generate MD/HTML.
- [ ] Keep source/symbol extraction usable without target build/run success.
  Separate optional resolution and execution capabilities and report coverage gaps.
- [ ] Write meaningful command/configuration contracts and migration examples.
  Current CLI remains documented in [CLI](CLI.md) until implementation changes it.

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
  datasets, freshness, provenance and unknowns. These facts can support later shared
  checks; no aggregate hit/count alone proves test adequacy.

## Priority 4 — Workspace-local cross-repository relationships

Start from concrete library/REST/named-pipe cases rather than a universal matcher.
The list below groups capabilities, not a strict REST-versus-IPC delivery order.
Existing CSV adaptation is temporary
under [TD-006](TECH-DEBT.md#td-006-retire-csv-independencies-out-workspace-adaptation):

1. [ ] Library/package dependencies: match consumers to providers within the explicit
   workspace, preserving identity/version/configuration and unresolved/candidate
   matches instead of equating package-name equality with a confirmed local edge.
2. [ ] IPC: supported shared-memory, named-pipe and macOS communication patterns,
   with evidence for endpoints and matching. Ubuntu-specific mechanisms are outside
   current scope; dynamic communication may remain unknown.
3. [ ] REST API declarations/calls and supported provider/consumer matching using
   Orrery as the concrete case. REST precedes other network mechanisms (GraphQL/RPC).

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

Branch-strategy observation remains lowest priority; it does not block scan/analyze,
engine extraction or the relationship cases above.

## Later capability — Shared checks and bounded suggestions

- [ ] After a second language engine and one or two concrete rule cases, derive a
  shared rule layer over common facts. Avoid duplicated rule implementations across
  engine repositories. Define declarative fact mappings/conditions and reusable
  algorithms together; do not assume JSON alone supplies missing language semantics.
- [ ] Let engines declare available symbol/relation/type/control-flow/data-flow and
  test/execution capabilities incrementally. Rules state required capabilities and
  retain identity/version, parameters, source provenance and evidence. Missing facts
  yield unsupported/unknown results rather than a pass.
- [ ] Add reproducible code-smell, bounded test-effectiveness and evidence-backed
  refactoring checks/suggestions. Keep findings separate from raw facts and report
  explicit limits. `suggest` and `check` are provisional names, not approved grammar.

[Decision 003](decisions/003-stack-discovery-and-shared-rules.md) refines the earlier
blanket assignment of judgments to consumers. Business/design trade-offs,
consultant-style advice, persistent evolution history and plan execution remain
outside Scryer. The shared-rule capability does not block the current CLI/Java
extraction and **2.1** Orrery adaptation sequence.
