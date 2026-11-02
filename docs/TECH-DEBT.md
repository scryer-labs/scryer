# Technical debt and existing analysis limitations

Updated 2026-11-02. Item IDs are not priority ranks. New analyzer installation,
language support and architecture/test fact capabilities belong in
[milestones](MILESTONES.md), not duplicated implementation tasks here. The CLI/configuration and
engine-extraction sequence takes precedence unless a real regression requires a fix.

## TD-001: Finish the Go/JVM migration cleanup

- [ ] Status: deferred; preserve behavior during analyzer extraction.

The old Kotlin CLI/renderers remain in test sources as migration regression
fixtures; public commands/rendering already live in Go. Decide which checks remain
useful when Java moves, migrate shared report/contract coverage to its owner and
remove duplicate fixtures only after equivalent checks exist. Do not delete the
regression baseline just because production no longer calls it.

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
proven by those artifact checks. Separate core and analyzer packaging without
breaking relocation, runtime discovery, legal notices or the distinction between
the analyzer JVM and target JVM. Installer/repository design is owned by
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

- [ ] Status: temporary compatibility capability; blocked on workspace dependency scanning.

Retain the Mod-fed CSV capability for now: it normalizes externally collected facts
and renders JSON/HTML; it does not scan repository dependencies itself. Orrery
currently depends on Mod CLI collection followed by this adapter.

The scope-first entry is now `workspace import --projects ... --dependencies ...`.
The old `analyze workspace` entry is removed alongside immediate provider adaptation;
there is no compatibility alias. The temporary adapter remains until native scanning
replaces it. This supersedes the prior staged alias policy under decision 004.

After native workspace scanning supports the needed dependency collection and the
Orrery provider/scan plans have migrated and been verified, completely remove the
temporary adapter and its exclusive code/tests/docs. Shared fact/report code still
needed by native scanning may remain. Do not break the Mod-based consumer before
the corresponding migration is ready; CSV input is not a permanent requirement.
