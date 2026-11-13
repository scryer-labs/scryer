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

## Orrery behavior

Orrery applies the supplied markers by default and preserves the original
facts. A user can add, change or clear a mark on an entity. Store the override
separately against its identity, retain it across rescans when that identity
still matches, and distinguish automatic marks from manual ones. Clearing an
automatic mark is an explicit override. Resetting to automatic removes the
override. Unknown or unmarked entities remain visible in the production view.
