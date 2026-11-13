# React architecture facts

The TS/JS engine publishes React facts under
`engineFacts[].data.react` for the collected `architecture` outcome.
The surrounding `data.components` array describes packages.

- `react.components[]` contains named JSX component declarations, with
  `componentId`, `name`, `kind`, `packageId`, `file`, `line`, and `column`.
  `packageId` references the enclosing package component.
- `react.links[]` contains JSX use sites in those components, with
  `fromComponentId`, `tag`, `resolution`, and source location.
  A `resolved` site has `toComponentId` referencing another React component.
  A JSX import backed by a declared npm dependency can be `external`, with
  `importSpecifier` and `importedName` as provenance. A JSX name bound inside
  the render function can be `dynamic` when it cannot be resolved to a
  declaration. These sites retain their location without a target component ID.
  Other unknown sites remain `unresolved`. Multiple sites can reference the
  same component pair.
- Locations are repository-relative and use one-based line and byte-column.
  IDs identify declarations within one scan snapshot.

This is static composition evidence for top-level named functions and
constant function values returning JSX, including JSX in supported map callbacks
and in a same-block `const` initializer returned by the component.
Resolution covers same-file components, supported named and default imports,
local TypeScript path mappings, explicit re-exports and single wildcard
forwarding. Wrappers, class components and general render data flow are outside
the current coverage.
