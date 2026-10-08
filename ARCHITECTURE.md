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

`analyze.AnalyzeService` owns one snapshot lifetime for static impact and optional after execution. `analyze.execute.TestRunPlanner` selects conventional Wrapper commands and a target JVM; `TargetTestRunner` handles process lifecycle, timeout, persistent logs and after-snapshot integrity. These execution results are separate from static graph facts. Target-test command success is not coverage/per-test evidence. Dependency/module classpaths are scheduled before evidence matching; custom test commands are a later extension.

`analyze.evidence` owns freshness inventory, retained/hash-checked artifacts, JUnit XML parsing and independent JaCoCo datasets. `EvidenceCollector` runs inside the shared snapshot lifetime; its normalized evidence is separate from command results and static impact. `JacocoExecReader` uses JaCoCo core on retained production classes; `EvidenceXml` disables external resource resolution while accepting normal JaCoCo report doctypes. `ExecutionEvidenceRenderer` summarizes collection facts without claiming test attribution or impact confidence.

`analyze.model` collects Wrapper-derived source roots, dependency classpaths and reactor edges per snapshot. `CallGraphCollector` attributes each module with its reachable reactor sources and merges only unambiguous source symbols. Model failures remain partial-analysis notes; scan collectors are independent.

`analyze.match` matches after production impact identities and module output provenance to independently collected coverage datasets. It preserves UNKNOWN, partial instruction/branch hits, uncovered callers, removed symbols and unresolved changes. Static test-route candidates are separate from execution observations; no test attribution or path/safety score is inferred. `ImpactEvidenceRenderer` owns terminal presentation.
