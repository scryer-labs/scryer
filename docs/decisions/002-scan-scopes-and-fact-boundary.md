# Work decision 002: Explicit scan scopes, configuration and fact ownership

Date: 2026-10-31
Status: Agreed direction; CLI/configuration/engine changes are not implemented.
The blanket rule-judgment exclusion below is refined by
[decision 003](003-stack-discovery-and-shared-rules.md): shared reproducible checks
and bounded suggestions may belong to Scryer; consultant-style advice remains outside.

## Context

Orrery now supplies workspace/project context and consumes Scryer facts through
providers. Scryer currently exposes Java scan/analyze commands and Go-only
`analyze workspace` CSV processing. A workspace is a scope, not intrinsically an
analysis operation. Cross-repository scanning needs a reusable explicit scope.

This decision supersedes earlier roadmap ownership of architecture-rule verdicts,
test-effectiveness/strategy judgments and refactoring recommendations in Scryer.
It supplements [decision 001](001-analyzer-distribution.md): validate installation
and a minimal invocation before moving Java. It does not remove existing behavior.

## Scope and portable configuration

- A project represents a repository, potentially with multiple languages and build
  modules. A workspace explicitly groups projects. Cargo/other build workspaces
  remain build structures; they do not automatically define Scryer membership.
- Workspace and project may each have `.scryer/` configuration. Start with handwritten
  files. Workspace configuration retains member paths relative to the workspace
  root, so users do not re-enter scan scope on each invocation.
- Membership has one owner: workspace configuration. Project configuration holds
  its own scan options, not a reverse list of workspace memberships. Project scans
  can use defaults without requiring membership or a project configuration file.
- Support a root that is both workspace and project, with a member path of `.`.
  Do not infer membership by recursively finding repositories. Invalid configured
  members or missing required workspace configuration must produce explicit results,
  not silently substitute a different scan scope.
- Orrery registration and Scryer scan configuration have distinct owners. Future
  explicit Orrery integration may generate/update Scryer entries from registered
  projects, but must expose differences and preserve handwritten options.
  Automatic synchronization/configuration generation is deferred.

Candidate commands for journey design are `scryer project scan <path>` and
`scryer workspace scan <path>`. These examples are not a final grammar. File names,
serialization format, schema, option precedence, path validation, partial-member
policy and command migration must be specified before implementation.

## Fact ownership and source-first collection

Scryer manages installable language engines and produces facts for people, AI and
other tools. Future plugin categories can be considered when concrete needs exist.
Do not build a general plugin framework in anticipation of them.

Source parsing, symbol extraction, resolution and evidence-backed relationship
matching are fact production. Architectural fitness, rule verdicts, test adequacy,
recommended refactoring order and persistent tracking belong to consumers such as
Orrery or a future Hunch-test comparator. Inventory current analyze/matching/plan
outputs before deliberately changing them; retain existing Java evidence during
extraction rather than deleting it based on this direction alone.

Basic source/symbol facts must be available without requiring the target to build
or run. Enhanced resolution can require dependencies, compiler configuration or
build metadata; execution evidence requires explicit capabilities/options. Retain
available facts, unresolved symbols and coverage limitations. Do not advertise a
partial source model as a complete graph. Architecture-pattern signals and explicit
user layer mappings are not proof of original design intent.

Scryer may eventually scan supplied Git history or branch policies, but does not
retain its own scan timeline. Configuration, installed-engine state, caches,
temporary checkouts and explicit output/evidence files are permitted state. Orrery
owns its history identities and persisted context; no engine imports Orrery types.

## JSON and incomplete outcomes

Every public command must provide structured JSON, including install/uninstall/list.
JSON is part of the default result design, not merely an optional renderer for some
scan commands. Decide whether defaults mean JSON stdout or an accompanying file,
and define locations/overwrite behavior before implementation. JSON-mode stdout
contains only machine-readable results; progress and diagnostics go to stderr.

Retain schema/tool/engine versions, input scope/source provenance and supported
coverage. Distinguish unavailable engine, incompatible protocol/version, unsupported
capability, execution failure and partial success in both terminal and JSON results.
Define matching exit codes deliberately; success cannot imply complete collection.
Mixed projects retain supported engine results while exposing missing capabilities.

Relationships retain types and source evidence. Library/package identity, selected
version and configuration matter when matching workspace-local providers. Endpoint
or package-name equality can be a candidate match without proving a connection.
No relationship found is different from a relationship not scanned.

## Engine and platform boundary

`engine` is a provisional category name. Installation validation includes package
identity/version/capabilities, local install/uninstall/list and an actual protocol
request/response. Listing copied files alone does not validate integration. Failed
installation must not appear usable. Start with local packages; defer a public
registry/update service. Then move Java to the user-created `scryer-java-engine`
repository and preserve the Go wrapper/Kotlin worker ownership from decision 001.

macOS is the primary development/acceptance platform. Windows migration for daily
input-method development waits until both Scryer and Orrery stabilize. Existing
Ubuntu CI is retained; Ubuntu-specific IPC expansion is outside the current scope.
Keep platform boundaries clear without adding Windows implementation now.

## Consequences and tracking

CLI, configuration and engine contracts must be agreed through concrete journeys
before renaming commands. Language selection waits until Java extraction and the
Orrery adaptation; C++, Rust, TypeScript, Go and Markdown remain candidates.
Cross-repository relationships prioritize libraries, IPC, then REST.

[Milestones](../MILESTONES.md) is the only delivery-order index, including **2.1**
for the post-extraction Orrery provider update. [CLI](../CLI.md),
[architecture](../../ARCHITECTURE.md) and [current contracts](../contracts/README.md)
remain descriptions of implemented behavior until corresponding code changes land.
