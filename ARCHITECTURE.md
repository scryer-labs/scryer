# Scryer architecture

Scryer remains a single Gradle module. Packages separate the current scan responsibilities; runtime dependencies, CLI flags and JSON schema are unchanged by this refactor.

| Package | Responsibility |
|---|---|
| `com.edwardnoaland.scryer` | Thin executable entry point |
| `cli` | Command dispatch, command-specific argument parsing, exit codes and service composition |
| `cli.output` | Console sections, styles, dependency tree, JSON output and Markdown report rendering/writing |
| `scan` | `ScanService` coordinates local inspection, optional resolution and enrichment |
| `scan.model` | Facts, declarations, graphs and scan errors; preserves unknown values and provenance |
| `scan.inspect` | Local build declarations, modules, source signals and conventional verification commands |
| `scan.resolve` | Gradle and Maven collectors, Maven tree parsing, child-process lifecycle and target JDK selection |
| `serialization` | Shared Jackson configuration for adapter input and JSON output |

The normal flow is `Main → CLI → ScanService → RepositoryScanner → optional DependencyResolver → enrichment → output`. `--static` stops after local inspection. A collector returns facts; it never prints a report. Progress is an injected callback wired to stderr by the CLI, while JSON/console output goes to stdout. Optional Markdown export renders the same facts without ANSI styling; the file writer uses a temporary sibling file and replacement to avoid truncating an existing report before rendering succeeds.

`DependencyResolver` owns the temporary workspace and chooses the build-tool collector. `GradleModelCollector` and `MavenDependencyCollector` own their respective commands and model formats. `BuildToolProcess` owns environment setup, timeout, logs and process cleanup. Each Maven module can fail independently without discarding successful graphs.

The fact model is shared by collectors and renderers. It does not depend on CLI, file inspection or subprocess execution. It remains a serializable DTO model with the existing Jackson `JsonIgnore` annotation on one convenience property; no separate transport model is introduced yet. Declared and selected versions remain distinct, and unavailable facts are not substituted with zero.

`Cli.kt` owns global help and command dispatch. `ScanCommand.kt` owns scan arguments, service composition and output selection; `AnalyzeCommand.kt` owns analyze arguments and Git comparison output. Each command receives arguments without the command name.

## Code style

- Prefer named arguments when constructing large fact objects.
- Use named intermediate results and small functions to expose meaningful steps.
- Keep lambdas, collection pipelines and sequences when their transformations are easy to follow. Use explicit branches and loops for control flow and side effects when clearer.
- Do not compress multiple statements or nested branches into one line.
- Separate independent concerns rather than creating a class for every function.

This is a package-level design, not a plugin framework or a set of independently published modules. No recipe runner or AI layer is added. Introduce interfaces or further Gradle modules when a concrete replacement, testing need or ownership boundary requires them; `analyze` remains separate from scan's collectors.

Local declaration readers remain best-effort lexical/XML inspection. Moving them into a package does not expand their supported syntax or turn source signals into execution evidence. Verification suggestions retain the existing fixture-specific script convention explicitly.

`analyze.GitComparer` resolves commit refs and collects snapshot file/line differences through Git. It reads the target repository without checkout or mutation; CLI formatting remains in `AnalyzeCommand.kt`.

`analyze.MethodAnalyzer` coordinates isolated snapshots and method matching. `IsolatedSnapshots` owns a temporary shared clone and detached worktrees, with cleanup on success/failure. `JavaMethods` uses JDK compiler parsing only (no compilation, classpath resolution or annotation processing). `GitProcess` owns Git subprocess execution and timeouts. Method comparison keys remain source signatures; the impact layer maps them to resolved erased symbol identities and partial call graphs.

`analyze.ImpactAnalyzer` reuses one isolated workspace for method comparison and before/after call graphs. `CallGraphCollector` uses JDK source attribution with snapshot-local module dependency inputs and without annotation processing, records erased symbol IDs and unresolved boundaries, and conservatively expands source overrides. Reverse reachability is separate for each snapshot. It is a partial static graph, not execution or coverage evidence.

Impact scope has separate reverse caller and forward conservative candidate sets, with an inclusion edge for each indirect candidate. CLI prints all resolved scope edges rather than enumerating recursive paths. Source roles are conventional-path classifications with UNKNOWN fallback; they are independent from impact classification and execution evidence. `LiteralReflection` performs bounded local constant tracking and contributes REFLECTION edges for known invoke targets; unsupported dataflow stays unresolved.

`analyze.AnalyzeService` owns one snapshot lifetime for static impact and optional after execution. `analyze.execute.TestRunPlanner` selects conventional Wrapper commands and a target JVM; `TargetTestRunner` handles process lifecycle, timeout, persistent logs and after-snapshot integrity. These execution results are separate from static graph facts. Target-test command success is not coverage/per-test evidence. Dependency/module classpaths are scheduled before evidence matching; explicit custom test commands replace only the test lifecycle, not model collection.

`analyze.evidence` owns freshness inventory, retained/hash-checked artifacts, JUnit XML parsing and independent JaCoCo datasets. `EvidenceCollector` runs inside the shared snapshot lifetime; its normalized evidence is separate from command results and static impact. `JacocoExecReader` uses JaCoCo core on retained production classes; `EvidenceXml` disables external resource resolution while accepting normal JaCoCo report doctypes. `ExecutionEvidenceRenderer` summarizes collection facts without claiming test attribution or impact confidence.

`analyze.model` collects Wrapper-derived source roots, dependency classpaths and reactor edges per snapshot. `CallGraphCollector` attributes each module with its reachable reactor sources and merges only unambiguous source symbols. Model failures remain partial-analysis notes; scan collectors are independent.

`analyze.match` matches after production impact identities and module output provenance to independently collected coverage datasets. It preserves UNKNOWN, partial instruction/branch hits, uncovered callers, removed symbols and unresolved changes. Static test-route candidates are separate from execution observations; no test attribution or path/safety score is inferred. `ImpactEvidenceRenderer` owns terminal presentation.

`analyze.report` projects analysis into a versioned, format-neutral output model with plain string paths and symbol identities. It retains changed declarations, full resolved impact-scope edges, relevant/unknown-caller boundaries, raw artifacts and independent matching datasets. `cli.output` renderers consume that model; default terminal output prioritizes outcomes/gaps while `--verbose` adds complete details. Presentation does not recompute evidence or collapse UNKNOWN.

Analyze exports select format from `-o` extension and reuse the atomic report writer. JSON serializes the report DTO; Markdown owns escaping/fences and Mermaid rendering. HTML embeds bundled `analyze-report` resources and script-safe JSON into one offline file. Client views filter existing facts; they do not run analyzers, fetch dependencies or reinterpret coverage attribution.

HTML separates report views (`report.js`), pure SCC/layout/visibility operations (`graph-model.js`) and SVG interaction (`graph.js`). The graph uses iterative traversals and keeps every resolved scope edge in the embedded model; view filters/collapse explicitly display visible/total counts. Motion is decorative and respects reduced-motion preferences. No third-party browser assets or frontend build pipeline is required.

Compact display labels belong to `cli.output` and the HTML view layer, never to graph matching or identity. Ambiguous short signatures retain qualified owners; JSON and symbol details keep full identities. Source basenames are display hints, not unique keys. Markdown includes a full symbol/source index. Test-node styling is orthogonal to changed/caller/indirect impact labels.

Future evolution targets Go orchestration with Kotlin Java/Kotlin analysis and separate C++ capabilities. Preserve the current analysis/report boundary. The versioned report DTO can seed a process transport, but a callable protocol should be extracted only with a concrete Go consumer and explicit request, error, lifecycle and compatibility requirements. Reflection improvements and evidence semantics remain analyzer responsibilities, independent of orchestration language.

`scan.remote` enriches direct dependency declarations with Maven Central release metadata. Transport/lookup and Maven version comparison are independent of CLI rendering; timestamped DTO rows live in `scan.model`. Network access is opt-in. Terminal/JSON/Markdown consume the same rows; private repositories and upgrade strategy are outside this collector.

`--test-command` flows from CLI options through `AnalyzeService` to `TestRunPlanner` as one explicit POSIX shell string. `TargetTestRunner` retains the existing environment, timeout, logs and snapshot-integrity validation. Build-model commands remain independently selected; custom commands do not relax freshness or coverage-matching rules.

Reflection resolves exact source overloads from class literals, compile-time string constants, initialized local class arrays, and single-return static no-argument constant helpers. Public/declared constructors are distinct and never inherited. Array mutation or escape into unknown methods/constructors invalidates tracked parameter types. Dynamic names, virtual/helper branching, framework wiring and unsupported control flow stay explicit boundaries.

Maven analysis creates model-only temporary POMs from effective dependency declarations, excludes reactor coordinates, and resolves only external jars. Reachable sibling source modules contribute their external classpaths; no prior reactor install is required. This remains a partial model: profile-only modules, generated roots and ambiguous module association are reported as limitations.

`repository.BuildTool` is the shared explicit selection contract. Scan retains all detected build facts but inspects selected declarations/modules and passes selected builds to dependency/JVM adapters. Analyze carries one selection through both model collectors and the after execution planner. Custom commands replace the test lifecycle only. A selection never silently falls back to another Wrapper; reports retain selection provenance.
