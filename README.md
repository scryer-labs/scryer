# Scryer

`scan` is a repository facts collector for Java repositories. It reports declarations, evaluated build-model facts and lightweight source signals. It does not recommend upgrades, plan transformations or produce a confidence score. `analyze` currently exposes the initial command placeholder only.

See [ARCHITECTURE.md](ARCHITECTURE.md) for package responsibilities, the scan flow and code-style guidelines.

GitHub Actions runs `clean`, `test` and `build` with the project Wrapper and Temurin Java 21 on pushes to `main`, pull requests and manual dispatch. The workflow uses a Gradle cache and validates the Wrapper. Executable release packaging is deferred until after the first `analyze` implementation.

Commit subjects follow `<type>: <summary>` with `feat`, `refactor`, `tests`, `docs`, `chore`, `fix` or `revert`; see [AGENTS.md](AGENTS.md).

## Run

Scryer uses Java 21, Kotlin 2.2.21 and its Gradle 8.14.3 Wrapper. Target repositories may use older JDKs and build tools.

The first analyze increment accepts the command shape:

```sh
scryer analyze --before HEAD~1 --after HEAD
scryer analyze --help
```

A valid invocation currently prints only `Analyzing…` and exits successfully. Both nonblank reference arguments are required; duplicate/unknown options or missing values exit with code 2. References are not yet resolved, and no Git comparison, checkout, build or test execution occurs. JSON/Markdown analyze reports belong to later increments.

```sh
export JAVA_HOME="$(mise where java)"
./gradlew test installDist
export PATH="$PWD/build/install/scryer/bin:$PATH"
scryer scan ../legacy-java-fixture
```

Default scan shows a concise summary, attempts dependency resolution with the target repository's Wrapper, and retains the full model internally. This evaluates target build configuration and may download dependency metadata/plugins; it does **not** compile, test or package the target. Arbitrary build configuration code can have its own side effects. Use `--static` to inspect local declarations/source signals without executing target configuration or downloading anything.

```sh
scryer scan . --dependencies        # all observed direct declarations, grouped by module/configuration
scryer scan . --dependency-tree     # resolved graph per module/configuration, rendered as a tree
scryer scan . --json                # complete versioned facts model, no presentation truncation
scryer scan . --static              # local facts only; resolution is not_requested
scryer scan . --color always        # force styled output
scryer scan . -o reports/scan.md    # also write a Markdown report
scryer scan . --json -o scan.md     # JSON stdout plus a Markdown file
```

`-o <OUTPUT_FILE>` (alias `--output`) writes UTF-8 Markdown, creates missing parent directories and replaces an existing report. The terminal/JSON output is retained; the file confirmation goes to stderr. Markdown includes modules, all observed direct dependencies with declared/selected versions, source/test facts, suggested commands and all collection notes/failures. `--dependency-tree` additionally includes a plain-text tree in the file. Reports have no terminal color codes. Collection status and unknown facts remain explicit; exporting a report does not verify the target build or tests.

Flags may be combined. JSON always remains plain machine-readable JSON even with `--color always`; dependency/tree view flags do not trim JSON. `--color auto|always|never` defaults to auto, which enables color only for an interactive stdout and honors `NO_COLOR`. Explicit always overrides automatic detection. Styling uses Mordant, with cyan/bold headings, dim metadata, green complete/configured values, yellow signals/conflicts/partial collection and red unknown/unresolved values. Shared graph nodes use `[already shown]` references; JSON retains every edge.

Quote paths with spaces. The supplied path must contain a recognized build/settings definition; scan does not search arbitrary ancestor repositories. Exit codes: 0 facts collected (possibly partial); 1 invalid/unreadable input; 2 invalid CLI arguments. Check `resolution.status`/notes rather than treating exit 0 as full resolution or a passing target build.

## Summary and count semantics

The default view groups Project, Dependencies, Key dependencies (at most five), Testing, Code Characteristics, Verification and collection notes. It does not print the entire transitive graph. `--json` retains all notes and facts; the terminal summary shows at most four notes.

For the legacy fixture, scan observes Java source/target 8, Gradle 4.10.3, one module, Spring Boot plugin 2.1.18.RELEASE, JUnit 4, JaCoCo configuration, javax usage and reflection. Actual resolved Spring starters are 2.1.18.RELEASE and Mockito is 2.23.4.

- Direct (observed): unique module/group/artifact declarations across configurations, including dependencies added by evaluated build configuration. Dependency detail retains each configuration and declaration expression. Managed-only entries are not counted as direct dependencies; explicitly declared platforms are.
- Resolved components: unique group/artifact/version across collected configurations. Includes build-tool configurations such as JaCoCo, not only application runtime.
- Transitive-only: resolved component identities not selected as a direct root edge in any collected configuration. A library may be direct in one configuration and transitive in another; per-edge `direct` records that distinction. Counts of declarations, selected component versions and graph edges are different measures and need not sum.
- Conflict selections / forced overrides: unique module/selected-component with the corresponding Gradle selection reason, deduplicated across configurations. Each edge's requested/selected versions and reasons remain in JSON. A different selected version is not automatically called a conflict or forced override.
- Empty/unavailable collection uses null/unknown, not invented zero counts. Partial counts describe observed configurations, not proven completeness.
- Testing counts are statically classified **source files**, not JUnit method counts, tests run, assertion quality or confidence. Integration classification uses conventional integration roots and `*IntegrationTest` / `*IT` filenames. Custom test task/root metadata is also retained from the evaluated Gradle model.
- Jupiter imports identify the framework without establishing its major version. Static mode reports `JUnit Jupiter`; resolved versions refine it, for example `JUnit Jupiter 6`. Framework signals can include transitive test libraries and are not proof that tests executed.

## Full facts model

`RepositoryFacts` is separate from `ScanRenderer`; JSON exports schemaVersion 1 and all collector data, not terminal strings. The model includes:

- Build definitions, wrapper presence/version, modules, parent POM coordinates, declared plugins, repository signals and custom build files.
- Declared/evaluated Java source, target and toolchain versions; preview-feature signal; Java/Kotlin/Groovy source presence; JPMS descriptor paths.
- Dependency group/artifact, original declared version/expression, expanded local version, selected versions, module/configuration, declaration kind, local management provenance and raw declaration.
- Evaluated Gradle projects, source sets, test tasks, repositories, plugin implementation classes, configured direct dependencies and management-plugin sources.
- Per-configuration nodes and edges: requested versions, selected versions, parent/child relationships, direct/transitive context, constraint flag, selection descriptions, conflict/forced flags and unresolved failures. Maven nodes additionally preserve type/classifier/optional and edges preserve scope.
- Framework/testing signals, Mockito/AssertJ/Hamcrest coordinates, source layouts, existing generated roots, ignored/disabled annotation signals, JaCoCo configuration and existing report paths.
- Annotation-processor declarations (including Maven compiler paths), Lombok/codegen signals, reflection, Class.forName, class loading, ServiceLoader, proxy annotations, JNI, serialization and bytecode-library signals with source locations.
- Build/test/package command conventions, CI filenames and quality-tool signals.

Signals are lexical observations, not AST analysis, proof of framework activation or runtime execution. Commands are inferred conventions, not verified build/test evidence. Report paths are existing files, not coverage percentages or evidence bound to the current revision. Generated roots distinguish existing directories from configured Gradle source-set metadata; scan does not run code generation.

## Build-model adapters and limits

### Gradle

A bundled init script adds one standalone model task using Gradle's `ResolutionResult`. It evaluates all projects and attempts each `canBeResolved` configuration, recording individual failures. It does not add dependencies on compile/test tasks or edit target source/build files. Project caches use a temporary directory; dependency caches use an existing target `.tooling/gradle` when available, otherwise `SCRYER_CACHE_HOME` or a cache below the system temporary directory.

Gradle dependencies can be computed by arbitrary code, so static DSL parsing remains a best-effort fallback. Literal Groovy/Kotlin declarations, map notation, properties, local variables, platforms and simple module/projectDir declarations are supported. Evaluated projects/declarations can fill gaps such as version catalogs/custom source sets. Composite builds and remote convention/plugin/buildscript dependency graphs are not fully modeled. Selection reasons and the management plugin are retained, but Gradle does not always expose which exact BOM entry produced a version; the collector does not invent it.

Scryer runs on Java 21. For an older target Wrapper, set a compatible JVM for the child process:

```sh
SCRYER_JAVA_HOME=/path/to/jdk8 scryer scan /path/to/legacy-project
```

For the existing fixture, a project-local Java 8 under `.tooling/mise/data/installs/java` is auto-detected. A toolchain declaration is distinct from the JVM used to evaluate the build. Missing/incompatible target JVM, timeout or resolution failure produces unavailable/partial facts with notes. Global Java settings are not changed.

### Maven

Static scanning reads reactor modules, root/module properties, dependencies, local dependencyManagement, parent coordinates and plugins. XML DTD/external entity declarations are rejected.

With `mvnw`, the adapter invokes pinned `maven-dependency-plugin:3.8.1:tree` in JSON mode for each module, preserving selected tree relationships/scopes. Compilation/tests are not run. Dependency and Wrapper caches stay below `SCRYER_CACHE_HOME`/the system temporary cache unless the user supplied `MAVEN_USER_HOME`. Modules that cannot resolve are retained as failed graphs; already-collected graphs are preserved.

Maven's JSON tree does not provide authoritative omitted conflict candidates, override reasons or exact effective BOM/parent provenance. Those fields/counts are **unavailable**, and Maven resolution is marked **partial**, even when the selected tree succeeds. Requested root versions are filled only when supported by local declaration/management facts. Reactor dependencies not installed/resolvable independently, remote parent properties, activated profiles and effective compiler configuration can need a later richer Maven adapter. Static source-version hints and evaluated Gradle defaults have different provenance and should not be confused.

Neither adapter silently falls back to a global Maven/Gradle installation. Each model subprocess has a 600-second timeout, configurable with `SCRYER_RESOLUTION_TIMEOUT_SECONDS` (positive seconds). Cold Maven scans can download hundreds of dependency descriptors and query every repository declared by the project. Progress goes to stderr every 15 seconds, keeping JSON stdout clean. Failed modules display their error and the last build-tool output; a failed scan does not claim that trees were collected. Successful downloads are cached for subsequent scans. Repository order, mirrors and dependency declarations are not changed to speed up collection. The JSON graph is selected-model evidence, not artifact compilation or build health.

## Review / verification

Small commits separate static model/topology collection from build-model adapters and presentation. Tests cover existing declaration behavior, module discovery/remaps/boundaries, source/test/code signals, raw versus managed versions, null versus zero, concise summary/full JSON, flags/colors, graph cycles/shared nodes, scopes and conflict rendering.

Real integration smoke checks use:

- The healthy Java 8 / Gradle 4.10.3 legacy fixture: 65 resolved components in the current validation environment, actual starter/Mockito versions, valid full JSON.
- A disposable Gradle 8.14.3 project: JUnit 4.12 requested → 4.13.2 selected by conflict, Guava 20.0 → 21.0 forced; reasons distinguish both.
- A disposable Maven 3.9.9 project: JUnit 4.12 → Hamcrest 1.3 with test scope; missing conflict reasons remain null.

These smoke projects are temporary checks, not a new maintained Maven fixture. No source changes are made to the legacy fixture by Scryer; target configuration itself remains ordinary executable build code.

Future: remote latest-version enrichment (`--remote-list`), richer Maven effective-model provenance, and the analysis behind `analyze --before --after`. Only the analyze command placeholder is currently implemented. No modernization strategy or single confidence score is produced.

Implementation references: [Gradle ResolutionResult](https://docs.gradle.org/8.14.3/javadoc/org/gradle/api/artifacts/result/ResolutionResult.html), [Maven dependency tree JSON](https://maven.apache.org/components/plugins-archives/maven-dependency-plugin-3.8.1/tree-mojo.html), [Mordant styling](https://ajalt.github.io/mordant/guide/).
