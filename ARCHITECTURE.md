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

This is a package-level design, not a plugin framework or a set of independently published modules. No analyzer, recipe runner or AI layer is added. Introduce interfaces or further Gradle modules when a concrete replacement, testing need or ownership boundary requires them; future `analyze` should not be forced into scan's collectors.

Local declaration readers remain best-effort lexical/XML inspection. Moving them into a package does not expand their supported syntax or turn source signals into execution evidence. Verification suggestions retain the existing fixture-specific script convention explicitly.

`analyze.GitComparer` resolves commit refs and collects snapshot file/line differences through Git. It reads the target repository without checkout or mutation; CLI formatting remains in `AnalyzeCommand.kt`.
