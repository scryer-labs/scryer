# Scryer–Orrery test classification contract

This contract defines the boundary between Scryer producers and Orrery
consumers.

## Scryer facts

- Classification is deterministic for the same source, scan configuration and
  engine version. An engine emits a test-only marker only when language or
  build-system evidence establishes that scope. Names containing `test` and
  directory names alone carry no classification. A framework dependency
  alone identifies a dependency, without classifying the component using it.
- Component kind and purpose are independent. A library may serve as
  `test-support`, while executables may have purpose `test` or `benchmark`.
  Identify Google Benchmark separately from GoogleTest and Catch2. A future
  component-purpose fact must carry evidence for that purpose.
- Mark the narrowest proven scope. A production source file can contain
  test-only modules, functions and dependency references. A known test edge
  cannot classify a whole file or component. Missing evidence means unknown,
  including when an engine does not implement this contract.
- Keep static test discovery separate from execution, coverage and test
  effectiveness. A future `tests` inventory can refer to existing component
  and symbol IDs without duplicating them.

## Current Rust wire format

Rust modules, imports and explicit crate-qualified source references can
carry the following additive object:

```json
{
  "testContext": {
    "scope": "test-only",
    "evidence": [
      {
        "kind": "rust-cfg",
        "expression": "#[cfg(all(test, unix))]",
        "file": "src/infrastructure/execution/agent_runtime.rs",
        "line": 291,
        "column": 1
      }
    ]
  }
}
```

Locations use repository-relative files and one-based line and byte-column.
Rust recognizes built-in `#[test]` on a function and supported `cfg` conditions
that require the `test` predicate. For example, `all(test, unix)` requires
`test`; `any(test, feature = "debug")` may be enabled outside a test build.
Inline scopes inherit their enclosing evidence. A resolved external module
file inherits a guard when all known Cargo target routes into that file require
test evidence. A production route suppresses the inherited mark.

`files.imports[].paths` lists expanded syntactic import paths, including grouped
imports. `files.references[]` contains explicit `crate::` paths outside imports,
with locations and the same optional `testContext`. Paths are source references
without semantic symbol resolution or proof of a function call. Missing
`testContext` leaves the fact visible in the default view.

## TS/JS wire format

An explicit test registration call can carry `testEvidence` with `module`,
`export`, and `kind`. Supported kinds include `test-registration`,
`test-suite`, and `benchmark-registration`. This evidence applies to the
call. A matching filename or package dependency does not mark the file.

## Go wire format

Go marks language-defined test files and their source observations with
`testOnly`. A package can contain both implementation and test files.
See the [Go facts contract](https://github.com/scryer-labs/scryer-go-engine/blob/main/docs/FACTS.md).

## C++ boundary

Resolved C++ call-graph symbols and impact nodes can carry `role: "TEST"`
and `testContext: {"scope": "test-only", "evidence": [...]}`. Each registration
evidence records `kind: "compiler-test-registration"`, `framework`, `symbolUsr`,
`definitionFile`, and repository-relative `file`, `line`, `column`.

The engine matches compiler-bound registration APIs against its isolated
`internal/testclassification/registrations.json` rules. The initial rule covers
GoogleTest's `MakeAndRegisterTestInfo` and its framework declaration. It does
not depend on a source macro being spelled `TEST`: a wrapper macro can qualify
when compiler expansion provides the same evidence at its definition's start.
An ordinary function that calls a registration API is not itself a test body.
Helpers, callees, linked libraries, filenames and framework dependencies do not
inherit a test-only role. Missing evidence stays unknown. Static scanning alone
currently does not emit this marker. Core preserves engine roles and evidence
without containing framework recognition rules.

Future independent evidence providers can read build-system test registrations
or explicit user configuration. Their scope must stay precise: CTest registering
an executable establishes that executable's purpose, not a test-only role for
all production functions linked into it. Running an arbitrary executable as a
selected coverage target does not establish test semantics. Rules/configuration
for other registration APIs and build providers are not implemented by this change.

The C++ IPC collector's older path-based `sourceRole` remains an IPC-specific
hint and is not a substitute for these strict markers.

## Orrery behavior

Orrery applies the supplied markers by default and preserves the original
facts. A user can add, change or clear a mark on an entity. Store the override
separately against its identity, retain it across rescans when that identity
still matches, and distinguish automatic marks from manual ones. Clearing an
automatic mark is an explicit override. Resetting to automatic removes the
override. Unknown or unmarked entities remain visible in the production view.
