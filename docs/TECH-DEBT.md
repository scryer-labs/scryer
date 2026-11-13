# Technical debt and existing analysis limitations

Updated 2026-11-10. Item IDs are not priority ranks. New analyzer installation,
language support and architecture/test fact capabilities belong in
[milestones](MILESTONES.md), not duplicated implementation tasks here. The CLI/configuration and
engine-extraction sequence takes precedence unless a real regression requires a fix.

## TD-001: Finish the Go/JVM migration cleanup

- [x] Status: resolved during independent Java-engine extraction, 2026-11-02.

Kotlin CLI/rendering fixtures and their presentation tests were removed rather
than coupling engine tests to Core resources. Fact/JSON assertions from mixed
suites remain with the engine. Core owns terminal/MD/HTML/browser regression tests;
workspace acceptance exercises real engine JSON through Core scan/analyze reports.
No Kotlin sources or Gradle build files remain in Core.

## TD-002: Java resolution and reflection boundaries

- [ ] Status: ongoing, fixture-driven improvements; not full runtime discovery.

Current Java analysis has bounded literal reflection, partial build/source models
and explicit unresolved/external/framework/generated-code boundaries. Classpath
ambiguity, custom/generated outputs, richer Maven effective-model provenance and
unsupported dynamic flows can limit attribution. Improve concrete cases with
regression fixtures and retain unknown boundaries; do not infer no callers or
complete resolution from a partial graph. Details remain in the
[command reference](CLI.md) rather than copied into this register.

## TD-003: Java evidence discovery and attribution limits

- [ ] Status: deferred extensions, independently of cross-stack feature work.

Current discovery covers supported JUnit XML/JaCoCo files and conventional class
outputs. Custom output locations, other coverage formats and test-to-method causal
attribution are incomplete/unavailable. Freshness, class-ID provenance and separate
datasets must survive new adapters. Static test-role file counts, aggregate method
hits and command success do not establish assertion effectiveness. Address a real
collector/evidence use case at a time; bounded shared checks may follow
[decision 003](decisions/003-stack-discovery-and-shared-rules.md), but do not infer
missing evidence or test adequacy from current reports.

## TD-004: Packaging portability and independent release transition

- [ ] Status: deferred maintenance coordinated with M1.

Packages are host-specific directory distributions; current CI builds Ubuntu/Linux
amd64 packages and uploads workflow artifacts. Windows-native distribution,
additional OS/architecture coverage and published release installation are not
proven by those artifact checks. Core and engine packaging are now separate; maintain relocation, runtime
discovery, legal notices and the distinction between analyzer and target JVMs. Installer/repository design is owned by
[decision 001](decisions/001-analyzer-distribution.md).

## TD-005: Remove refactor-plan generation from Scryer

- [x] **Done** — ~~Remove refactor-plan generation.~~ Code removed 2026-11-01;
  accepted by **DevOps / Daily Development** run `20261101T220229829Z-83db3881`
  (all eight steps succeeded on macOS).

Removed `refactor-plan generate`, its phase/cycle scheduling implementation,
platform-plan YAML export and dedicated presentation/tests. A CLI regression check
requires the removed command to fail without invoking an analyzer. Orrery already derives
phases and creates plan drafts from dependency facts; its Scryer provider calls
`analyze workspace`, not `refactor-plan`. Execution-plan generation belongs to the
consuming platform. Preserve the CSV dependency facts and JSON/HTML path used by
Orrery. Future bounded shared checks do not justify keeping this plan generator.

## TD-006: Retire CSV-in/dependencies-out workspace adaptation

- [x] Status: deprecated and removed. Scryer no longer supports the Mod-fed
  `workspace import` operation or its CSV-based JSON/HTML reports. Orrery
  `scryer@v1.9` omits the former `analyze-workspace` operation. Older plans
  that call it require a Scryer binary that still supports the adapter.

Future Java cross-project dependency scanning will use Java-engine facts and
native `workspace scan` aggregation. C++ already follows this pattern for
CMake library references: scanning Scriptorium matches Brush and Inkstone
to Felt within the configured workspace. Java workspace dependency matching
remains planned.

## TD-007: Retire Java-only project dependency flags

- [x] Status: both legacy Java dependency view flags removed.

`project scan --facts dependencies` returns Java's direct dependency
declarations. It scans statically by default. `--resolve` opts into build-tool
execution for engines that declare dependency resolution support. The Java
engine returns resolution status and graph nodes and edges when available.
Engines that only support static collection return those facts with a message.

Orrery's **Scryer + Java Engine Integration** run
`20261110T123633569Z-e4313f87` verified both modes in a mixed Java and C++
fixture. It used a deterministic Maven Wrapper fixture to verify execution
and graph transport while preserving C++ include facts.

Both `project scan --dependencies` and `project scan --dependency-tree` have
been removed. The unified terminal and Markdown reports group Java dependency
declarations by module and configuration and render a text tree when the graph
is available. JSON consumers can use the full graph from the same scan.

### Remaining scan option migration

- Retire the legacy `project scan --static` flag once the unified scan replaces
  that path. The static mode is the default and resolution is opt-in.
- The legacy `project scan --stack` selector has been removed. Unified scans
  use automatic language detection and engine dispatch. `--facts stack` reports
  detected technologies and `--facts loc` reports line counts.
  `project analyze --stack` remains available.
- Keep `--remote-list` as an opt-in for remote version metadata. Treat it
  separately from local dependency graph resolution and move it onto the
  unified scan as engines gain support.

### Build-tool selection

Keep build-tool selection separate from language detection and engine dispatch.
A project can use Gradle to build C++, for example. Today Core treats
`--build-tool` as a Java option and only accepts Maven or Gradle. Move build-tool selection onto the unified scan and remove this
Java-specific routing and validation from Core. Each language engine can then select or validate a build
tool based on project configuration or detection and its supported adapters.

Dedicated build-tool engines such as `cmake-engine`, `gradle-engine`, and
`maven-engine` are a possible future direction if build facts need to be shared
across language engines. There are currently no plans to introduce them.
