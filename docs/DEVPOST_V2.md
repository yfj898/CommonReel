# CommonReel — A Group-Recorded Choice for Movie Night

**Track:** Alexa+ (Self-Hosted MCP Server `2025-11-25` + Companion Web Simulation)
**Mini-challenges:** No eligibility claim is made from this primary-track repository alone. Open Source requires a qualifying additional contribution and its URL; AWS Builder is not claimed.
**Repository:** [https://github.com/yfj898/CommonReel](https://github.com/yfj898/CommonReel)

---

## Inspiration

Picking a movie with two to four friends in the same room is rarely solved by a single personal recommendation score. Every person brings hard boundaries: how much time the group has tonight, which US streaming subscriptions are available on the screen, genres someone wants to skip (*genre vetoes*), and films people have already watched.

Two engineering concerns shaped the implementation. The concurrency case was exercised with controlled tests; the model failure was observed in a real local-model run:

1. **Concurrent state collisions (controlled test scenario):** If one person edits a runtime limit or genre veto while others are recording responses on a candidate film, naive state handling either overwrites edits or lets a group save an outdated decision.
2. **LLM streaming-provider hallucinations:** In our first live local-model run (`docs/evidence/agent-real-v2-before-grounding.json`), `qwen3.5:4b` read numeric provider IDs from tool output, invented streaming platform names (*Sundance Now, TNT Drama, TruTV*), and skipped calling the candidate explanation tool.

**CommonReel** addresses both problems: it brings a small group's movie-night limits into one explicit, versioned record, and pairs a real **MCP `2025-11-25` server/client workflow** with **deterministic Go fact grounding** so an AI assistant can plan tool calls while Go controls displayed film facts and enforces group decision rules.

---

## What It Does

1. **Zero-Account LAN Room Setup (2–4 Adults):** One person creates a room and shares a capability link or locally generated QR code over same-Wi-Fi LAN. Friends join with a nickname—no accounts or cloud sign-ins required.
2. **Explicit Shared Limits & Individual Vetoes:** The room sets tonight's max runtime, active US streaming subscriptions, and rewatch policy. Each member sets liked genres, hard genre vetoes, and known watched films (manually, via optional **Netflix Viewing Activity CSV** import, or via a **reusable profile recovery code**).
3. **Strict Eligibility Before Ranking:** Go filters a **217-film Wikidata CC0 catalog** against verified **Watchmode US subscription facts**. Unknown runtime, conflicting metadata, unconfirmed/expired supply, and rent/buy-only listings are strictly excluded. Eligible candidates are ranked using an explainable group estimate ($0.65 \times \min + 0.35 \times \text{mean}$), where *"watched does not mean liked."*
4. **Alexa+ Web Simulation via Real MCP (`2025-11-25`):** Users can ask the assistant to update explicit limits, find qualifying films, or explain a candidate. A local Ollama model calls our self-hosted MCP server over **Streamable HTTP (`2025-11-25`)**, while all forms remain usable if inference is slow or offline.
5. **Versioned Group-Recorded Choice:** Proposing a candidate opens an acceptance record for every member entry. Anyone holding the room invite may record for another member, so the outcome is explicitly modeled as a **group-recorded choice** (not proof of personal consent). Saving refreshes live streaming supply outside the SQLite write transaction and commits only if all member entries accept the current `decision_version`.

---

## How We Built It

- **Unified Go Workflow Engine & Dual-Version Concurrency (`internal/workflow`):** Both the web UI (`/api/v1`) and the MCP server (`/mcp`) execute against a single Go domain module backed by **SQLite WAL** (`modernc.org/sqlite`). We separated `room_revision` (incremented on every write for CAS and SSE sync) from `decision_version` (incremented only when members, preferences, policies, or candidate eligibility facts change). Recording one member's acceptance increments `room_revision` without clearing other members' responses, whereas changing a veto atomically increments both versions and invalidates stale responses.
- **Official Go MCP SDK (`v1.8.0`) over Streamable HTTP (`2025-11-25`):** The server exposes 7 whitelisted tools (`catalog_info`, `get_room`, `set_member`, `set_policy`, `shortlist`, `search_movies`, `explain_candidate`). Final-authority actions (`record_acceptance`, `save_choice`, `remove_member`, `close_session`, profile maintenance) are excluded from MCP and restricted to page controls.
- **Capability Isolation & Transport Binding:** Room invite tokens and 256-bit profile recovery codes are generated with browser `crypto.getRandomValues`, sent in HTTP headers, and stored as SHA-256 hashes. The Go MCP client binds `Authorization: Bearer <room-secret>` and `X-CommonReel-Session` at the transport layer (`capabilityTransport`), keeping capability secrets out of model arguments and binding calls to one room; cross-room injection is covered by controlled tests.
- **Two-Layer Hallucination & Authority Guard (`agent_policy.go` & `grounding.go`):**
  - **Write Guard (`guardAgentWrite`):** Before executing `set_policy` or `set_member`, Go checks that the user's request contains explicit lexical evidence for any relaxed hard limit.
  - **Read Grounding (`grounding_guard`):** Go renders all user-facing film facts, provider labels, and score breakdowns directly from verified MCP tool results. If the model omits a requested explanation call, Go executes a logged MCP `explain_candidate` call tagged `origin=grounding_guard` (kept distinct from `origin=model`).
- **Dependency-Free Dark Cinema Frontend (`internal/workflow/web`):** Redesigned on 2026-10-15 with a dark visual hierarchy inspired by JustWatch and Letterboxd (`docs/FRONTEND_REDESIGN.md`). Embedded into the Go binary with zero Node/bundler runtime dependencies, responsive down to 360px viewports, and synchronized via `fetch`-streamed SSE.

---

## Challenges We Ran Into & What We Learned

1. **Turning a Real LLM Failure into Deterministic Grounding:** When `qwen3.5:4b` invented provider names from numeric source IDs and skipped `explain_candidate` in our first real run, we retained that failed trace (`docs/evidence/agent-real-v2-before-grounding.json`), moved factual prose rendering to Go, added `origin=grounding_guard`, and verified the grounded rerun (`docs/evidence/agent-real-v2.json`).
2. **Wikidata SPARQL Rate Limits & Genre Mapping:** Wikidata SPARQL returned HTTP 429 during catalog construction, so `scripts/build-catalog-entities.py` switched to the official Wikidata Entity API with exact IMDb cross-checks and verified English genre labels (`assets/genre-map.json`).
3. **Out-of-Transaction Supply Refresh on Save:** To prevent network latency from locking SQLite writes or allowing stale commits, `save_choice` refreshes Watchmode supply outside the write transaction and re-verifies `room_revision`, `decision_version`, and `facts_version` before committing (`TestSaveRefreshDoesNotLockRoomAndRejectsConcurrentPolicyChange`).

---

## Verification Scope & Honest Boundaries

- **Target Audience & Human Outcomes:** The target pain point is a product hypothesis. Live supplier probes, multi-origin browser sync, and 360px viewport checks used constructed 2–3 member groups and Netflix-format CSV fixtures. **No human group study has been run, and no group satisfaction or time-saving claim is made.**
- **Alexa+ Simulation Scope:** This is a web simulation backed by a working self-hosted MCP server (`2025-11-25`); it does not connect to or claim certification on an Alexa device.
- **Physical Phone Status:** LAN listener binding (`-lan`) and 360×800 narrow browser viewports are verified (`PASS`). Testing on a physical phone remained `NOT_RUN` on 2026-10-14/15 because the owner could not test a phone at that time.
- **Legacy Prototype Separation:** Earlier offline MovieLens 1M user-kNN metrics (`docs/LEGACY_PROTOTYPE.md`, port 8092) remain independent research benchmarks and are not claimed as live v2 group results.

---

## Submission Links

- **GitHub Repository:** [yfj898/CommonReel](https://github.com/yfj898/CommonReel)
- **Public YouTube/Vimeo Demo (< 3 min):** PENDING — local English edited demo (`dist/CommonReel-demo.mp4`); inspect `docs/evidence/demo-video-v2.json` for current duration, source captures and hash.
- **Product Feedback:** [docs/PRODUCT_FEEDBACK_V2.md](PRODUCT_FEEDBACK_V2.md)
- **Acceptance & Implementation Evidence:** [docs/ACCEPTANCE.md](ACCEPTANCE.md) and [docs/IMPLEMENTATION_STATUS.md](IMPLEMENTATION_STATUS.md)


Status: English submission draft, not posted. The [official rules](https://amazonappdev2026.devpost.com/rules) were rechecked on 2026-10-16. The owner still needs to review eligibility and final fields and supply a public YouTube/Vimeo URL. The primary-track code and the separate Open Source mini-challenge contribution requirement are distinct.
