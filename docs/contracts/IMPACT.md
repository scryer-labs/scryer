# Static impact projection

Core prepares immutable before and after checkouts and passes their paths and
commit identities to an engine declaring the `impact` operation. The JSON
request uses `impact` with `before`, `after`, `beforeRoot`, and `afterRoot`.
The C ABI and JSON protocol version remain 1.

The engine returns `schemaVersion`, `beforeSha`, `afterSha`, `before` and
`after`. Each snapshot contains:

- `symbols[]` with opaque `id`, correspondence `key`, display `signature`,
  content `fingerprint`, source text, and `location`.
- `calls[]` with `callSiteId`, optional `callerSymbolId`, `callee`,
  optional `candidateSymbolId`, `resolution`, and `site`.
- `contexts[]` with source `path` and a fingerprint of tokens outside
  function definitions.
- `notes[]` with collection limits.

Locations have `file` and one-based `start` and exclusive `end` line/byte-column
coordinates. Files are relative to their respective snapshot roots. IDs only
join within a snapshot. Engine correspondence keys pair declarations across
snapshots and remain internal to that engine's versioned observations.

## C++ source evidence

C++ uses Tree-sitter function definitions. Named enclosing classes/namespaces
and declarator tokens contribute to a file-local correspondence key. Whole
function token fingerprints identify body or declaration changes. Comments
and whitespace do not change token fingerprints. A file rename or declarator
change is reported as removal and addition. Duplicate keys stay ambiguous.
Lambdas, anonymous scopes and malformed definitions have no correspondence key.

Call candidates are unique textual name matches from the existing static
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
remain `UNKNOWN` for C++.

## After coverage

Core reuses the coverage use case against the after snapshot by default.
It checks checkout HEAD and source status before and after collection.
Generated evidence on an unchanged snapshot gains
`sourceMatch: "verified-snapshot"` and `sourceSha: afterSha`.
Imported artifacts stay unverified. Source mutation invalidates matching and
sets `execution.status: "SNAPSHOT_CHANGED"`. Build, test, and no-profile
failures retain static impact and their execution diagnostics.

The engine normalizes LLVM segments into `files[].regions[]` with
`start`, exclusive `end` line/byte-column, and execution `count`.
Gap and count-less segments are excluded. Core intersects these source
intervals with affected symbol locations. Matching is `EXECUTED` when all
observed intervals have hits, `NOT_EXECUTED` when none have hits, and
`PARTIALLY_EXECUTED` when both occur. Missing or unverified evidence stays
`UNKNOWN`. These statuses describe source observations and do not prove
branch adequacy, causal test attribution or complete execution paths.

`--skip-tests` retains static impact only. `impact,coverage` adds a top-level
`coverageReport` for the complete collection result. Impact alone includes
the collection under `evidence.coverage[]` and its affected-symbol matching.
