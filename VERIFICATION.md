# Scan verification — 2026-10-08

Scryer build/test/distribution command:

```sh
JAVA_HOME="$(mise where java)" ./gradlew test installDist
```

Development verification used an ignored local Gradle cache and project-cache directory. **47 tests passed, 0 failures/errors/skips**. The static facts commit was also compiled/tested as an independent staged snapshot before committing.

Installed CLI smoke checks:

| Target | Observed result |
|---|---|
| Existing Java 8 / Gradle 4.10.3 fixture | complete dependency resolution; 65 distinct group/artifact/version components across 17 configurations; source/target 8; selected Spring starters 2.1.18.RELEASE; Mockito 2.23.4 |
| Disposable Gradle 8.14.3 conflict project | JUnit 4.12 request → 4.13.2 selected with conflict reason; Guava 20.0 → 21.0 with forced reason; unique conflict count 1, forced override count 1 |
| Disposable Maven 3.9.9 project | JUnit 4.12 → Hamcrest 1.3, test scope preserved; conflict/override flags and counts null, overall partial collection |
| Existing Spring Petclinic / Maven 3.6.3 | 19 observed direct dependencies; 152 resolved components, 133 transitive-only; Spring starters 2.4.1, Mockito 3.6.28; partial because conflict/override reasons are unavailable; warm scan 1.38 seconds |

JSON was parsed and checked for declared-versus-selected versions, graph/status/count semantics and absence of ANSI bytes even with forced color. Styled tree output was checked for ANSI styling, tree characters and both conflict/forced annotations. Unit tests check concise summary versus 50 retained JSON declarations, repeated requested versions, color controls, module discovery/boundaries, processor paths, unknowns and scope/relationship preservation.

No new maintained Maven fixture repository was created. Temporary smoke projects and resolver logs are not committed. The original fixture remained clean after scans; no compile/test/package tasks were invoked there by the resolver. Scan reports repository facts and dependency model results, not target build health or test confidence.

Regression tests exercise a failed Maven Wrapper and a timed-out subprocess. Both retain actionable errors in JSON/summary; failed collection no longer reports successful tree collection. The timeout is configurable, and progress uses stderr so JSON stdout remains parseable.

Petclinic was also scanned using a fresh, empty dependency cache. Collection completed in approximately six minutes (beyond the old 120-second cutoff), with the same 152 components and no failed modules. Its POM queries Spring snapshot/milestone repositories before Central, and first-time descriptor/plugin downloads dominate this run. No repository order or mirrors were rewritten. Both cold and warm JSON outputs were parsed and checked for counts/status and module failures. Petclinic's working tree remained clean.

## Package and readability refactor

The package split and extracted scan service/collectors were verified with all 38 tests and `installDist`. Before/after JSON was compared structurally for the legacy fixture's static scan and Petclinic's resolved scan; both remained identical, including schema, declarations, notes and graphs. The extracted Gradle collector was also exercised against the legacy fixture: complete resolution, 65 components and 17 configurations. Target working trees remained clean. Architecture responsibilities and code-style guidelines are recorded in `ARCHITECTURE.md`.

## Markdown export and external repositories

Seven export tests cover UTF-8/spaced paths, parent directory creation, replacement, invalid arguments, output errors, ANSI-free Markdown, complete notes/escaping, optional tree output and JSON stdout combined with Markdown export. Two additional tests verify that Jupiter imports do not invent a major version and resolved Jupiter 6 is not labeled JUnit 5.

External checks used shallow clones under the sibling `external` directory:

| Project/revision | Results |
|---|---|
| Resilience4j `b7c0069802e373ed72dff0b817c270fbb6e966ba` | Gradle 9.4.1; 30 projects, 410 configurations, 542 resolved components; complete, no failures; 1,241 tracked Java files; warm scan 9.28 seconds |
| MyBatis `34147d0bd2f5ed0016ef367c92b25e174149326b` | Maven 3.9.16; 59 components; no failed modules; partial because Maven selection reasons are unavailable; Mockito 5.23.0, Jupiter 6.1.3; 1,396 tracked Java files; warm scan 2.47 seconds |

The Gradle 9 check exposed an internal-project classification bug. The collector now uses `ProjectDependency` rather than the removed property; a separate native two-module Gradle 9 model distinguishes an internal project from an external library with the same group. Resilience4j's corrected external dependency count is 510 module/coordinate pairs; internal project declarations remain in JSON but are not counted as external libraries.

Both static and resolved JSON/Markdown reports were generated outside the target checkouts, parsed/checked, and compared with tracked source counts and selected dependencies. Both checkouts remained clean. Reports and the reproducibility summary are in `../external/scan-reports`; they are not bundled into Scryer. Static version-catalog and Maven effective compiler-model limitations remain explicit.

## CI configuration

The GitHub Actions workflow runs separate `clean`, `test` and `build` steps on Ubuntu with Temurin 21 and the repository Wrapper. Official actions are pinned to verified commit SHAs. The workflow was checked with actionlint 1.7.12, and the same task sequence was executed locally: all 47 tests passed and the build succeeded. A hosted GitHub run has not yet been verified. No executable release/publishing stage is configured; Gradle's existing standard build outputs are unchanged.
