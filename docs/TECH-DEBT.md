# Technical debt and existing analysis limitations

Updated 2026-10-25. Item IDs are not priority ranks. New analyzer installation,
language support, architecture rules and test-strategy diagrams belong in
[milestones](MILESTONES.md), not duplicated implementation tasks here. The next
analyzer-extraction slice takes precedence unless a real regression requires a fix.

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
collector/evidence use case at a time; broader strategy analysis is M4, not a
promise to infer missing evidence from current reports.

## TD-004: Packaging portability and independent release transition

- [ ] Status: deferred maintenance coordinated with M1.

Packages are host-specific directory distributions; current CI builds Ubuntu/Linux
amd64 packages and uploads workflow artifacts. Windows-native distribution,
additional OS/architecture coverage and published release installation are not
proven by those artifact checks. Separate core and analyzer packaging without
breaking relocation, runtime discovery, legal notices or the distinction between
the analyzer JVM and target JVM. Installer/repository design is owned by
[decision 001](decisions/001-analyzer-distribution.md).
