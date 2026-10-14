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
