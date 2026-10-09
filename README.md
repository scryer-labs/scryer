# Scryer

Scryer uses a Go Cobra CLI and shared reporting layer with a Kotlin analyzer for Java repositories. The installed `scan` and `analyze` commands keep their existing flags and outputs.

## Build the Go + Kotlin application

Prerequisites: Go 1.24 or later, Java 21, Git; the project Gradle Wrapper builds the analyzer.

```sh
./scripts/build
./build/install/scryer/bin/scryer --help
./build/install/scryer/bin/scryer scan ../legacy-maven-fixture --static
```

`go build ./cmd/scryer` builds the frontend only. For development, point `SCRYER_JAVA_CLASSPATH` at the analyzer jars, or use the combined installation produced by `scripts/build`. `SCRYER_ANALYZER_JAVA_HOME` selects the Java 21 runtime for the analyzer; `SCRYER_JAVA_HOME` independently selects the target project's JVM. `--stack java` is optional and currently the only registered stack.

```sh
go test -race ./...
go vet ./...
./gradlew test
./scripts/package
```

Packages contain the native Go executable and Java analyzer jars. Build on the target OS/architecture; these packages do not bundle a JRE. The usual installed path is still `build/install/scryer/bin/scryer`. Running `./gradlew installDist` alone now installs the internal JVM worker, **not** the public CLI; use `scripts/build` for the full application.

See [ARCHITECTURE.md](ARCHITECTURE.md) for layers and [docs/contracts/README.md](docs/contracts/README.md) for analyzer input/output contracts. Kotlin's former CLI/renderers live only in test sources to retain regression checks during the migration. The following sections describe the Java analyzer capabilities.

## Java analyzer capabilities

`scan` is a repository facts collector for Java repositories. It reports declarations, evaluated build-model facts and lightweight source signals. It does not recommend upgrades, plan transformations or produce a confidence score. `analyze` currently compares two Git commit snapshots and reports changed files, Java method/symbol changes and potential caller impact.

See [ARCHITECTURE.md](ARCHITECTURE.md) for package responsibilities, the scan flow and code-style guidelines.

GitHub Actions runs `clean`, `test` and `build` with the project Wrapper and Temurin Java 21 on pushes to `main`, pull requests and manual dispatch. The workflow uses a Gradle cache and validates the Wrapper. After checks pass, CI packages ZIP and TAR distributions, unpacks both and smoke-checks CLI help, a static scan and analyze help. Download `scryer-package-<commit SHA>` from the workflow run's Artifacts; it contains both archives and `SHA256SUMS` and is retained for 14 days. Packages include a native Go executable and JVM analyzer libraries; Java 21 or newer must be installed (the target project's test JVM is configured separately). Packages currently target the build machine's POSIX OS/architecture. Run `bin/scryer` directly; it is a native executable, not a shell script.

To package locally, run `./scripts/package`; archives are created under `build/distributions/`. This CI stage uploads workflow artifacts; it does not publish a GitHub Release.

Commit subjects follow `<type>: <summary>` with `feat`, `refactor`, `tests`, `docs`, `chore`, `fix` or `revert`; see [AGENTS.md](AGENTS.md).

## Run

The Java analyzer uses Java 21, Kotlin 2.2.21 and the Gradle 8.14.3 Wrapper; public commands are provided by Go/Cobra. Target repositories may use older JDKs and build tools.

Run analyze from inside the target Git repository (or a subdirectory):

```sh
scryer analyze --before HEAD~1 --after HEAD
scryer analyze --before HEAD~1 --after HEAD --skip-tests
scryer analyze --help
```

References resolve to commit SHAs through Git: SHA IDs, branches, tags and expressions such as `HEAD~1` are accepted. The comparison is a direct before-to-after snapshot diff, not a merge-base diff. Working tree changes are excluded. Output lists statuses, before/after paths (including detected renames), and zero-context old/new line ranges. A zero-length range is an insertion/deletion boundary. Binary, mode-only and pure rename changes can have no textual ranges. Rename detection uses Git similarity heuristics.

Both reference arguments are required; invalid arguments exit with code 2. Invalid refs, a non-Git current directory or Git failures exit with code 1. Analysis creates a temporary shared clone and two standalone detached checkouts, reads changed `.java` files with the JDK Java parser, and removes the temporary workspace afterwards. The target repository's working tree and worktree registrations remain unchanged. Git and a full JDK 21 are required.

Methods and constructors are matched by package, enclosing class, name and source parameter types. Output distinguishes ADDED, MODIFIED and DELETED declarations with before/after locations. Overloads and named nested classes are supported. Signature changes appear as deletion plus addition; a pure file rename with identical declarations produces no method changes. AST comparison ignores comments and formatting and includes declaration annotations, return types and method bodies. Test methods are included too, with production/test/unknown source roles retained.

Method-difference detection is source syntax comparison; the later attribution/impact phase adds resolved symbol identities and partial static impact. Imports, fields, initializer blocks and inheritance changes are reported as a context-analysis limitation for changed Java files; they can affect unchanged methods. Anonymous classes and ambiguous local-class method identities fail explicitly rather than producing a misleading result. Parse errors also exit with code 1. Parsing uses JDK 21 syntax; unsupported newer syntax, generated sources, submodule contents and Git LFS content are not materialized/analyzed as Java source in this increment. The static phase does not compile or execute target code. The command now follows it with target test execution unless `--skip-tests` is supplied; fresh execution artifacts are collected before cleanup and matched against production impact methods.

Analyze also performs JDK source attribution for both snapshots and constructs **partial** call graphs. Symbol IDs use binary class names, method names and JVM-style erased descriptors, for example `example.Order#total(I)Ljava/math/BigDecimal;`. Changes are mapped to resolved declarations; unresolved mappings are explicitly listed. Reverse reachability includes changed methods and their transitive callers, retaining separate before/after graphs so deleted callers/targets are not lost.

`DIRECT` edges mean the compile-time target, not observed runtime execution. Source overrides are included as `POSSIBLE_DISPATCH`; `super`, static, private and final calls are not expanded. Method references and lambda bodies are potential calls; creating a callback does not prove it executes. Constructors and recursive cycles are supported.

The collector currently includes Java files outside `.git`, `.tooling`, `.gradle`, `build`, `target` and `node_modules`. With build-model execution enabled, it uses discovered module/source roots and target dependency classpaths; source-only fallback remains partial. Missing types/dependencies, external targets and initializer calls are boundaries. Duplicate symbol identities across files are excluded rather than merged. Javac attribution diagnostics indicate incomplete resolution, not target build results. Framework/DI wiring, dynamic reflection, generated code, dynamically loaded/external subclasses and non-method context impact remain unknown. No absence of callers or graph percentage implies safety. JSON, Markdown and HTML exports retain these boundaries.

```sh
export JAVA_HOME="$(mise where java)"
./scripts/build
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

The default view groups Project, Dependencies, Key dependencies (at most five), Testing, Code Characteristics, Verification and collection notes. It does not print the entire transitive graph. `--json` retains all notes and facts; the terminal summary retains collection notes.

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

Remote release enrichment is available through `--remote-list`; richer Maven effective-model provenance remains future work. Analyze currently implements Git snapshot differences, Java method/symbol changes and partial static caller impact. No modernization strategy or single confidence score is produced.

Implementation references: [Gradle ResolutionResult](https://docs.gradle.org/8.14.3/javadoc/org/gradle/api/artifacts/result/ResolutionResult.html), [Maven dependency tree JSON](https://maven.apache.org/components/plugins-archives/maven-dependency-plugin-3.8.1/tree-mojo.html), [Mordant styling](https://ajalt.github.io/mordant/guide/).

Impact is reported in two layers: reverse-reachable caller impact and potential indirect impact (forward reachability from changed methods and their callers). Indirect candidates may share data/control context; actual behavioral impact is not proven. The terminal draws an indented caller-to-callee graph with CHANGED, CALLER and POTENTIAL INDIRECT markers and source-role labels. All resolved edges inside this scope are drawn, including cycles. Shared nodes use numbered references and recursive edges use CYCLE references; nodes are expanded once to keep the graph finite. Boundary counts attach to their caller nodes; complete boundary details and unknown-caller sites are listed after the graph. This finite graph represents all discovered call chains without enumerating infinitely many recursive paths. There is no depth truncation; unresolved/external edges remain boundaries.

Changed/caller/indirect symbols are labelled PRODUCTION, TEST or UNKNOWN by conventional source paths. Main roots are production; test, integrationTest, integration-test and testFixtures roots are test. Build-model main/test roots now override path conventions; unknown source-set semantics remain UNKNOWN. Tests are retained for future evidence matching; a TEST label or static call does not imply execution.

Literal reflection support tracks local Class/Method bindings inside straight-line method bodies (including try bodies). It recognizes one-argument `Class.forName("fully.qualified.Name")`, `getMethod`/`getDeclaredMethod` with constant method names and class-literal parameters (including initialized local arrays), and subsequent `Method.invoke`. Public lookup includes inherited source methods; declared lookup is restricted to the named class. Exact overloads are matched and reported as REFLECTION edges, with possible source override dispatch retained. No classes are loaded and no reflected code is executed. Access checks, receiver validity and successful runtime execution are not proven.

Compile-time string constants and single-return static no-argument helpers returning a constant string/class literal are supported. Exact `getConstructor`/`getDeclaredConstructor` plus `Constructor.newInstance`, and public zero-argument `Class.newInstance`, contribute constructor edges; constructors are never inherited. Mutable parameter arrays lose knowledge after modification or escape to unknown methods/constructors. Dynamic names, arbitrary helper dataflow, custom class loaders, reflective fields, reassignment, branch/loop merges and catch/finally/resource bodies remain boundaries. Unsupported control flow invalidates local knowledge rather than inventing an edge. Resolved invoke sites replace their unresolved/external call boundary; other reflection API operations can still remain external boundaries.

Analyze also exports Markdown/Mermaid and HTML impact maps with `-o`; see the report section below.

## After test command execution

Analyze now runs a conventional test lifecycle in the isolated after checkout by default, after static analysis. Gradle uses the root Wrapper with `clean test`, `--rerun-tasks` and `--no-build-cache` to prevent cached test outcomes; Maven uses its root Wrapper with `clean verify` so configured integration-test lifecycle phases can run. This is not discovery of every custom test task: Gradle integration/evidence tasks outside `test` are not automatically invoked, and Maven profiles/skips remain as configured. The default test lifecycle requires one root POSIX Wrapper and uses no global build-tool fallback. Mixed or missing Wrappers are UNAVAILABLE unless an explicit custom test command is supplied. Use `--test-command` for an explicit custom lifecycle.

`--skip-tests` explicitly returns static analysis only and reports SKIPPED. Otherwise the report records after SHA, argument list, target JAVA_HOME, duration, exit code and a durable combined stdout/stderr log. SUCCEEDED means command exit 0, not proof that any test executed, covered a method or that a release package was produced. Failed, timed-out, unavailable or snapshot-mutating commands exit Scryer with code 1, while preserving static impact output. Tracked source/index or HEAD changes invalidate the command result for the requested snapshot.

Scryer itself runs on JDK 21. `SCRYER_JAVA_HOME` selects the target build JVM only; this does not alter global mise configuration. Without an override, Gradle 4–6 can reuse an existing fixture-local Java 8 installation under the source repository's `.tooling/mise/data/installs/java`; otherwise they require an explicit compatible JVM. Other targets default to Scryer's runtime JDK; automatic JVM compatibility selection/provisioning is not yet available.

```bash
SCRYER_JAVA_HOME=/path/to/compatible/jdk scryer analyze --before HEAD~1 --after HEAD
```

`SCRYER_TEST_TIMEOUT_SECONDS` sets a positive per-command timeout (default 600 seconds). `SCRYER_CACHE_HOME` selects build caches and retained logs (default `${java.io.tmpdir}/scryer-cache`). Run logs live under `runs/analyze-*/test.log` and remain after checkouts are removed. Progress goes to stderr. `clean` and build outputs operate inside the temporary after checkout; caches and logs are outside it. Test/plugin code executes normally with the current user's environment; an isolated checkout isolates repository files, not arbitrary process side effects.

Method execution evidence is matched to impact below; per-test attribution still requires separately established coverage scope. Aggregate hits do not establish direct/indirect test execution or full-path coverage.

## Execution artifact collection

After test execution, analyze retains fresh conventional JUnit XML reports, JaCoCo `.exec`/XML coverage files and compiled production classes alongside the run log. An internal `evidence/manifest.json` retains normalized records and provenance: after SHA, command status, original relative paths and SHA-256 hashes for all copied artifacts, including unreadable ones. This is run metadata, not the planned CLI JSON output flag. Hashes, timestamps and file identities are captured before execution; unchanged pre-existing files are ignored, and retained copies are hash-checked. Data is collected before temporary checkouts are removed. Artifact read errors are reported without changing the command exit result.

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

`analyze --json` writes the complete report to stdout (`schemaVersion: 1`), with progress/errors only on stderr. `-o report.json` saves the same model with atomic file replacement; it can be combined with `--json`. JSON uses plain string paths and symbol identities, arrays for scope nodes/edges, explicit nullable unknown metrics and independent coverage/matching datasets. `--verbose` and `--json` are mutually exclusive. Failed test outcomes still export a report and retain exit code 1; report-writing failure also exits 1.

`-o report.md` exports the complete Markdown report: outcomes, independent dataset evidence tables, changed declarations, before/after Mermaid graphs, boundaries, test records, class-ID provenance and retained artifact hashes. Graphs include every resolved scope edge, cycles and shared nodes, with changed/caller/indirect colors and explicit production/test roles. A Markdown viewer with Mermaid support renders the diagrams; other viewers retain readable graph source. Names are escaped according to [Mermaid flowchart syntax](https://mermaid.js.org/syntax/flowchart.html). Markdown escaping and variable-length source fences prevent repository text from breaking table/code structure. `--json -o report.md` keeps stdout JSON while saving Markdown.

`-o report.html` produces a single offline HTML file with all CSS, JavaScript and report data embedded. It provides dataset selection, method/source search, evidence/impact/role filters, clickable symbol details, before/after changes, execution/artifact tables, and an embedded JSON download. Dataset selection never merges coverage. Before graph symbols do not receive after coverage; test symbols are separate from production metrics. The report uses no CDN, remote fonts or network requests and works directly under `file://`. Artifact paths refer to the generating machine; raw evidence files are not bundled. Repository strings are script-escaped on embedding and rendered with text-only DOM APIs.

HTML's impact map is a local SVG graph with pan/zoom/fit, draggable nodes with live call-edge updates, node selection and evidence details, collapse/expand calls, one-hop neighbor focus and restore. Filtering/collapse counts show visible versus total scope nodes/edges; no graph facts are silently discarded. Shared descendants stay visible through expanded callers, and SCC layout preserves recursive/disconnected components without enumerating infinite paths. Before views show static facts only. Dashed edges identify potential dispatch/reference/reflection; animations never simulate test execution. Dark/light themes, responsive layouts, keyboard controls and `prefers-reduced-motion` are supported. The map uses no graph service or runtime dependency. The Scryer Labs logo is bundled from `internal/report/assets/scryer-logo.png` and embedded as a data URI, so the report remains a single portable file.

```bash
scryer analyze --before HEAD~1 --after HEAD
scryer analyze --before HEAD~1 --after HEAD --verbose
scryer analyze --before HEAD~1 --after HEAD --json -o report.json
scryer analyze --before HEAD~1 --after HEAD -o report.md
scryer analyze --before HEAD~1 --after HEAD -o report.html
```

Graph algorithm checks use Node's built-in test runner: `node --test scripts/test-report-graph.cjs`; CI runs these alongside Kotlin tests and validates browser scripts. Node is needed only for these frontend checks, not to run Scryer or open its reports. The checks cover cycles, shared descendants, collapse/restore, focus/filter subsets, edge kinds and a 12,000-node chain.

## Next increments

Completed: compact symbol/path presentation in HTML and Markdown, with separate purple/dashed test nodes in the HTML impact map. Full identities remain in details and the Markdown symbol index.
Completed: `scan --remote-list` comparisons on stdout, with explicit failures and color support.
Completed: remote comparisons in Markdown.
Completed: explicit custom test commands with provenance, timeout and fresh-evidence handling.

Reflection analysis needs continued work beyond bounded local literal tracking. Prioritize concrete fixture cases and conservative unresolved boundaries; do not treat guessed dynamic targets as resolved calls. Report usability will continue to evolve independently of analysis evidence.

The longer-term direction is Go orchestration and language-specific analyzers: Kotlin for Java/Kotlin, with future C++ inspection and architecture checks. The current Kotlin implementation validates the Java MVP. Keep collection/analysis, execution, report models and rendering separate; the versioned JSON report is a useful starting point for a future process boundary. It is not yet a Go adapter protocol. Define request/version/error/cancellation contracts when the first real Go caller is introduced, rather than migrating analysis logic or adding unused interfaces now.

### Remote dependency release comparison

`scryer scan . --remote-list` queries Maven Central release metadata for direct dependencies only. Selected versions take precedence over declarations; multiple selected versions remain separate rows. Each coordinate is queried once per scan, with four concurrent workers and 5-second connection/10-second request timeouts. `--static --remote-list` skips build-model execution but still performs remote metadata queries. No private repository credentials, mirrors or project repository URLs are consulted. Not-found artifacts, network errors and unresolved current versions remain explicit; partial lookup failures do not fail repository inspection. `--json` includes the enrichment and keeps progress on stderr.

`--color auto|always|never` applies to comparisons (green current, yellow update/ahead, red unknown/unavailable); `NO_COLOR` disables auto coloring. The remote release is the repository's `<release>` field, not a guarantee of stability, Java compatibility or a recommended upgrade. Comparison uses Maven version ordering. Coordinates are sent to Maven Central only when the flag is supplied. See [Maven metadata semantics](https://maven.apache.org/repositories/metadata.html).

Use `scryer scan . --remote-list -o dependency-report.md` to include the same comparison rows, lookup timestamp, metadata URLs and failure notes in Markdown. `--json -o dependency-report.md` keeps stdout JSON while saving the Markdown report. The report contains text statuses without terminal escape sequences.

### Custom analyze test lifecycle

```bash
scryer analyze --before HEAD~1 --after HEAD --test-command './gradlew clean test integrationTest --rerun-tasks --no-build-cache'
scryer analyze --before HEAD~1 --after HEAD --test-command './mvnw -B -Pintegration clean verify' -o report.html
```

The quoted string is passed verbatim to `sh -c` at the isolated after checkout root. It replaces the default test command; Scryer adds no clean/test/task flags. Shell quoting, pipes and environment assignments follow POSIX shell semantics. POSIX support applies to this first version; Windows native execution is not implemented. Build-model collection remains separate and uses conventional Wrappers for before/after; a custom test command does not replace model discovery.

The command inherits the existing target JVM selection (`SCRYER_JAVA_HOME`), build cache environment and timeout (`SCRYER_TEST_TIMEOUT_SECONDS`). Full argv, target Java, exit status, retained log and freshness-checked evidence appear in all analyze report formats. Missing Wrappers no longer prevent an explicitly supplied test command, but model/coverage gaps remain visible. `--skip-tests` and `--test-command` are mutually exclusive.

Choose a lifecycle that actually reruns the intended tests and produces JUnit XML/JaCoCo artifacts under supported conventional output paths. A custom command exiting zero is not proof that tests ran; unchanged artifacts remain excluded. Tracked source/index or HEAD mutation invalidates after evidence. The checkout isolates repository files; shell commands run with your local user permissions and are not a process sandbox.

## Selecting a build tool in dual-build repositories

When both Maven and Gradle are present, choose explicitly:

```sh
scryer scan ../external/spring-petclinic-migration --build-tool maven --dependencies
scryer analyze --before <ref> --after <ref> --build-tool maven
```

`--build-tool gradle` is also supported. Scan lists all detected build tools, but reads modules/dependency declarations and resolves dependencies only for the selected tool. The selection is retained in JSON (`selectedBuildTool`), terminal and Markdown output. `--static` still performs no build commands.

Analyze applies the same selection to both snapshot models and the after test lifecycle. Explicit selections require that tool's Wrapper in both snapshots; it never silently switches tools. `--test-command` still replaces only the test lifecycle, while `--build-tool` selects model collection and target-JVM conventions. With `--skip-tests`, model/test commands remain disabled. Gradle 6 and earlier need a compatible JVM via `SCRYER_JAVA_HOME`; Scryer itself runs on Java 21.

Without a selection, single-build repositories retain automatic detection. Dual-build scan reports ambiguity; dual-Wrapper analyze does not guess a model or default lifecycle. Dependency resolution can still fail independently because of repositories, network access, credentials or target-JVM incompatibility; these remain explicit collection notes.

`--remote-list` groups terminal and Markdown comparisons by module and original declaration scope/configuration. Maven keeps `compile`, `test`, `runtime`, `provided`, etc.; Gradle keeps `implementation`, `testImplementation`, `runtimeOnly` and custom names. The same coordinate can appear in multiple groups, but Maven Central metadata is queried once per group:artifact coordinate.

Current selected versions are matched within the declaration's module and scope. Maven uses direct dependency-tree edge scopes; Gradle uses the actual configuration hierarchy collected from the build (including custom configurations), rather than guessing from names. A Gradle declaration inherited by several resolved configurations may have multiple observed selected versions. If no scoped match is available, the declared version is reported as declared; missing/dynamic declarations stay unknown. JSON retains the original module/configuration fields, and configuration graphs include additive `declarationConfigurations` provenance.

### Analyze another repository or uncommitted changes

```sh
scryer analyze ../external/spring-petclinic-migration --before HEAD --after . --build-tool maven -o report.html
scryer analyze ../external/spring-petclinic-migration --before HEAD~1 --after HEAD
```

The optional path defaults to the current directory. Git refs are resolved in the selected repository. `--after .` captures its current files, including staged and unstaged changes, deletions and non-ignored untracked files. If a staged file also has unstaged edits, the current file content wins. Ignored untracked caches/build products are excluded; tracked/staged files remain included even when an ignore rule matches them.

Go uses a private clone/index and private snapshot commit, then analyzes and runs tests in isolated checkouts. The original HEAD, index, files and worktree registrations are untouched. JSON includes `afterSource` (`kind: WORKING_TREE`, `baseHead`, `snapshotSha`); terminal/Markdown/HTML label the after state as a working-tree snapshot. Execution and coverage refer to this immutable snapshot SHA, not to the original HEAD. Snapshot commits are never created in the original repository. Capture checks for concurrent changes and asks you to retry when detected; avoid editing files while capture is in progress. Working-tree snapshots with submodules are currently rejected; use committed refs instead.

Snapshots use standalone Git checkouts with normal `.git` directories, sharing objects within the temporary workspace. This supports legacy embedded Git clients such as the old JGit bundled with `git-commit-id-plugin`; linked Git worktree metadata is not exposed to target build plugins.

### Terminal colors

Both `scan` and `analyze` accept `--color auto|always|never` (default `auto`). Automatic colors are disabled for redirected output, `NO_COLOR` (including an empty value), and `TERM=dumb`. Explicit `always` overrides those settings. JSON and Markdown/HTML exports never receive terminal ANSI styling.

Section headings are bold cyan; successful commands/test records and execution hits are green; failures and missing execution hits are red; partial/unknown evidence and analysis boundaries are yellow. Zero-count status labels are dimmed. In verbose impact graphs, changed production symbols are emphasized red, callers blue, potential indirect impacts yellow, and test symbols purple. These colors describe evidence/status, not a refactor safety score. Remote dependency comparisons retain green for current versions, yellow for available updates/current-ahead, and red for unavailable/unknown comparisons.
