# Amazon Developer & Tooling Product Feedback (v2)

Historical observations through 2026-10-15; document checked 2026-10-16 on Windows (local Go 1.26.4, SQLite WAL, MCP Go SDK v1.8.0, Ollama 0.33.3). This report covers only tools, specifications, and APIs actually exercised in building CommonReel. No AWS cloud runtime service, physical Alexa device, Fire TV, Bee, or Ring hardware API was used. Alexa+ feedback concerns the competition documentation and permitted simulation path; it is not feedback from an Alexa runtime integration.

---

## 1. Developer Tools, APIs, and SDKs Used (and How)

- **Amazon Alexa+ Track Specification & MCP Protocol (`2025-11-25`)**: Used to architect CommonReel as a self-hosted Model Context Protocol (MCP) server and companion web simulation for shared living-room movie selection.
- **Official MCP Go SDK (`github.com/modelcontextprotocol/go-sdk v1.8.0`)**: Used at runtime for both the MCP server (`/mcp` over Streamable HTTP `2025-11-25`) and the in-process MCP client invoked during local assistant turns. Seven whitelisted tools (`catalog_info`, `get_room`, `set_member`, `set_policy`, `shortlist`, `search_movies`, `explain_candidate`) share one authoritative Go business module (`internal/workflow`) with the page HTTP handlers.
- **SQLite (`modernc.org/sqlite v1.36.3`)**: Pure-Go SQLite driver in WAL mode for atomic room state transitions, dual-version optimistic concurrency (`room_revision` vs. `decision_version`), 20-way concurrent idempotency receipts, and reusable profile storage.
- **Local Ollama (`0.33.3` + `qwen3.5:4b`)**: Loopback-only local model planning bounded MCP tool calls (max 6 calls / 90s deadline) without paid API keys or cloud data egress.
- **Watchmode Free API & Wikidata CC0 Entity API**: Wikidata provides the reproducible 217-film CC0 catalog with verified IMDb/Watchmode cross-references and runtime units; Watchmode provides bounded, cached US subscription availability checks.
- **Browser Web APIs & Vendored QRCode.js (MIT)**: `crypto.getRandomValues` for 256-bit capability secrets over LAN HTTP, `fetch`-based SSE with `Authorization` headers, and local QR code generation.

---

## 2. What Worked Well (Installation, Documentation, Testing, Performance, Stability)

- **Clear Alexa+ Simulation & Protocol Rules**: The hackathon documentation's explicit minimum protocol requirement (`2025-11-25`) and allowance for a self-hosted MCP server + web simulation made it possible to build and test a complete, deterministic multi-user application without gated hardware access.
- **MCP Go SDK Streamable HTTP & Schema Validation**: `mcp.NewServer` with `SupportedProtocolVersions: []string{"2025-11-25"}` and typed Go struct handlers (`mcp.AddTool`) enforced strict JSON schema validation out of the box. Unknown fields, cross-room injection attempts (`room_id`), and unlisted tools (`save_choice`, `remove_member`) were cleanly rejected during automated adversarial tests (`TestMCPUsesBoundRoomAndOnlyAllowsSevenTools`).
- **Pure-Go SQLite Stability**: `modernc.org/sqlite` compiled cleanly on Windows without requiring a C compiler (CGO), passing 20-goroutine identical-retry storms, competing CAS writes, and process restart recovery tests.

---

## 3. What Could Be Improved (Friction Log, Limitations, and Workarounds)

| Tool / Spec | Purpose & What Worked | Observed Friction / Limitation | Workaround / Engineering Response |
|---|---|---|---|
| **Amazon Alexa+ MCP Track Docs** | Clear protocol version gate (`2025-11-25`) and simulation scope | We did not find a minimal reference Go `Streamable HTTP` server + browser companion template showing capability-header binding in the materials consulted (`initialize` -> `tools/list` -> `tools/call`) | Built a custom `http.RoundTripper` (`capabilityTransport`) that binds `Authorization: Bearer <room-secret>` and `X-CommonReel-Session` at the transport layer so secrets never enter LLM tool arguments |
| **MCP Go SDK v1.8.0** | Real server + client over Streamable HTTP `2025-11-25`; native schema validation | Tool handlers do not natively scope per-request tenant capabilities unless wrapped per request or injected via custom transport headers | Constructed per-request bound MCP server/client instances (`a.mcpServer(id, secret)`) sharing the SQLite-backed workflow module; no SDK defect was established |
| **Ollama 0.33.3 + `qwen3.5:4b`** | Zero paid API; actual local tool planning | Cold model load + first tool request took ~86s on this machine (warm runs: 6.26s–16.86s). Worse, an initial ungrounded run hallucinated streaming provider names from numeric IDs and skipped calling `explain_candidate` | Retained the failed run (`evidence/agent-real-v2-before-grounding.json`); moved factual text rendering to Go (`grounding.go`), added `guardAgentWrite` intent checks, and added an explicit MCP follow-up call tagged `origin=grounding_guard` (`evidence/agent-real-v2.json`) |
| **SQLite / `modernc`** | WAL, durable CAS, and minimal receipts | Storing session copies as JSON simplifies atomic transactions but limits scaling for very large pending watch histories; network I/O cannot sit inside write transactions | Moved Watchmode supply refresh outside the SQLite write transaction (`workflow.go`), then verified `room_revision`, `decision_version`, and `facts_version` inside the commit transaction (`TestSaveRefreshDoesNotLockRoomAndRejectsConcurrentPolicyChange`) |
| **Watchmode Free API** | Exact IMDb match and US source requests succeeded (200); invalid title returned 404 | Consumer website login differs from developer API-key signup; free quota requires strict request budgeting and separation of `sub` vs. `rent`/`buy` | Added explicit developer signup instructions, 24h SQLite supply cache (30-day cap), max 30 probes/round, concurrency 4, process cap (default 600), and immediate pause on HTTP 429 (`TestSupplier429StopsFurtherRefreshAndLeavesNoCandidates`) |
| **Wikidata Query / Entity API** | CC0 entity metadata and stable identifiers for 217 films | Public SPARQL endpoint returned HTTP 429 under rate limiting; initial local assumptions about numeric genre IDs were wrong | Switched `scripts/build-catalog-entities.py` to the Wikidata Entity API with retries and fetched official English genre labels (`assets/genre-map.json`) |
| **Browser APIs & Windows Go Toolchain** | `crypto.getRandomValues`, `fetch` SSE, and 360px responsive layout worked over LAN HTTP | Changing `#room=...&invite=...` hash fragments in-page initially left stale room state rendered; Windows lacked a local CGO toolchain for `go test -race` | Added `hashchange` state reset + stale-response generation guards in `app.js`; configured `go test -race -count=1 ./...` in `.github/workflows/go.yml` on `ubuntu-latest` |

---

## 4. Onboarding Experience (Zero to Hello World)

- **MCP Go SDK (`v1.8.0`)**: Typed Go handlers and transport support reduced setup boilerplate. No onboarding duration was measured. Having `mcp.AddTool` infer JSON schemas directly from Go struct tags eliminated manual JSON Schema boilerplate.
- **Watchmode & Local Ollama**: Ollama local setup (`ollama pull qwen3.5:4b` with `OLLAMA_NO_CLOUD=1`) was straightforward, though documenting the distinction between Watchmode's consumer portal and its developer API key request page was necessary to smooth onboarding for anyone reproducing the repo.

---

## 5. Would We Continue Building with These Tools in the Future?

- **Yes — MCP Go SDK (`2025-11-25`) & Alexa+ MCP Architecture**: Yes. Binding an MCP server directly to the same domain state machine used by the web UI proved to be a clean, testable pattern for multi-modal apps. Even when a small local model (`qwen3.5:4b`) hallucinated prose or skipped an explanation call, keeping strict authority in Go MCP tools and deterministic grounding let us bound the tested assistant operations alongside multi-user forms. This is evidence for the exercised scenarios, not a general safety guarantee.
- **Yes — `modernc.org/sqlite` + Pure Go Single-Binary Server**: Yes, for local-first and household/LAN group tools where zero setup (no Docker, Node build, or cloud database) is a clear practical advantage.


## Additional tooling and evidence scope

QRCode.js (MIT) generates invitations locally, without an external QR service. Its vendored license is retained. GitHub Actions runs Linux race tests, vet and build; `actions/checkout@v7` and `actions/setup-go@v7` are published official versions, so the proposed downgrade was not applied. The Windows machine still lacks a CGO compiler for a local race run.

The real supplier and model observations remain dated evidence, not a performance benchmark. HTTP 429 behavior is covered with an injected supplier response; no real account quota was deliberately exhausted. Guard-requested MCP calls are not counted as model-requested calls. Physical phone and human-study outcomes remain untested.

The five sections above follow the [official feedback prompts](https://amazonappdev2026.devpost.com/rules), checked on 2026-10-16. The implementation used [checkout v7.0.1](https://github.com/actions/checkout/releases/tag/v7.0.1) and [setup-go v7.0.0](https://github.com/actions/setup-go/releases/tag/v7.0.0) major references.
