# Work decision 003: Stack discovery, engine auto-install and shared rules

Date: 2026-10-31
Status: Agreed direction; discovery, auto-install and shared rules are not implemented.

## Context and refinement of decision 002

[Decision 002](002-scan-scopes-and-fact-boundary.md) made the consumer boundary too
broad by assigning all rule verdicts and test-effectiveness judgments to consumers.
Scryer may provide reproducible, evidence-backed checks and bounded suggestions.
Business/design trade-offs and consultant-style recommendations remain human or
consumer responsibilities. This refines that ownership statement without adding
history management or changing the CLI/Java extraction delivery order.

## Stack discovery and engine auto-install

Core must recognize basic technology-stack signals without requiring the relevant
language engine to be installed. Start with lightweight configurable detection of
source files, manifests and build declarations. Preserve evidence paths and the
limits of each signal: a manifest indicates a declaration, not a successful build
or proof that a framework is active. Multiple stacks may coexist in one project.
Workspace discovery aggregates results for its explicitly configured members; it
does not discover/register extra projects by walking unrelated repositories.

Keep these capabilities independently callable and compose them in auto-install:

1. Discover project/workspace technology-stack signals.
2. Match required capabilities against the available-engine catalog and installed
   engine inventory, including version/platform/protocol compatibility.
3. Install missing compatible engines and report per-engine results.

Installed inventory and the catalog of available engines are separate concepts.
Initially the catalog can ship with Core; an online marketplace is unnecessary.
Auto-install reuses installation/discovery primitives after their lifecycle and
actual invocation have been verified. Ordinary scans do not implicitly install
software; installation requires an explicit command or option. Exact command names,
matching/version policy, package sources and configuration schema remain open.
Discovery and management results follow the common JSON/partial-result contract.

## Shared rules over language facts

Keep language parsing and semantic extraction in engines. Put reusable rule
algorithms and composition in a shared Core layer rather than duplicating the same
check in each engine repository. Declarative configuration can map language facts
into common concepts and define rule requirements, matching conditions, parameters
and result/evidence projections. JSON is a candidate configuration format, not a
finalized rule schema or a replacement for analysis algorithms.

The fact boundary must extend beyond symbol names: progressively support typed
relations, source locations, type information and, when needed, control/data flow,
test and execution evidence. Engines advertise capabilities; they need not supply
all of these to be usable. Preserve language-specific details and unresolved
boundaries instead of forcing every language into a Java-shaped model.

There are two distinct mappings: language structures to common facts, and common
facts to shared rule results. Language-specific semantics such as C++ virtual calls
or Rust trait dispatch still require language-aware extraction. Configuration cannot
infer missing semantics. Engines remain focused fact producers, not deliberately
incomplete extractors that report every unresolved relation as absent.

For example, a configured forbidden module dependency can reuse one graph rule
across supported engines. Test execution, assertion observations and mutation-test
outcomes can support separately defined checks, but no single observation proves
that a test suite is adequate. Checks must state their scope and evidence.

Each rule declares required capabilities and returns reproducible results with
rule identity/version, parameters, input provenance and supporting source/evidence
locations. Missing capabilities or insufficient evidence produce unsupported or
unknown outcomes, not a passing check. Rule findings remain distinct from raw facts.
Human-readable and JSON views use the same result data.

`suggest` and `check` are candidate command categories. Do not finalize their names
or a general DSL now. Prototype one or two concrete rules with a second language
engine before generalizing the schema. Shared rules remain a later capability;
lightweight stack discovery enters CLI design now, and auto-install follows the
installation lifecycle. No implementation is claimed by this decision.

## Consequences

Scryer can provide deterministic code-smell checks, bounded test-effectiveness
checks and evidence-backed refactoring suggestions when explicit rules support
them. It does not assume responsibility for arbitrary business/design advice,
refactoring execution, persistent scan tracking or Orrery plan orchestration.
Orrery and other consumers may also evaluate facts; they are not required to route
every comparison through Scryer.

[Milestones](../MILESTONES.md) remains the single priority/status index. Java
extraction, post-extraction Orrery adaptation (**2.1**), language selection and the
macOS-first platform agreement remain in effect. Current CLI and engine behavior
remain described by [CLI](../CLI.md) and [contracts](../contracts/README.md).
