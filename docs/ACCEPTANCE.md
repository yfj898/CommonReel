# v2 acceptance evidence

2026-10-14. Engineering tests, real integrations, browser checks and human
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
| AT22 | 217 fixed films, official genre-label map, runtime units and manifest hashes; real US API 200 and invalid-title 404, separate controlled 429 | PASS |
| AT23 | One IAB engine, two origins with separate browser storage; actual 360×800 viewport. Owner said a phone cannot be tested now. No independent browser-engine or human group trial | Browser PASS; phone NOT_RUN |
| AT24 | English draft, run instructions, tool feedback, source published at [yfj898/CommonReel](https://github.com/yfj898/CommonReel), local source bundle and local edited demo; public video and final submission not done | PARTIAL |

Reproduce the controlled seam tests with `go test -count=1 ./...` and inspect
`evidence/tests-v2.json`. These use real temporary SQLite and actual MCP SDK
transport. Supplier/model failure scenarios use explicit adapters. The live
model and supplier reports are separate files. `go vet ./...` passed.
`go test -race` is NOT_RUN because this Windows environment lacks the required
CGO compiler; ordinary concurrency tests do not replace it.

Browser scope is precise: localhost and the selected WLAN address in the
same desktop IAB, not separate people, hardware or network paths. A member
joined on the other origin and appeared by SSE; switching invite fragments
was reproduced, fixed and checked. A four-row CSV fixture yielded one match,
one duplicate and two pending titles. The chosen movie was saved after three
acceptance records; this proves the record workflow, not anyone's consent.

The physical phone status follows the owner's explicit response on
2026-10-14. No satisfaction, retention, decision-time improvement or production
availability statistic is inferred. Dataset coverage and source facts can
change; unknowns stay excluded. Runtime/source facts and scores remain distinct.
