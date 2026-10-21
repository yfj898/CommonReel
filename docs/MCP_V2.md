# v2 protocol and authority

Official Go MCP SDK v1.8.0; Streamable HTTP; supported protocol 2025-11-25.
The running application imports and calls both MCP server and client at runtime.
No Alexa device connection, voice capture or certification is claimed.

`/mcp` requires a room capability in Authorization and a session ID in
X-CommonReel-Session. Both are bound by the server/client, not tool arguments.
The server checks current lifecycle before every request and business call.

| Tool | Authority |
|---|---|
| catalog_info | Bounded coverage, genres, data notice |
| get_room | Task view, revisions, explicit limits, counts; no full history |
| set_member | Existing explicit member's tonight patch; CAS and operation key |
| set_policy | Explicit shared policy; CAS and operation key |
| shortlist | Same Go eligibility/ranking as the page; max 6 |
| search_movies | Supported catalog search, not proof of eligibility |
| explain_candidate | Checked single-film facts and reproducible estimates |
| runtime_preview | Cache-only full eligibility comparison for a larger runtime; no room writes or supplier calls |

Unknown fields, other-room fields and unlisted tools are rejected. Agent writes
also require an explicit member target; hard-input relaxations need evidence
in the request. Responses, save, remove-member, close-room and profile
operations are intentionally absent. The page uses `/api/v1` for these actions.

The local model receives a bounded task view and tool schemas. At most one
turn runs per room, with 6 tools and a 90-second deadline. Model calls do not
hold SQLite write transactions. Page updates can produce a revision conflict;
the agent does not blindly replay rejected writes. Closing/expiring a room or
deleting a linked profile cancels the turn and clears bounded conversation.
Raw conversation is not written to SQLite.

Factual replies use Go-rendered tool results. A model that omits a requested
explanation may be followed by an actual SDK `explain_candidate` call tagged
`origin=grounding_guard`; model-requested calls are tagged `origin=model`.
This distinction remains in real-run evidence. The guard consumes the same
six-call budget. It does not invent a tool result or pretend the model made
the omitted call. Unsupported model provider names and runtime prose are
discarded rather than displayed as facts.

Final candidate facts are version checked before display. Missing model,
timeout, protocol failure, unknown tool or ambiguous target leaves forms
available. A successful operation's minimal receipt is durable; agent turn
retries return an already-processed receipt and current-state instruction,
not a stored raw chat transcript.

An agent-turn receipt write failure returns an HTTP error. Individual tool
writes may already be committed; read the current room before retrying.
Tool rejection traces retain the business error code and message, such as
`REVISION_CONFLICT` and its instruction to refresh the room.

## Runtime preview HTTP and agent contract

`POST /api/v1/sessions/{id}/runtime-preview` uses the usual room capability.
Body: `{"max_runtime_minutes": 135}`. The value must exceed the current saved
limit and be at most 600. The response contains:

- `room_revision`, `decision_version`, `current_max_runtime_minutes`, `max_runtime_minutes`;
- `current_feasible_count`, `proposed_feasible_count`, `newly_feasible_count`;
- `current_unknown_count`, `proposed_unknown_count`, `catalog_total`;
- up to six newly eligible `candidates`, with checked offers/times and preference evidence;
- `preview_basis`, binding this room/revision, target runtime, catalog content and cached supply facts/freshness.

Both counts use the same complete eligibility function as ordinary candidates,
before display truncation. Unknown counters follow metadata/supply exclusions
in each full evaluation. Other constraints do not change. The endpoint and MCP
tool do not call `current`, fetch a supplier, clean up a candidate, or persist a
room/receipt. Missing or expired facts remain unknown.

The page applies through the existing commands endpoint:

```json
{
  "type": "apply_runtime_preview",
  "operation_key": "a-new-unique-operation-key",
  "expected_room_revision": 8,
  "max_runtime_minutes": 135,
  "preview_basis": "the-basis-returned-by-the-preview"
}
```

The existing transaction rechecks the room version and cache-bound basis.
`REVISION_CONFLICT` / `PREVIEW_STALE` (409) requires a fresh preview.
`NO_NEW_CANDIDATES` (409) cannot apply; the page omits the apply button for this
case. Invalid target runtime returns `INVALID_ARGUMENT` (400). Successful apply
changes only runtime, advances the decision, clears the old candidate/responses,
and uses existing idempotency receipts. No new database table was added.

Agent requests can set `mode: "runtime_preview"`. The host advertises and
permits only `runtime_preview`, `catalog_info`, `search_movies` in that mode.
The initial task view supplies room context; no lifecycle-mutating or supply-
fetching tool is exposed. A model asking for a writer is rejected by the host.
The returned `runtime_preview` object, not model prose, renders the card.
Ordinary explicit edits retain their original tools. A hypothetical request
cannot authorize a writer in ordinary mode, and an ordinary-mode read preview
also receives the same structured card. Applying remains a page button.
