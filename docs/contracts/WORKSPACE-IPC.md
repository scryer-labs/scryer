# Project and workspace IPC contract

Scryer records Windows named-pipe and shared-memory evidence. Both project and
workspace reports publish communication relationships and their source locations.
The matching contract is shared by all engines. The engine ID records where
an observation came from.

## Scan and inspect

```sh
scryer project scan --facts architecture,ipc --json > project.json
scryer workspace scan --facts dependencies,ipc --json > workspace.json
```

The workspace root must contain `.scryer/workspace.yaml` with its member paths.
Use a current Scryer build and an installed C++ Engine that advertises `ipc`.
The C++ engine reads source and CMake declarations without building the Windows
target. Use Orrery's `scryer@v1.9/scan-workspace` for workspace IPC. For a project
scan plan, pass `architecture,ipc` to `scryer@v1.9/scan`. `ipc` alone also
collects executable ownership evidence. `architecture` supplies the component
table for joining and display.

## Read the report

- In project JSON, `communication.ipc` contains matches between distinct
  executable components. Link `from` and `to` refer to component IDs.
  Endpoints with no recovered executable remain visible without an inferred
  component link. Known test executables can participate in project matches.
- In workspace JSON, `communication.ipc` contains matches between distinct
  member projects. Link `from` and `to` refer to member paths. The workspace
  imports normalized endpoints and can enrich named-pipe calls with evidence
  from referenced projects. Imported endpoints have a `projectEndpointId`
  that refers to the ID in that member's `facts.communication.ipc`. Project
  links remain under the member facts.
- `projects[].path` identifies a workspace member. A member contains `facts`
  or `error`. A project's raw IPC facts are in its collected
  `engineFacts[]` outcome for `fact: "ipc"`. The outcome's
  `data.windows.namedPipe` contains endpoint observations and parameter
  evidence. `data.calls` can contain qualified calls with known arguments.
- `dependencies.references[]` holds library references. In the example
  workspace, Brush and Inkstone both reference Felt.
- `communication.ipc.windows.namedPipe` holds normalized `endpoints[]`
  and `links[]`. Each endpoint has an `id`, `role`, original pipe
  `name`, `engine` and `sourceRole`. It can also carry a `componentId`
  and, at workspace scope, a member `project`. Direct observations carry
  `observation`. Derived observations carry `call` and `parameterEvidence`.
  A wrapper resolved across repositories also has `libraryProject`. These objects retain source
  positions relative to the caller or library project that owns them.
  Endpoints and parameter evidence may include `accessMode` (`read`,
  `write`, `read-write` or `unknown`) and the source expression in
  `accessFlags`. Access is relative to the endpoint. The C++ engine
  derives it from the access argument to `CreateNamedPipe` or `CreateFile`
  when it can read the flags directly.
- A link uses `clientEndpointId` and `serverEndpointIds[]` to reference
  endpoint records. `from` identifies the client and `to` identifies
  the server at the scope of the report. A single server endpoint can appear in several
  links. `direction` describes data flow allowed by the observed access
  modes. It does not assert that a message was sent.

  | `direction` | Meaning |
  | --- | --- |
  | `client-to-server` | Client writes and server reads |
  | `server-to-client` | Server writes and client reads |
  | `duplex` | Both endpoints can read and write |
  | `incompatible` | The server does not allow the access requested by the client |
  | `unknown` | No unique server, missing access evidence or differing conclusions across server observations |

  Consumers should treat a missing `direction` in older artifacts as
  `unknown`. Pipe-name matching and its `status` are independent of
  access compatibility.
- `matched` identifies a single peer at the scope of the report and may
  retain several endpoint observations. `ambiguous` lists candidate peers.
  `unmatched` has no peer. Endpoints remain visible even without a match.
  Workspace matching excludes observations marked `sourceRole: "test"`.
- Endpoint IDs are local to one report. Use them for joins in that report.
  The match uses compatible roles and the case-normalized local Windows
  pipe name. Workspace edges represent static evidence of intended
  communication. The current matcher covers local pipe names.

## Windows shared memory

The C++ engine emits this fact at
`engineFacts[].data.windows.sharedMemory` for `fact: "ipc"`. Request
`--facts architecture,ipc` to include CMake component IDs for declared
entrypoint sources. `mappings[].operations[]` records named create/open
operations and `views[]` records mapping calls. Entries preserve API and
source locations, access flags, and available `entrypoints[].callChain`.
Use `entrypoints[].componentIds` to join to project architecture.
`handleSourceCandidateId`, when present, points to an observation within
the same project report. Access modes describe permissions; consumers
should avoid inferring message direction or actual writes from them.

Both scopes publish normalized `communication.ipc.windows.sharedMemory`
endpoints and links when shared-memory evidence is present. Links match
create/open observations by the exact mapping name. `openerEndpointId`
and `creatorEndpointIds[]` reference the participating endpoints.
`from` identifies the opener and `to` identifies the creator. Shared-memory
links describe a named resource association without a data-flow direction.
The `shared_memory_exp` project demonstrates this between its `app` and
`tester` executables.

## First Orrery view

Display a single duplex IPC edge **Brush ↔ Inkstone** for
`scriptorium_input_protocol_pipe`. Brush is the client and Inkstone is
the server. Render `client-to-server` with an arrowhead at the server,
`server-to-client` at the client, and `duplex` at both ends. For
`unknown` or `incompatible`, show the relationship and its status
without a data-flow arrowhead. Keep the two Felt library edges under
their own relation type. An IPC edge detail should show the client and
server call locations and the corresponding Felt parameter evidence. Keep
Inkstone's UI pipe available as a server-only endpoint. Keep Felt's test
endpoint distinguishable from production evidence.

Suggested checks for the view are a shared server referenced by multiple
clients, an ambiguous server match and a server-only endpoint. Run Orrery's
existing Daily Dev Pipeline after changes to the UI or backend.

## Verified local artifacts

Scryer + C++ Engine Integration run `20261112T134947228Z-56fbf5a1` passed in
`scryer-lab`. Its `workspace-scan` step scanned the existing Scriptorium
workspace and verified the Brush–Inkstone duplex link in the [complete workspace JSON](/Users/edwardnoaland/Desktop/experiment/scryer-lab/.orrery/plans/runs/20261112T134947228Z-56fbf5a1/outputs/20261112T134947228Z-56fbf5a1/workspace-scan/dccc1262-2a10-4411-afca-aca34c4f7c95/workspace.json).

The `shared_memory_exp` project scan produced a [project facts artifact](/Users/edwardnoaland/Desktop/experiment/shared_memory_exp/.orrery/plans/runs/20261112T135125327Z-737425f9/outputs/20261112T135125327Z-da4dfc06/scan/7cbcc97d-96a7-4945-875d-0ef77f60a32c/facts.json)
with the `tester` and `app` shared-memory link under
`communication.ipc.windows.sharedMemory`.

Felt's project-level IPC Verify run `20261112T111656560Z-0818ae63` passed in
`scriptorium-workspace`. Its [IPC evidence JSON](/Users/edwardnoaland/Desktop/experiment/input_method/scriptorium-workspace/.orrery/plans/runs/20261112T111656560Z-0818ae63/outputs/20261112T111656560Z-0818ae63/scan/b2e17994-1fa0-494c-a0eb-4f1104a21930/ipc.json)
shows three factory and three constructor parameter observations and a
concrete test endpoint.

See the [CLI reference](../CLI.md), [Workspace report type](../../internal/application/workspace_scan.go)
and [IPC types and matcher](../../internal/application/workspace_ipc.go).
