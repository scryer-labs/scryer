# Product milestones

Updated 2026-11-02. This is the single priority/status index, not authorization to
implement all stages. [Product direction](PRODUCT-DIRECTION.md) defines scope;
[current design decision](decisions/004-multilanguage-scopes-cli-and-engine-config.md) records agreements;
[technical debt](TECH-DEBT.md) records existing limitations. Planned commands and
configuration below are not current capabilities.

## Done — current baseline

- [x] **Done** — ~~Separate the Go/Cobra public CLI, repository/snapshot lifecycle and shared reporting from a Kotlin/JVM Java worker.~~ Java now lives in the independent engine repository.
- [x] **Done** — ~~Define a versioned analyzer request/response boundary with identity validation, cancellation and explicit partial/unknown evidence.~~ See [current contracts](contracts/README.md); this is not an installable plugin system.
- [x] **Done (Java)** — ~~Collect static declarations/source signals, build-model dependency graphs and optional remote-version comparisons through Maven/Gradle adapters.~~
- [x] **Done (Java, bounded)** — ~~Compare immutable before/after snapshots, Java methods/symbols, partial caller/indirect impact and bounded literal reflection.~~
- [x] **Done (Java)** — ~~Run default/custom target test lifecycles, retain fresh JUnit/JaCoCo evidence and match independent datasets to affected methods with explicit unknowns.~~ This does not prove assertion quality or complete behavioral coverage.
- [x] **Done** — ~~Support isolated working-tree after snapshots, explicit build-tool selection and target/analyzer JVM separation.~~
- [x] **Done** — ~~Render shared terminal/JSON/Markdown reports and standalone interactive HTML analysis reports.~~ Java scan file export currently writes Markdown; output support is command-specific.
- [x] **Done** — ~~Normalize workspace dependency CSVs to JSON/HTML.~~ No Mod CLI or JVM is needed for CSV processing. This adapter is temporary under TD-006; former plan generation has been removed.
- [x] **Done** — ~~Split Core Go/browser checks from independent engine Go/JVM checks; verify Core-only and engine slim/Java-21-runtime packages.~~ Workflow artifacts are not published releases.

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
  facts.~~ Legacy entries are removed alongside immediate consumer adaptation.
- [x] **Done** — ~~Keep Java flags/default behavior during this first slice.~~ No new
  stack/LOC, workspace-native scanning or engine installation support is claimed.

Acceptance after legacy removal: Scryer **Daily Development** run
`20261102T150144163Z-2704a5da`, Orrery daily run
`20261102T150501275Z-8b2c0035`, and workspace integration run
`20261102T150607227Z-b1fabe13` all succeeded. Old scan/analyze/analyze-workspace
entry invocations now fail with usage errors. Provider v1.4 adapts commands now;
the later 2.1 checkpoint remains for installed-engine changes.

## Priority 2 — Independent Java engine installation and extraction

See [decision 001](decisions/001-analyzer-distribution.md). Work in runnable slices:

1. [x] **Done (mock process connection)** — ~~Add engine list/install/uninstall and
   probe an independent Go mock engine.~~ Explicit --executable calls describe and
   verifies identity; receipts are mock/non-persistent. No inventory, package install
   or source build is implemented. Core run `20261102T160505366Z-0c862886`, engine run
   `20261102T160505751Z-3446bd8a`, and workspace linkage run
   `20261102T160706145Z-be3a98f0` succeeded. Remote-list/install-all remain deferred.
2. [x] **Done (local packages)** — Persistent install/list/discovery/uninstall of
   complete packages through engine.json and owned user storage. No source checkout
   dependency remains after installation. Installation is platform/identity checked,
   staged and locked; existing installs require explicit uninstall before reinstall.
   Release downloads, archive extraction, source compilation and version switching
   remain future work. Seven real project scans return to modernization's provider
   integration, no longer duplicated inside the engine linkage plan.
3. [x] **Done (behavior-preserving extraction)** — Move Kotlin production code,
   facts/JSON tests, Gradle and JVM/runtime packaging to scryer-java-engine. Core
   invokes wrapper execute through the existing v1 envelope and retains snapshots,
   validation and reporting. No engine imports Core internal packages or report
   resources. Capability/template/config negotiations remain future work.
   Acceptance: Core `20261102T170044849Z-e11cda2d`, engine
   `20261102T170137267Z-6d5f1a23`, workspace integration
   `20261102T170558213Z-f877edf4` all succeeded. The first integration attempt
   incorrectly expected scan HTML; its assertion was corrected to existing scan
   JSON/Markdown behavior, with analyze HTML retained. No scan feature was added.
4. [ ] Package/release the independent engine with bundled Java 21 runtime on GitHub.
   Install compatible stable assets; also support source builds through an engine-owned
   build entry producing the same package layout. Source builds require dev toolchains.
5. [ ] Provide automatic global config/data/cache defaults and optional SCRYER_HOME;
   users must not initialize a global home. The initial user home and persistent local
   inventory are implemented; cache separation and additional preferences remain future
   work. Core installs independently with go install.
   Core/engine CI and plans are already split, and composed acceptance identifies
   known artifacts; persistent installed-home/config acceptance is still pending.

Extraction was intentionally advanced ahead of real installation by the user's
2026-11-02 instruction. The next pair session is a bounded **C ABI shared-library
experiment**, beginning with mock describe, versioning, buffer ownership/freeing,
errors, then cancellation/concurrency/lifecycle and Java worker startup. The goal
includes reusable input-method plugin design experience. Keep the current
Core → wrapper → JVM baseline; adopt a replacement only after comparing evidence.
This is not an implementation task for the extraction slice.

No dynamic Go ABI or general plugin framework is selected. Engine naming, package
schema and exact install flags remain open. Java is the validation engine, not a
requirement that every future capability be added to Java. macOS is the primary
acceptance target;
Windows migration is deferred until both Scryer and Orrery are stable.

### 2.1 — Update Orrery after Java extraction

- [ ] Adapt remaining installed-engine/configuration contracts after Priority 2;
  command migration is already integrated through Scryer provider v1.4. Publish a new provider
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

## 2026-11-02 persistent local installation acceptance

Engine project `20261102T190423713Z-12016f32`, repeatable installed-engine lifecycle
`20261102T190808090Z-4cb58c91`, and modernization formal-provider integration
`20261102T190934467Z-26d9261c` succeeded. Core `20261102T191115041Z-d7ae0ef0` also succeeded, including its final
missing-engine diagnostic check. The linkage plan owns installation/lifecycle;
seven project scans run only through modernization's existing v1.4 provider plans.
A bundled Java engine is installed in the normal user home, without provider path
configuration or a backend restart. This is a local package implementation, not
GitHub release downloading, archive extraction or source compilation.

Global-home correction: ~/.config/scryer is the default on macOS, matching
Orrery/nvim; the initial Library installation was moved once. Acceptance succeeded:
Core 20261102T193332344Z-d8e09434, engine 20261102T193414957Z-72a2c86b,
installed-engine integration 20261102T193525437Z-576265e2, modernization formal
provider integration 20261102T193539815Z-c4c90c1e. The former Library home no longer
exists; no provider settings or backend restart were needed.

Core report views now separate Java-specific presentation from shared formatting.
C ABI and project engine YAML remain pending.
