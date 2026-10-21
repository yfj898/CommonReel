# Developer-tool feedback — CommonReel, 2026-10-21

This Alexa+ submission uses a self-hosted MCP server and a companion web
simulation. Feedback below concerns tools and documentation actually used;
no Alexa device, AWS cloud runtime, Fire TV, Bee or Ring hardware was exercised.

## 1. What we used and why

| Tool / API / SDK | Actual use |
|---|---|
| Amazon Alexa+ hackathon specification | Chose the self-hosted MCP / web simulation path and the 2025-11-25 minimum protocol |
| Official Go MCP SDK v1.8.0 | Actual Streamable HTTP server and local-agent client; typed schemas, bound room capability and eight tools |
| Go 1.26.4 + net/http | One binary, shared HTTP/MCP domain rules, controlled integration tests and private listeners |
| modernc.org/sqlite v1.36.3 | SQLite WAL, CAS, dual versions, minimal idempotency receipts, profiles and room lifecycle |
| Ollama 0.33.3 + qwen3.5:4b | Free loopback model planning real MCP calls; no paid model or cloud fallback |
| Wikidata Entity API | CC0 catalog with exact IMDb/QID joins, minute durations and explicit source genres |
| Watchmode Free API | Bounded live US subscription checks, private 24-hour cache, save-time freshness |
| Browser Web APIs / QRCode.js MIT | Independent-origin storage, fetch SSE, secure random invite tokens, local QR and bilingual UI |
| GitHub Actions | Linux race, vet/build and frontend tests per source push |
| Codex, Edge/IAB Chromium and FFmpeg | Source/test assistance, browser inspection/mobile emulation, real-interaction capture and caption-led video editing; not app runtime dependencies |

## 2. What worked well

Typed MCP handlers avoided parallel hand-maintained schemas and kept the page
and tools on one Go business module. Room capability binding in transport
headers kept secrets out of tool arguments. The simulation route made it
possible to demonstrate a real working tool workflow without gated hardware.

SQLite compiled on Windows without a C compiler. Real temporary databases
supported competing versions, identical retry storms and restart checks. Linux
WSL provided a real race run rather than treating ordinary concurrency tests as
a substitute. English/Chinese labels did not require a frontend build service.

Local Qwen successfully requested the new preview tool through the actual MCP
transport. The recorded four-member task took **9.99 seconds** and returned
0→19 eligible films; a separate three-member task took 16.35 seconds. These are
individual observations, not a throughput or success-rate benchmark.

## 3. Recorded friction and actionable suggestions

| Task and steps | Expected vs. observed | Severity | Workaround and suggestion |
|---|---|---|---|
| Explain a real shortlist with local Qwen: asked for a recommendation and first-candidate explanation | Expected tool-grounded providers; the original run invented provider names and skipped the explanation tool. Trace retained in `evidence/agent-real-v2-before-grounding.json` | High for factual display | Go renders verified facts; a missing explanation can trigger a separately labeled grounding call. Tooling examples should distinguish model text, verified tool data and final-authority actions |
| Build the metadata catalog through Wikidata SPARQL | Expected result rows; actual HTTP 429 during the earlier service rate-limit incident, retained in the original data report | Medium | Used official Entity API plus cached responses and exact joins. Show bounded Entity-API fallback examples alongside SPARQL examples; this was a service event, not a fabricated outage |
| Configure the owner's Watchmode free key | Owner opened the consumer dashboard and asked how to obtain a developer key; key provisioning was a separate developer flow | Medium onboarding | Link directly to the Developer API key page in setup. A consumer-dashboard link to developer onboarding would reduce this confusion |
| Test touch and the actual 319px panel in IAB | Mobile metrics worked, but raw `Input.dispatchTouchEvent` was unsupported and narrow screenshot capture failed | Medium for testing | Edge Chromium provided actual emulated touch events and screenshots. Surface capability differences before a long mobile/recording task |

MCP read-only annotations are hints, not application authorization. In controlled
adversarial tests, the host rejects model-requested writers in preview mode;
an ordinary-mode hypothetical write was also reproduced and repaired. Those
are our application contract tests, not claims of an SDK vulnerability or a
real-world attack. No real account quota was deliberately exhausted.

## 4. Onboarding experience

The typed Go SDK and single-binary UI reduced initial setup. A downloaded local
model and a separately requested free data key were the two owner-specific
prerequisites. The README now names their exact local configuration and exposes
model/supplier health independently. We did not measure onboarding duration.

For the recent addition, included CC0 responses let another developer reproduce
all 30 film objects offline without re-requesting live data or entering a key.
Raw supplier data and personal history remain outside the source bundle.

## 5. Would we build with these tools again?

**Yes:** Go, SQLite and the official MCP SDK form a compact foundation for this
local shared-state workflow. A small local model is useful for planning, provided
read/write authority and displayed facts are owned by application code. We would
keep the same explicit supplier-data boundaries and cache-only preview path.

The next product evidence would be real group observation and physical-device
usage; neither is replaced by mobile emulation. No hardware-experience feedback
or AWS integration is invented to fill a submission field. This report follows
the five [official feedback prompts](https://amazonappdev2026.devpost.com/rules).
