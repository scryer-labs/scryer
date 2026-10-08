# Scryer

`scan` is a repository facts collector for Java repositories. It reports declarations, evaluated build-model facts and lightweight source signals. It does not recommend upgrades, plan transformations or produce a confidence score. `analyze` currently compares two Git commit snapshots and reports changed files, Java method/symbol changes and potential caller impact.

See [ARCHITECTURE.md](ARCHITECTURE.md) for package responsibilities, the scan flow and code-style guidelines.

GitHub Actions runs `clean`, `test` and `build` with the project Wrapper and Temurin Java 21 on pushes to `main`, pull requests and manual dispatch. The workflow uses a Gradle cache and validates the Wrapper. Executable release packaging is deferred until after the first `analyze` implementation.

Commit subjects follow `<type>: <summary>` with `feat`, `refactor`, `tests`, `docs`, `chore`, `fix` or `revert`; see [AGENTS.md](AGENTS.md).

## Run

Scryer uses Java 21, Kotlin 2.2.21 and its Gradle 8.14.3 Wrapper. Target repositories may use older JDKs and build tools.

Run analyze from inside the target Git repository (or a subdirectory):

```sh
scryer analyze --before HEAD~1 --after HEAD
scryer analyze --before HEAD~1 --after HEAD --skip-tests
scryer analyze --help
```

References resolve to commit SHAs through Git: SHA IDs, branches, tags and expressions such as `HEAD~1` are accepted. The comparison is a direct before-to-after snapshot diff, not a merge-base diff. Working tree changes are excluded. Output lists statuses, before/after paths (including detected renames), and zero-context old/new line ranges. A zero-length range is an insertion/deletion boundary. Binary, mode-only and pure rename changes can have no textual ranges. Rename detection uses Git similarity heuristics.

Both reference arguments are required; invalid arguments exit with code 2. Invalid refs, a non-Git current directory or Git failures exit with code 1. Analysis creates a temporary shared clone and two detached worktrees, reads changed `.java` files with the JDK Java parser, and removes the temporary workspace afterwards. The target repository's working tree and worktree registrations remain unchanged. Git and a full JDK 21 are required.

Methods and constructors are matched by package, enclosing class, name and source parameter types. Output distinguishes ADDED, MODIFIED and DELETED declarations with before/after locations. Overloads and named nested classes are supported. Signature changes appear as deletion plus addition; a pure file rename with identical declarations produces no method changes. AST comparison ignores comments and formatting and includes declaration annotations, return types and method bodies. Test methods are included too; production/test classification comes later.

This is source syntax comparison, not resolved symbol identity or semantic impact analysis. Imports, fields, initializer blocks and inheritance changes are reported as a context-analysis limitation for changed Java files; they can affect unchanged methods. Anonymous classes and ambiguous local-class method identities fail explicitly rather than producing a misleading result. Parse errors also exit with code 1. Parsing uses JDK 21 syntax; unsupported newer syntax, generated sources, submodule contents and Git LFS content are not materialized/analyzed as Java source in this increment. The static phase does not compile or execute target code. The command now follows it with target test execution unless `--skip-tests` is supplied; fresh execution artifacts are collected before cleanup, without impact/evidence matching yet.

Analyze also performs JDK source attribution for both snapshots and constructs **partial** call graphs. Symbol IDs use binary class names, method names and JVM-style erased descriptors, for example `example.Order#total(I)Ljava/math/BigDecimal;`. Changes are mapped to resolved declarations; unresolved mappings are explicitly listed. Reverse reachability includes changed methods and their transitive callers, retaining separate before/after graphs so deleted callers/targets are not lost.

`DIRECT` edges mean the compile-time target, not observed runtime execution. Source overrides are included as `POSSIBLE_DISPATCH`; `super`, static, private and final calls are not expanded. Method references and lambda bodies are potential calls; creating a callback does not prove it executes. Constructors and recursive cycles are supported.

The collector currently includes Java files outside `.git`, `.tooling`, `.gradle`, `build`, `target` and `node_modules`. It does not discover source sets or resolve target dependencies/module classpaths. Missing types/dependencies, external targets and initializer calls are boundaries. Duplicate symbol identities across files are excluded rather than merged. Javac attribution diagnostics indicate incomplete resolution, not target build results. Framework/DI wiring, dynamic reflection, generated code, dynamically loaded/external subclasses and non-method context impact remain unknown. No absence of callers or graph percentage implies safety. JSON/Markdown analyze reports belong to later increments.

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

Future: remote latest-version enrichment (`--remote-list`), richer Maven effective-model provenance, and the analysis behind `analyze --before --after`. Analyze currently implements Git snapshot differences, Java method/symbol changes and partial static caller impact. No modernization strategy or single confidence score is produced.

Implementation references: [Gradle ResolutionResult](https://docs.gradle.org/8.14.3/javadoc/org/gradle/api/artifacts/result/ResolutionResult.html), [Maven dependency tree JSON](https://maven.apache.org/components/plugins-archives/maven-dependency-plugin-3.8.1/tree-mojo.html), [Mordant styling](https://ajalt.github.io/mordant/guide/).

Impact is reported in two layers: reverse-reachable caller impact and potential indirect impact (forward reachability from changed methods and their callers). Indirect candidates may share data/control context; actual behavioral impact is not proven. The terminal draws an indented caller-to-callee graph with CHANGED, CALLER and POTENTIAL INDIRECT markers and source-role labels. All resolved edges inside this scope are drawn, including cycles. Shared nodes use numbered references and recursive edges use CYCLE references; nodes are expanded once to keep the graph finite. Boundary counts attach to their caller nodes; complete boundary details and unknown-caller sites are listed after the graph. This finite graph represents all discovered call chains without enumerating infinitely many recursive paths. There is no depth truncation; unresolved/external edges remain boundaries.

Changed/caller/indirect symbols are labelled PRODUCTION, TEST or UNKNOWN by conventional source paths. Main roots are production; test, integrationTest, integration-test and testFixtures roots are test. Build-model main/test roots now override path conventions; unknown source-set semantics remain UNKNOWN. Tests are retained for future evidence matching; a TEST label or static call does not imply execution.

Literal reflection support tracks local Class/Method bindings inside straight-line method bodies (including try bodies). It recognizes one-argument `Class.forName("fully.qualified.Name")`, `getMethod`/`getDeclaredMethod` with literal method names and explicit class-literal parameters, and subsequent `Method.invoke`. Public lookup includes inherited source methods; declared lookup is restricted to the named class. Exact overloads are matched and reported as REFLECTION edges, with possible source override dispatch retained. No classes are loaded and no reflected code is executed. Access checks, receiver validity and successful runtime execution are not proven.

Dynamic names, parameter arrays, custom class loaders, reflective constructors/fields, reassignment, branch/loop merges, catch/finally/resource bodies and cross-method dataflow remain boundaries. Unsupported control flow invalidates local knowledge rather than inventing an edge. Resolved invoke sites replace their unresolved/external call boundary; other reflection API operations can still remain external boundaries.

Analyze Markdown/Mermaid export remains a later increment; this terminal graph does not add an analyze `-o` flag.

## After test command execution

Analyze now runs a conventional test lifecycle in the isolated after worktree by default, after static analysis. Gradle uses the root Wrapper with `clean test`, `--rerun-tasks` and `--no-build-cache` to prevent cached test outcomes; Maven uses its root Wrapper with `clean verify` so configured integration-test lifecycle phases can run. This is not discovery of every custom test task: Gradle integration/evidence tasks outside `test` are not automatically invoked, and Maven profiles/skips remain as configured. Root POSIX Wrappers are required; no global build-tool fallback is used. Mixed or missing Wrappers are UNAVAILABLE. Custom run-test commands are planned, not implemented.

`--skip-tests` explicitly returns static analysis only and reports SKIPPED. Otherwise the report records after SHA, argument list, target JAVA_HOME, duration, exit code and a durable combined stdout/stderr log. SUCCEEDED means command exit 0, not proof that any test executed, covered a method or that a release package was produced. Failed, timed-out, unavailable or snapshot-mutating commands exit Scryer with code 1, while preserving static impact output. Tracked source/index or HEAD changes invalidate the command result for the requested snapshot.

Scryer itself runs on JDK 21. `SCRYER_JAVA_HOME` selects the target build JVM only; this does not alter global mise configuration. Without an override, Gradle 4–6 can reuse an existing fixture-local Java 8 installation under the source repository's `.tooling/mise/data/installs/java`; otherwise they require an explicit compatible JVM. Other targets default to Scryer's runtime JDK; automatic JVM compatibility selection/provisioning is not yet available.

```bash
SCRYER_JAVA_HOME=/path/to/compatible/jdk scryer analyze --before HEAD~1 --after HEAD
```

`SCRYER_TEST_TIMEOUT_SECONDS` sets a positive per-command timeout (default 600 seconds). `SCRYER_CACHE_HOME` selects build caches and retained logs (default `${java.io.tmpdir}/scryer-cache`). Run logs live under `runs/analyze-*/test.log` and remain after worktrees are removed. Progress goes to stderr. `clean` and build outputs operate inside the temporary after checkout; caches and logs are outside it. Test/plugin code executes normally with the current user's environment; a Git worktree isolates repository files, not arbitrary process side effects.

Next: match method execution evidence to impact; per-test attribution requires separately established coverage scope. No coverage percentage or direct/indirect execution classification is produced in this increment.

## Execution artifact collection

After test execution, analyze retains fresh conventional JUnit XML reports, JaCoCo `.exec`/XML coverage files and compiled production classes alongside the run log. An internal `evidence/manifest.json` retains normalized records and provenance: after SHA, command status, original relative paths and SHA-256 hashes for all copied artifacts, including unreadable ones. This is run metadata, not the planned CLI JSON output flag. Hashes, timestamps and file identities are captured before execution; unchanged pre-existing files are ignored, and retained copies are hash-checked. Data is collected before temporary worktrees are removed. Artifact read errors are reported without changing the command exit result.

Supported roots are module `build/test-results/**/TEST-*.xml`, `target/surefire-reports/TEST-*.xml`, `target/failsafe-reports/TEST-*.xml`, `.exec` files under build/target, Gradle `build/reports/jacoco`, and Maven `target/site/jacoco*` XML reports. Class analysis uses fresh `build/classes/java/main`, legacy `build/classes/main`, or `target/classes` outputs. Source/resource fixtures and cache directories are excluded. Custom output roots, Kotlin outputs and other coverage formats are not discovered yet.

JUnit results preserve each testcase record (including parameterized names, failures, errors and skips); record counts are not unique test counts. JaCoCo method records preserve binary owner/name/descriptor and instruction/branch/line counters. Exec datasets are processed independently using retained classes, with class IDs marked MATCHED, CLASS_ID_MISMATCH or NO_EXECUTION_RECORD. A missing execution record can mean never loaded or instrumentation exclusion. XML data is XML_UNVERIFIED because class IDs cannot be checked from that format. Malformed artifacts remain retained and are reported as unreadable, rather than treated as zero coverage.

No instrumentation is injected: projects without usable configured JaCoCo data report coverage unavailable, not 0%. Command failure can yield partial reports; timeout, skip, unavailable command and changed-snapshot results yield no authoritative evidence. Successful command exit without fresh XML reports leaves test counts unknown.

Coverage scope remains aggregate/unknown. Session IDs and XML report filenames do not establish which individual test executed a production method, and separate datasets are never silently merged. Direct versus indirect execution attribution remains unavailable from aggregate datasets; method impact matching and its explicitly scoped method-execution percentage are described below.

## Analyze module/classpath inputs

Before static attribution, default analyze resolves each snapshot's own build inputs using its Wrapper and target JVM. Gradle's injected metadata task reads Java source-set roots, compile classpaths and project dependencies without compiling or testing. Maven reads the effective POM and test-scope dependency classpaths per reactor module. Downloads may occur; logs are retained under `runs/model-*`. Analyze model commands default to a 180-second timeout each (`SCRYER_RESOLUTION_TIMEOUT_SECONDS`).

Javac attribution is separated per module. Reactor dependency sources support cross-module calls even before class outputs exist; supporting sources do not become duplicate graph nodes. Source-set classpaths are combined within a module in this increment, so contradictory main/test dependency versions are a limitation. Configured main/test roots provide role labels; unknown custom source-set semantics stay UNKNOWN. Independent modules declaring the same binary symbol are explicitly excluded as ambiguous rather than conflated.

Resolution failures preserve partial source analysis and notes. Annotation processors are disabled; generated sources, framework wiring, included builds, exact target-JDK boot APIs and ambiguous reactor artifact IDs are not fully modeled. `--skip-tests` also skips model subprocesses and uses partial source-only attribution. No global Java configuration or original repository files are modified.

## Impact / execution matching

Analyze now matches **after production methods** in changed, reverse-caller and potential-indirect scope against each fresh coverage dataset independently. Identity uses binary owner, method name, erased JVM descriptor and module output provenance. Only class-ID-matched JaCoCo execution records yield EXECUTED, PARTIALLY_EXECUTED or NOT_EXECUTED. Missing class/method records, class-ID mismatch, unverified XML, absent module provenance and methods without instructions remain UNKNOWN. Test/unknown-role symbols are excluded from production method metrics; unresolved changes are listed separately. Deleted before methods have no applicable after coverage.

The report preserves instruction/branch gaps and displays method execution percentage: methods with instruction hits divided by assessable production methods in that dataset. UNKNOWN is excluded and counted visibly. Even 100% here proves neither all paths nor safety; a changed method can have hits while an affected caller remains NOT_EXECUTED. Potential-indirect branches retain their conservative classification.

Static direct/indirect test routes are shown as **candidates**, alongside independent passed JUnit class-record counts. These are not direct/integration execution attribution: aggregate JaCoCo cannot identify which test produced a hit, and a test method or class with passing records may never execute the changed path. HTTP/Spring wiring can produce real hits without a resolved static test route. Coverage scope is never inferred from report filenames or session IDs. Per-test attribution and path execution proof remain unknown until separately collected.

Failed commands can still supply valid partial execution observations, but do not establish a passing build. Skipped/timed-out/unavailable/mutated snapshots and SHA mismatches cannot supply authoritative matching evidence. No single confidence/safety score is computed, and call boundaries remain visible.

Matching smoke validation used all seven legacy-fixture refs: tested change with uncovered caller; integration-only executed change; unused method; misleading test class; added unit/HTTP characterization evidence; compile failure; and executed regression. The first scenario observes 8/11 assessed production methods with hits while retaining quote/discount gaps. Added tests observe 12/12 with hits, with partial branches and unresolved boundaries still disclosed. Compile failure has unavailable coverage, and regression retains failed test outcomes despite instruction hits. A disposable Maven Wrapper project also validates effective custom source roots and JUnit attribution boundaries; without JaCoCo it correctly reports UNKNOWN coverage.

## Analyze reports

Default terminal output now leads with command/test outcomes and method evidence gaps. `--verbose` includes full before/after scope graphs, file ranges, boundaries, artifact provenance, test records and interpretation notes. Concise lists explicitly indicate omitted entries. All report formats use one versioned report model; production/test roles, independent datasets, unknowns and the distinction between method hits and path/test attribution are preserved.
