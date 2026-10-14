# Tool feedback and friction log

Observed 2026-10-14; factual, limited to this Windows machine and prototype.

| Tool | Purpose / worked | Observed friction / response |
|---|---|---|
| MCP Go SDK v1.8.0 | Real server + client, tools/list and calls over Streamable HTTP 2025-11-25; shared business rules | Client must bind room capability and reject extra room fields. Native schema validation helped reject unknown fields. No SDK defect was established. |
| SQLite / modernc | WAL, durable CAS and minimal receipts; restart and competing writes tested | Full JSON session copies simplify atomicity but are a scaling limit for very large pending histories. Network refresh must stay outside the write transaction. |
| Ollama 0.33.3 + qwen3.5:4b | Zero paid API; actual local tool planning | First load plus a tiny tool request took about 86 seconds. Warm tasks were much faster on this one GPU; this is not a benchmark. A real answer hallucinated provider names and skipped explanation. Go-grounded output and explicit guard-origin traces were added. |
| Watchmode Free API | Exact IMDb match and US source requests succeeded; invalid title returned 404 | The consumer dashboard differs from API-key signup. Separate developer signup guidance was needed. Free quota, caching, attribution and subscription-vs-rental distinctions must be explicit. No deliberate real-quota exhaustion was attempted. |
| Wikidata | CC0 entity metadata and stable identifiers, repeatable 217-film snapshot | SPARQL returned 429 during service rate limiting; entity API fallback worked. Our own wrong genre-ID assumptions were corrected by fetching official labels, rather than treating a source-ID guess as metadata. |
| Browser APIs | Secure random bytes work on LAN HTTP; SSE and forms work at 360px | In-page invite fragment changes initially left the old room rendered. Hash-change handling and stale-response guards were added, then checked across two origins. Clipboard has a text-selection fallback. |
| QRCode.js | Local QR generation, no third-party image request | Vendored MIT license retained. Invitations are capabilities; they are excluded from public screenshots/video. |

The initial model failure is in `evidence/agent-real-v2-before-grounding.json`;
subsequent measured output is in `evidence/agent-real-v2.json`. Guarded tool
calls are not counted as model-requested tool calls. No unsupported percentage
success rate, phone success or group satisfaction is inferred.

Windows `go test -race` could not run because a CGO compiler was unavailable.
Ordinary tests and `go vet` ran; they do not substitute for a race-detector run.
