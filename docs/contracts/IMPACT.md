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
HTML renderers. Changed-file rows summarize declaration changes and contain
no inferred Git hunks. Node roles remain `UNKNOWN` for C++ in this slice.
`execution.status` is `SKIPPED` and coverage datasets are empty. Coverage
remains unknown until the reusable coverage collector is bound to the
selected after snapshot. C++ runtime analysis and combined `impact,coverage`
are subsequent work.
