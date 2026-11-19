# Change impact projection

Core prepares immutable before and after checkouts and passes their paths and
commit identities to an engine declaring the `impact` operation. The JSON
request uses `impact` with `before`, `after`, `beforeRoot`, `afterRoot`, `repository` (original checkout),
and `resolve`.
The C ABI and JSON protocol version remain 1.

The engine returns `schemaVersion`, `beforeSha`, `afterSha`, `before` and
`after`. Each snapshot contains:

- `symbols[]` with opaque `id`, correspondence `key`, display `signature`,
  content `fingerprint`, source text, and `location`.
- `calls[]` with optional evidence `kind` (`call`, `declaration-reference`,
  `type-reference`), `callSiteId`, optional `callerSymbolId`, `callee`,
  optional `candidateSymbolId`, `resolution`, and `site`.
- `contexts[]` with source `path` and a fingerprint of tokens outside
  function definitions.
- `notes[]` with collection limits.

Locations have `file` and one-based `start` and exclusive `end` line/byte-column
coordinates. Files are relative to their respective snapshot roots. IDs only
join within a snapshot. Engine correspondence keys pair declarations across
snapshots and remain internal to that engine's versioned observations.

## C++ source evidence

C++ uses Tree-sitter function definitions and initialized variable-template
declarations. Variable templates appear as `kind: "variable-template"` in the
shared scan symbol table and can own compiler-instantiated expressions.
Named function definitions have optional `executionKind: "compile-time"` only
for an explicit `consteval` specifier. Variable-template nodes carry
`executionKind: "declaration"`; `constexpr` functions remain runtime-capable.

For function definitions, named enclosing classes/namespaces
and declarator tokens contribute to a file-local correspondence key. Whole
function token fingerprints identify body or declaration changes. Comments
and whitespace do not change token fingerprints. A file rename or declarator
change is reported as removal and addition. Duplicate keys stay ambiguous.
Lambdas, anonymous scopes and malformed definitions have no correspondence key.

Static call candidates are unique textual name matches from the shared scan
call graph. `NAME_CANDIDATE` edges express potential relationships. They carry
the call-site ID and location, and do not establish overload or type binding.
Unavailable source targets remain boundaries. Non-function context changes
appear in `unresolvedChanges`, since their effect on functions is unknown.

## Core comparison and traversal

Core compares fingerprints under unambiguous keys and produces added,
modified and deleted declarations. It computes reverse reachability from
changed declarations for `CALLER` nodes. Forward reachability from the
changed declarations supplies `POTENTIAL_INDIRECT` nodes. Changed nodes take
precedence over either reachability class. Both snapshot graphs are retained,
including cycles and source boundaries.

The report uses the current analyze envelope and JSON, Markdown and offline
HTML renderers. Changed-file rows summarize declaration changes. Node roles
come from engine evidence, defaulting to `UNKNOWN`. Compiler-proven C++ test
registrations supply `TEST` and their `testContext`; Core does not infer roles
from source spelling. See the [classification contract](test-classification.md).
The HTML graph groups evidence by ordered caller/callee pair and shows relation
types and record counts. Selecting an edge or its label opens the original
source records. Grouping changes presentation only; JSON retains all edges.

## Mode selection and after coverage

Without `--resolve`, test targets or binary paths, impact uses the static scan path.
Selecting `--resolve`, `--test-target` or `--test-binary` sets `impact.resolve: true`.
The engine calls the same scan path with compiler resolution for each
snapshot. A skipped or partial resolution is an error. Compiler-confirmed
definitions use `targetSymbolId` and yield `COMPILER_RESOLVED` edges.
Missing compiler targets stay unresolved without textual candidate fallback.
Resolution details are retained under the report's `extensions`.

The shared scan joins cross-translation-unit declarations to definitions
by libclang USR. Ambiguous definitions are left unresolved. The shared scan
adds local declaration/type references under `semantic.references[]`, separately
from `semantic.edges[]` calls. Impact preserves them as
`COMPILER_DECLARATION_REFERENCE` and `COMPILER_TYPE_REFERENCE` relations.
A variable-template source node aggregates references from its observed
instantiations. This conservative dependency union does not preserve a unique
per-instantiation path or assert runtime execution. For example, a test can
reference a variable template whose `decltype` expression references an
immediate function. The graph preserves both steps even though no runtime
profile exists for the immediate function. Types and other unprojected
declarations remain explicit boundaries. Dependent expressions that libclang
does not expose, and templates without observed instantiations, can still have
unresolved dependencies.

For dynamic impact Core reuses coverage collection on the after snapshot.
It checks checkout HEAD and source status before and after collection.
Generated evidence on an unchanged snapshot gains
`sourceMatch: "verified-snapshot"` and `sourceSha: afterSha`.
Source mutation, failed builds or tests, and missing profiles stop analysis
with an error. Core does not emit an impact report for failed dynamic analysis.
Existing coverage artifacts can be imported through standalone coverage.

The engine normalizes LLVM segments into `files[].regions[]` with
`start`, exclusive `end` line/byte-column, and execution `count`.
Gap and count-less segments are excluded. Core intersects these source
intervals with affected symbol locations. Matching is `EXECUTED` when all
observed intervals have hits, `NOT_EXECUTED` when none have hits, and
`PARTIALLY_EXECUTED` when both occur. Missing or unverified evidence stays
`UNKNOWN`. Nodes with `executionKind: "compile-time"`, `"declaration"` or `"test-harness"`
receive `NOT_APPLICABLE`, excluded from the assessed count and execution
percentage. Compilation and references do not prove constant evaluation.
These statuses describe source observations and do not prove
branch adequacy, causal test attribution or complete execution paths.

`--skip-tests` is consistent with static impact and conflicts with test
selection. `impact,coverage` adds a top-level
`coverageReport` for the complete collection result. Impact alone includes
the collection under `evidence.coverage[]` and its affected-symbol matching.

## Go, Rust and TS/JS projections

Each engine projects the same symbols/call sites used by its scan collector.
Core owns fingerprint comparison, graph traversal, snapshot lifecycle and
source-region coverage matching. `coverage` remains independently callable;
dynamic `impact` reuses it on the after snapshot.

- Go uses AST token identities and checkout-local bindings. `--resolve` validates
  Go build selection via go-list; interface dispatch and external native symbols
  remain boundaries. Go test-source nodes use `executionKind: "test-harness"`
  because coverprofiles instrument production source, not `_test.go`.
- Rust uses Tree-sitter identities and existing source call candidates. Dynamic
  scan validates both snapshots with frozen Cargo check. `basis: "cargo-check"`
  and `bindingBasis: "source-candidates"` expressly do **not** claim rustc
  type-directed binding. Runtime coverage can still match observed source regions.
- TS/JS uses shared source symbols; dynamic edges require TypeScript semantic
  targets and successful diagnostics. React TSX and Svelte source mappings share
  the scan resolver. Imported test registration evidence can classify callbacks;
  names and test-runner globals alone do not prove a test role.

Go/TS dynamic edges use `SCAN_RESOLVED`; this denotes each engine's documented
binding strength, not necessarily compiler-complete runtime dispatch. Rust keeps
`NAME_CANDIDATE` edges even after successful Cargo validation. Resolution metadata
and limitations stay available under `extensions`.

For TS/JS immutable snapshots, unchanged manifests and lockfiles permit reuse
of the original installed `node_modules` via an ignored snapshot link. Otherwise
npm lockfile projects attempt offline `npm ci --ignore-scripts` in the snapshot.
Missing cached dependencies fail explicitly. Preparation notes record the choice;
package/lock equality does not attest to the integrity of ambient dependencies.

## Multi-engine aggregation

Core selects all matching modern impact engines, with `--stack` as an optional
restriction. Each engine's graph remains independent. Aggregated node and
call-site IDs have an engine prefix, nodes and changes gain `engine`, and
`matching.datasets[].engine` disambiguates coverage. Complete child reports
remain at `extensions.engineReports[engineId]`; single-engine output is unchanged.
No cross-language calls are invented from co-location or matching symbol names.
Coverage datasets stay separate, with no combined percentage. Modern and legacy
Java analysis are not silently mixed; Java remains explicitly selectable.
