# Workspace IPC contract and Orrery handoff

Scryer records IPC evidence. Orrery consumes the Workspace JSON to display
communication relationships and their source locations. Windows named pipes
are the first implemented mechanism. The matching contract is shared by all
engines. The engine ID records where an observation came from.

## Scan and inspect

From a configured workspace root, run:

```sh
scryer workspace scan --facts dependencies,ipc --json > workspace.json
```

The workspace root must contain `.scryer/workspace.yaml` with its member paths.
Use a current Scryer build and an installed C++ Engine that advertises `ipc`.
The named-pipe scan reads source and works without building the Windows target.
The official Orrery `scryer@v1.7/scan-workspace` operation currently requests
`dependencies` only. For the first UI pass, use the verified artifact below.
A live IPC scan needs a workspace invocation that requests `ipc`.

## Read the report

- `projects[].path` identifies a workspace member. A member contains `facts`
  or `error`. A project's raw IPC facts are in its collected
  `engineFacts[]` outcome for `fact: "ipc"`. The outcome's
  `data.windows.namedPipe` contains endpoint observations and parameter
  evidence. `data.calls` can contain qualified calls with known arguments.
- `dependencies.references[]` holds library references. In the example
  workspace, Brush and Inkstone both reference Felt.
- `communication.ipc.windows.namedPipe` holds normalized `endpoints[]`
  and `links[]`. Each endpoint has an `id`, member `project`, `role`,
  original pipe `name`, `engine` and `sourceRole`. Direct observations
  carry `observation`. Derived observations carry `call`,
  `parameterEvidence` and `libraryProject`. These objects retain source
  positions relative to the caller or library project that owns them.
  Endpoints and parameter evidence may include `accessMode` (`read`,
  `write`, `read-write` or `unknown`) and the source expression in
  `accessFlags`. Access is relative to the endpoint. The C++ engine
  derives it from the access argument to `CreateNamedPipe` or `CreateFile`
  when it can read the flags directly.
- A link uses `clientEndpointId` and `serverEndpointIds[]` to reference
  endpoint records. `from` is the client member path and `to` is the
  server member path. A single server endpoint can appear in several
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
- `matched` identifies a server project and may retain several endpoint
  observations in that project. `ambiguous` lists candidate projects and
  endpoint observations. `unmatched` has no server project. A server with
  no incoming match remains in `endpoints`. Test observations have
  `sourceRole: "test"` and do not participate in production matches.
- Endpoint IDs are local to one report. Use them for joins in that report.
  The match uses compatible roles and the case-normalized local Windows
  pipe name. Workspace edges represent static evidence of intended
  communication. The current matcher covers local pipe names.

## Project-level Windows shared memory

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

The `shared_memory_exp` project provides an example with an app and a
tester mapping the same object. Workspace IPC aggregation currently applies to
named pipes. Shared-memory observations remain project facts.

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

Scryer + C++ Engine Integration run `20261112T123241847Z-e0727f0e` passed in
`scryer-lab`. Its `workspace-scan` step scanned the existing Scriptorium
workspace and verified the Brush–Inkstone duplex link in the [complete workspace JSON](/Users/edwardnoaland/Desktop/experiment/scryer-lab/.orrery/plans/runs/20261112T123241847Z-e0727f0e/outputs/20261112T123241847Z-e0727f0e/workspace-scan/8081b7b5-cd43-4a00-bbc3-340401315cff/workspace.json).

Felt's project-level IPC Verify run `20261112T111656560Z-0818ae63` passed in
`scriptorium-workspace`. Its [IPC evidence JSON](/Users/edwardnoaland/Desktop/experiment/input_method/scriptorium-workspace/.orrery/plans/runs/20261112T111656560Z-0818ae63/outputs/20261112T111656560Z-0818ae63/scan/b2e17994-1fa0-494c-a0eb-4f1104a21930/ipc.json)
shows three factory and three constructor parameter observations and a
concrete test endpoint.

See the [CLI reference](../CLI.md), [Workspace report type](../../internal/application/workspace_scan.go)
and [IPC types and matcher](../../internal/application/workspace_ipc.go).
