# Scryer

A Kotlin CLI for collecting local Java repository facts. Later, `analyze --before <ref> --after <ref>` will examine change impact and test evidence. This first version implements only `scan`.

## Run

Scryer itself uses Java 21, Kotlin 2.2.21 and Gradle Wrapper 8.14.3. Scanned repositories may use Java 8 or other versions: scan never runs their code. No global Gradle or Kotlin installation is needed.

With your existing mise Java 21 active:

```sh
export JAVA_HOME="$(mise where java)"
./gradlew test installDist
./build/install/scryer/bin/scryer scan ../legacy-java-fixture
```

To invoke it as `scryer scan <path>` from elsewhere:

```sh
export PATH="$PWD/build/install/scryer/bin:$PATH"
scryer scan /path/to/project
```

`--help` prints usage. Exit codes: 0 successful scan (possibly incomplete, with notes); 1 invalid/unreadable repository or metadata; 2 invalid CLI arguments. Quote paths containing spaces. Supply the directory containing the build definition; scan does not search parent directories or all descendant repositories.

## Current output

For the Order Service fixture, the scanner finds:

```text
Build: Gradle 4.10.3 [build.gradle]

Declared dependencies (root only; not a resolved/transitive graph):
  [compile] org.springframework.boot:spring-boot-starter-web  unspecified (possibly managed; not resolved)
  [compile] commons-lang:commons-lang  2.6
  [compile] com.google.guava:guava  20.0
  [testCompile] org.springframework.boot:spring-boot-starter-test  unspecified (possibly managed; not resolved)

Declared build plugins:
  org.springframework.boot  2.1.18.RELEASE
  io.spring.dependency-management  1.0.11.RELEASE
```

Actual output additionally shows declaration kind/source. Spring Boot's plugin version is not presented as the resolved starter version. Unknown or unspecified values remain visible rather than being guessed.

## Supported facts and limits

- Detects `build.gradle`, `build.gradle.kts`, and `pom.xml`. If both tools are present, reports both.
- Reads Gradle distribution version from `gradle/wrapper/gradle-wrapper.properties`, Maven distribution version from `.mvn/wrapper/maven-wrapper.properties`. It does not confuse Maven Wrapper's own version with Maven's version. Missing/unrecognized wrapper version is unknown; an installed global tool version is not repository evidence.
- Gradle: root literal dependencies in common Groovy/Kotlin DSL syntax, legacy `compile`/`testCompile`, string-valued Groovy map notation, platform declarations, simple string variable assignments and `gradle.properties` substitutions. Lists literal versioned plugins separately.
- Maven: root dependency declarations, scopes, project properties, local dependencyManagement entries and versioned build plugins. Parent coordinates are reported as a note. The XML reader rejects DTD/external entity declarations and never fetches XML resources.
- No transitive graph, remote parent/BOM resolution, Maven effective model, Gradle execution/model evaluation, profile activation, version catalog resolution, custom configuration execution or module expansion. Unsupported Gradle dependency statements are shown as notes; detected modules/profiles explicitly indicate incomplete scope. Managed declarations are distinct from actual dependency declarations.
- Static Gradle reading is best effort for the documented forms, not a general Groovy/Kotlin parser. Computed coordinates, triple-quoted strings, complex variable scope/control flow, custom DSLs and applied convention scripts can require a future model adapter. Even a scan without notes does not prove completeness. The output always identifies declarations, not an authoritative resolved dependency list.

## How Kotlin scans a Java project

This slice doesn't parse Java source yet. Kotlin runs on the JVM and uses ordinary filesystem APIs to read build metadata: `Properties` for wrappers, a small literal reader for Gradle DSL, and a JDK XML parser for POMs. The CLI, fact data classes, scanners and terminal presentation are separate enough to add facts without coupling them to output.

Relevant code:

- `Main.kt`: CLI arguments, exit codes and text output.
- `RepositoryScanner.kt`: root validation, tool detection and wrapper versions.
- `Declarations.kt`: dependency/plugin facts and conservative property expansion.
- `GradleDeclarations.kt` / `MavenDeclarations.kt`: local build declarations.

Java symbols/AST/bytecode and state-bound coverage belong to the future analyze engine, not this facts collector.

## Verification and next steps

Tests cover wrapper versions, legacy/Kotlin declarations, properties, unknown expressions, comments/exclusions, explicit unsupported declarations, Maven namespaces/local management/parent limitations, malformed XML/external entities, CLI errors and paths with spaces. The installed CLI is also checked against the real legacy fixture. Tests create tiny temporary build files, not another Maven fixture repository.

Coverage is deferred: a useful confidence report needs the code state, an actual test run and attributable coverage. CI capability inspection is later nice-to-have. Planned `--remote-list` and `-o <OUTPUT_FILE>` Markdown output are not accepted CLI options yet. No plan/apply, recipe orchestration, remote queries or confidence score is included.

Build compatibility reference: [Kotlin Gradle configuration](https://kotlinlang.org/docs/gradle-configure-project.html). Wrapper format references: [Gradle Wrapper](https://docs.gradle.org/8.14.3/userguide/gradle_wrapper.html), [Maven Wrapper](https://maven.apache.org/tools/wrapper/index.html).
