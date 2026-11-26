# TS/JS call graph

Request `call-graph` from the TS/JS engine. It uses the same public fields as
the [Rust call graph](rust-call-graph.md), including `symbols[]`, `calls[]`,
`callerSymbolId`, `targetSymbolId` and `resolution`. Svelte script ranges
refer back to the original file.

The resolver matches stable lexical bindings and direct named/default imports.
It supports local JSONC `tsconfig.json` path mappings, relative `extends`,
explicit re-exports and single wildcard forwarding. Dynamic calls, namespace
calls, mutable bindings and external symbols remain unresolved. Use
`resolutionReason` for observed but unresolved calls and `limitations[]`
for known gaps in scan coverage.

An optional `testEvidence` on a call records the source import's `module`,
`export`, and `kind`. It identifies an explicit registration call from
Vitest, Jest, Playwright or Node's test API, at the call site only.

JSX render relationships remain in `architecture.react` and have additive
source-reference projections described below.
`architecture.svelte` exposes component files and template tags with source
locations. A unique default import pointing to a Svelte file produces an
`import-candidate`; template scope and dynamic selection remain unverified.

For SvelteKit, `files.imports[]` can report `resolution: "convention-candidate"`
and a `targetFile` when the default `$lib` layout points to one file and no
explicit alias resolves. This remains a candidate because app configuration
can override the default.


## Source references and non-executable component entities

The graph adds `references[]`, separate from invocation `calls[]`. Each reference
contains:

- `referenceId`: opaque identity for this observed relationship within a scan;
- `sourceSymbolId`: originating function/callback or component source entity;
- optional `targetSymbolId`: uniquely identified source candidate, when available;
- `name`: observed tag or handler name;
- `basis`: `react-render`, `react-handler`, `svelte-component` or `svelte-handler`;
- `resolution`: the originating framework collector's resolution evidence;
- `site`: original-source `file`, one-based line/byte-column `start` and exclusive
  `end`, using the same range format as calls.

A reference target is a source dependency candidate, not a compiler-confirmed
invocation. The field name does not strengthen the originating `resolution` or
`basis`. Missing targets remain explicit observations; imports, mutable/shadowed
bindings, template scope and dynamic components can prevent a safe source join.
Requests with semantic resolution still retain these source-only framework
references without promoting them to TypeScript semantic call targets.

Svelte files additionally contribute a `symbols[]` entity with `kind: "component"`,
`executionKind: "non-executable"`, display `name`, `symbolId` and original-file
`definition`. It represents the component source, including template context;
it does not claim a generated JavaScript function. Framework component/handler
references can originate from this entity and point to component or callable
symbols. IDs are opaque and only join within the current graph.

Ordinary anonymous callbacks are retained as callable symbols independently of
test recognition. Impact correspondence uses enclosing lexical and call/attribute
anchors rather than offsets; repeated indistinguishable anchors remain ambiguous.
Being passed to a hook, promise, iterator or JSX attribute does not by itself
prove invocation. Only imported supported registration evidence supplies a test
classification; a string argument or callback name alone does not.

Impact projects references as `kind: "framework-reference"` with
`candidateSymbolId` and `candidateBasis`, distinct from semantic call targets.
Core preserves `FRAMEWORK_REFERENCE` and `basis` in its potential-impact graph.
Svelte component source entities remain non-executable and receive
`NOT_APPLICABLE` runtime function coverage. See [impact projection](IMPACT.md).
