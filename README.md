# Scryer

Kotlin CLI for repository facts and, later, safe-refactor test evidence analysis.

First slice: `scryer scan <path>` recognizes Gradle/Maven build definitions and reads the build tool version from the project's wrapper. It never executes the target project's build scripts. Without a recognized wrapper, the tool version is unknown.

Requires Java 21 to run Scryer; scanned repositories may target older Java versions. No global Gradle or Kotlin installation required.

```sh
export JAVA_HOME="$(mise where java)"
./gradlew test installDist
./build/install/scryer/bin/scryer scan ../legacy-java-fixture
```

Initial scope: local command-line output. Dependency declarations follow in the next small commit. Coverage belongs to state-specific test execution for the later `analyze` capability. CI inspection, `--remote-list` and Markdown `-o` export are future work.
