# Product direction and engine handoff

Agreed 2026-10-27. Planned boundaries, not additional implemented CLI commands or
an installer. [Milestones](MILESTONES.md) owns priority/status; [CLI](CLI.md) and
[architecture](../ARCHITECTURE.md) describe current behavior.

## Independent facts tool

Scryer is an independent, lightweight code-fact discovery and analysis tool. Given
a repository, snapshots or supported workspace inputs, it produces structured facts
and human-readable reports. It must remain useful without Orrery. Orrery preserves,
organizes, visualizes and acts on those facts in a persistent workspace context.

Scryer must not become the owner of Orrery scan IDs, persistent scan timelines,
workspace registration, plan/run history or UI. Being stateless with respect to
that engineering context does not forbid installed-engine settings, temporary
analysis checkouts, output files or regenerable caches. Existing retained evidence
artifacts also remain valid; this decision does not delete or reinterpret them.
Operation/request IDs and source revision provenance are not Orrery's history IDs.

Scryer owns bounded extraction/analysis and architecture-rule evaluation. Engines
own language/build-specific implementation. Shared command/report contracts and
presentation remain outside language collectors. Engine and Orrery provider
interfaces need not be the same: independent contracts serve different consumers.

## Next development sequence

Orrery first completes its bounded foundation: usable local execution/providers,
parameters/results/reports, retry and workspace access. It can then run development
checks for Scryer while primary implementation returns here. Orrery's official
Scryer provider consumes public CLI/fact contracts; no engine imports Orrery types.

1. Review public CLI semantics before moving the Java engine. Define scan versus
   analyze, project/workspace scope, input/ref options, stdout versus output-file
   formats, diagnostics/progress, exit/partial-result semantics and compatibility.
   Write concrete examples and contract tests before renaming commands/flags.
   Do not invent the new grammar in this planning update.
2. Establish the smallest external engine boundary from existing process/contracts,
   then extract the Java engine and its runtime packaging. Preserve existing Java
   capabilities and report evidence; keep Core usable without a JVM for non-Java
   operations. Java runtime requirements belong to the optional Java engine.
3. Update the Orrery Scryer provider against the agreed CLI/version/capability
   contract and validate the integration in actual Scryer development.
4. Prototype C++, Rust or a Tauri/mixed-stack slice and choose the first by measured
   effort and usefulness on a real repository. Repository/package names such as
   `scryer-java-engine` or `scryer-lang-java` are candidates, not selected names.
   Do not promise all three complete semantic engines at once. Tauri composes Rust,
   frontend and configuration facts plus supported bridge relations; it is not a
   standalone source language.
5. Add Go and Markdown capabilities incrementally afterward. Go package/build facts
   and Markdown document/link facts have distinct semantics. Explicit unknowns and
   unsupported operations are preferable to a Java-shaped universal model.

The Java extraction/distribution decision remains authoritative on package/runtime
constraints; this adds the CLI review and real-consumer validation before/around it.
No repository, command rename, install mechanism or new engine is created here.

## Architecture facts and the consumer loop

A valuable first architecture slice is a small real C++ repository: source →
versioned architecture JSON → Orrery persisted scan → interactive project diagram
→ source/evidence navigation. Return to Orrery for that UI after usable project
facts exist; do not require every language or IPC relationship before the first view.

Facts should retain typed entities/edges, source locations and relation evidence,
coverage/unknowns, source state and engine/schema/tool provenance. Keep observed
facts, explicit user architecture mappings/constraints and inferred relationships
separate. Static/shared linking, package dependencies, calls, IPC and REST are not
one undifferentiated dependency relation. Nodes may have several evidence locations.
Dynamic routing, dispatch and cross-process matching need supported pattern scopes;
static analysis cannot claim exhaustive runtime behavior.

Orrery owns durable history, code-state associations, user display-name overrides,
queries and interactive rendering. It may compare saved snapshots; Scryer can
analyze explicitly supplied before/after inputs without maintaining that timeline.
Web/nvim/TUI should consume Orrery's shared context, not independently duplicate
history or call engines behind the backend's back. Diagrams/reports derive from the
same facts rather than becoming competing analysis sources.

Architecture-guard DSL, richer cross-repository communication and historical
architecture evolution follow bounded fact extraction. Rule expectations are not
observed facts; missing evidence cannot become a passing architecture test.
