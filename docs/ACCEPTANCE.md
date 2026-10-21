# v2 acceptance evidence

Updated 2026-10-21. Engineering tests, real integrations, browser checks and human
outcomes are different evidence classes. PASS below refers to the described
scenario, not exhaustive proof across devices, networks or third-party data.

| Spec | Observed evidence / boundary | State |
|---|---|---|
| AT01 | Independent HTTP requests, 2–4 members, fifth rejected; three entries seen across two browser origins | PASS |
| AT02 | Wrong room/profile capabilities denied; session capability cannot read reusable profile | PASS |
| AT03 | 20 identical concurrent retries commit once; two competing revisions yield one winner; key reuse with changed payload rejected | PASS |
| AT04 | SQLite restart restores room state and minimal receipt; no full room JSON replay | PASS |
| AT05 | Runtime equality/overflow, unknown/conflicting metadata, merged vetoes and watched exclusion; fixed controlled catalog | PASS |
| AT06 | Only US + selected source + sub; rent, buy and CA fail | PASS |
| AT07 | Controlled timeout, 429 pause, expired facts and unknown exclusion; no paid fallback. Real quota exhaustion was not attempted | PASS, controlled faults |
| AT08 | Empty result counts and ranking modes retain identical eligibility; no silent relaxation | PASS |
| AT09 | Missing acceptance refuses save; three browser entries record responses without clearing prior entries | PASS |
| AT10 | Input changes and expired chosen facts revoke proposal/responses atomically; pure responses retain decision version | PASS |
| AT11 | Real save refresh; controlled changed/failed supply refuses; network refresh permits concurrent policy edit and rejects stale save; retry reuses receipt | PASS |
| AT12 | BOM, comma/quoted newline, duplicate, ambiguous, unknown header, unmatched, explicit known episode, UTF-8 rejection, 8 KiB/50,000-row/5 MiB caps. Real page import used a constructed fixture | PASS, fixtures |
| AT13 | Seen does not create positive score; replace import preserves manual inputs; successful save leaves watched counts unchanged | PASS |
| AT14 | Profile import/copy, independent room edits, profile update not implicit, explicit reapply | PASS |
| AT15 | Deleted profile read/replay denied, active copies withdrawn, inflight model cancelled; minimal receipts cannot restore history | PASS at normal interfaces |
| AT16 | Exact creation +24h reads and ordinary retries denied; startup/minute cleanup implemented; forensic erasure is not claimed | PASS for access gates; disk forensics out of scope |
| AT17 | Immediate version snapshot on connection; reconnect after intentional app restart and cross-origin updates observed. SSE contains versions/status only | PASS |
| AT18 | Allowed private Host/same Origin accepted; unknown Host/cross Origin refused. LAN HTTP create/import/save worked on this computer; 360px viewport had no horizontal overflow | PASS for tested browser |
| AT19 | Unknown tool, other-room fields, ambiguous speaker and forbidden operations refused through actual SDK/controlled model | PASS for tested adversarial cases |
| AT20 | Real local model updates via MCP; recommendation explanation uses logged Go grounding when model omits it. Initial ungrounded model run failed and is retained | PASS_WITH_GUARD |
| AT21 | Closing and profile deletion cancel model requests; private history/capability absent from model body; no raw chat persistence | PASS |
| AT22 | 247 fixed films, retained 217 + checked 30 recent; reproducible CC0 identifiers, minute runtimes and explicit genres; real supply and controlled failures separated | PASS |
| AT23 | IAB/Edge Chromium mobile emulation, loopback and actual private-IP origins, 2/3/4 constructed groups; 360×800 / 390×844 touch, 319px and 1280px; reconnect and drafts | Simulation PASS; physical phone and human outcomes NOT_RUN |
| AT24 | English current source/docs/feedback, approximately 2:42 real-interaction video, subtitles and upload copy; owner takes YouTube upload and final Devpost submission | Local materials PASS; owner steps OPEN |
| AT25 | Exact runtime before/after/new sets, six-display truncation only after counts, no preview write or supplier call, stale facts/revisions/expiry rejected, apply idempotent | PASS |
| AT26 | Eighth MCP tool, host read-only mode rejects writers, hypothetical ordinary requests cannot authorize edits, cards use validated tool objects | PASS |
| AT27 | Optional explicit film likes, baseline/partial evidence labels, unchanged scoring, old external IMDb watched identity after catalog expansion | PASS |

Reproduce controlled HTTP/MCP seams with `go test -count=1 ./...`; temporary
SQLite databases and the actual MCP SDK are used. Supplier/model failure
adapters and clocks are explicit fixtures. The October 21 run passed **38
top-level tests and 17 subtests**, vet/build, and **9 Node tests**. The same Go
cases passed actual Linux WSL `-race`; Windows local race was not run because
that environment lacks its CGO compiler.

See [the current matrix](FOUR_IMPROVEMENTS_2026-10-21.md) and
[evidence/four-improvements-2026-10-21.json](evidence/four-improvements-2026-10-21.json)
for individual browser, real model, controlled-fault and layout results. The
recorded demo's 0→19 result is a particular valid cache snapshot; it is not a
catalog-wide availability promise or a user-outcome metric. Physical phones
and human group satisfaction/decision-time outcomes remain NOT_RUN.
