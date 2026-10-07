# Scan verification — 2026-10-07

Scryer build/test/distribution command:

```sh
JAVA_HOME="$(mise where java)" ./gradlew test installDist
```

Development verification used an ignored local Gradle cache and project-cache directory. **36 tests passed, 0 failures/errors/skips**. The static facts commit was also compiled/tested as an independent staged snapshot before committing.

Installed CLI smoke checks:

| Target | Observed result |
|---|---|
| Existing Java 8 / Gradle 4.10.3 fixture | complete dependency resolution; 65 distinct group/artifact/version components across 17 configurations; source/target 8; selected Spring starters 2.1.18.RELEASE; Mockito 2.23.4 |
| Disposable Gradle 8.14.3 conflict project | JUnit 4.12 request → 4.13.2 selected with conflict reason; Guava 20.0 → 21.0 with forced reason; unique conflict count 1, forced override count 1 |
| Disposable Maven 3.9.9 project | JUnit 4.12 → Hamcrest 1.3, test scope preserved; conflict/override flags and counts null, overall partial collection |

JSON was parsed and checked for declared-versus-selected versions, graph/status/count semantics and absence of ANSI bytes even with forced color. Styled tree output was checked for ANSI styling, tree characters and both conflict/forced annotations. Unit tests check concise summary versus 50 retained JSON declarations, repeated requested versions, color controls, module discovery/boundaries, processor paths, unknowns and scope/relationship preservation.

No new maintained Maven fixture repository was created. Temporary smoke projects and resolver logs are not committed. The original fixture remained clean after scans; no compile/test/package tasks were invoked there by the resolver. Scan reports repository facts and dependency model results, not target build health or test confidence.
