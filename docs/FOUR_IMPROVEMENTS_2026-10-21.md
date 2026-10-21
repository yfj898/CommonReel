# Four improvements — release evidence, 2026-10-21

**Local implementation and simulated acceptance complete.** The owner will
upload the English video to YouTube and perform final Devpost submission.
The actual submitted commit and its CI run are pinned in the delivery manifest
produced with the source ZIP (`dist/CommonReel-delivery.json`).

## Catalog and cold start

The catalog contains **247** films: the original 217 objects/identifiers plus
30 checked films from **2015–2024**. The fixed modern seeds, seven official
responses, exact IMDb joins, unique minute runtimes, explicit genre mappings,
exclusions and offline builder are retained. Both `Magic Realist` and
`Apocalyptic` have Chinese canonical labels; Apocalyptic is not Post-Apocalyptic.

Member editors offer optional 1–3 explicit likes through the existing patch
contract. Watching is not liking. Candidate badges mark full/partial explicit
evidence or the catalog baseline; scoring remains 0.65 × minimum + 0.35 × mean.
Old `imdb:` watched records remain effective after a film is added to the catalog.

## Preview before changing the room

The page and eighth MCP tool share a cache-only comparison. Both complete
eligible sets are evaluated, then the genuinely new set is counted before its
six-card display limit. The page can apply only a nonempty, current preview;
room/fact/expiry changes reject the old basis. Applying changes only runtime
and invalidates the previous candidate/responses using the existing transaction.

A recorded local Qwen turn used `runtime_preview` once (`origin=model`):
81 → 96 minutes, 0 → 19 eligible films, 19 newly eligible, six displayed,
zero unknown under that proposed limit; **9.99 seconds**. This is one observed
constructed task, not a latency benchmark. A separate three-member run took
16.35 seconds and returned 2 → 14 eligible, 12 newly eligible, 43 unknown.
Different cached coverage produces different counts; previews never fill it in.

## Chromium simulation, not a phone study

One Go instance used the independent acceptance database
`.runtime/four-improvements-acceptance.db`, loopback and the actual WLAN
private-address listener. Clients on different origins used separate storage.
IAB and Edge are both Chromium-based; no Safari or physical mobile engine was tested.

| Scenario | Observation | Evidence class |
|---|---|---|
| 2 constructed members | Invite from UI, independent-origin join, liked Arrival, language retained `Comedy, Sci-Fi` draft, preview 80→95 counted 0→6, competing edit disabled old preview, reapply 81→96, Inside Out 2 saved 2/2 and reloaded | Browser PASS |
| 3 constructed members | Optional likes skipped; baseline badge, real local-model preview, explicit apply, responses and save/reload for Hellraiser: Bloodline 3/3 | Browser + real model PASS |
| 4 constructed members | Optional Inside Out 2 like, Horror/Musical vetoes, room capacity, zero results, preview/apply, 4/4 records, Inside Out 2 save and refresh | Browser PASS |
| Model unavailable | Only the acceptance instance was restarted with unavailable loopback model port; actual `MODEL_UNAVAILABLE` result, then all four-member form operations completed; normal Ollama endpoint restored | Real local fault PASS |
| Offline and reconnect | Edge network emulation produced `net::ERR_INTERNET_DISCONNECTED` and `Reconnecting…`; other client recorded a response; recovery showed 2/3 records without a new edit | Browser fault PASS |
| 360×800, 390×844 | Mobile metrics and touch emulation; actual CDP touch taps in Edge, no horizontal overflow | Chromium mobile simulation PASS |
| 319px / 1280px | Measured client and scroll widths equal; visible control bounds checked at 319px; Chinese and English screenshots retained | Layout PASS |
| Supply changed/expired/unknown; multiple blockers | Controlled supplier/clock through real HTTP and SQLite; unrelated room refresh changed facts without changing this room revision, and stale apply was refused | Integration PASS, controlled |
| Physical phone / real Wi-Fi path from another device | No physical phone was available | NOT_RUN |
| Human group satisfaction / decision time | No real group participants or study | NOT_RUN |

The IAB lacks raw touch injection and its 319px screenshot path failed; Edge
Chromium provided the touch and screenshot evidence. This is a tooling-surface
boundary, not physical-phone evidence. Mobile simulation never updates the
physical-phone or human-outcome status.

## Checks and reproducible artifacts

- Windows Go: **38 top-level tests, 17 subtests**, no failures; vet and all-package build passed.
- Ubuntu-24.04 WSL / Go 1.26.4: the same **38 + 17** passed with actual `-race`; Windows local race is not claimed.
- Frontend: app/i18n syntax checks and **9 Node tests** passed.
- New public-interface tests cover exact counts beyond six, no room/supplier side effect, apply invalidation/idempotency, competing versions, changed/expired facts, unknowns and composite blockers, ordinary-mode hypothetical write rejection, structured read previews and legacy watched IDs.
- Compact machine report: [four-improvements-2026-10-21.json](evidence/four-improvements-2026-10-21.json).
- Real model trace: [recorded preview](evidence/four-improvements-demo-agent-preview.txt).
- Current demo: [method and observations](DEMO_VIDEO_V2.md); 84 actual screencast frames and one terminal capture are included for rebuilding.

## Submission handoff

[Devpost copy](DEVPOST_V2.md), [tool feedback](PRODUCT_FEEDBACK_V2.md),
[YouTube copy](YOUTUBE_UPLOAD.md), setup and source package describe this release.
The MP4 is approximately 2:42. Public upload is **OWNER_PENDING**, as explicitly
chosen by the owner; final rule acceptance and submission are also owner actions.
Keep the submitted commit and video fixed before **2026-10-24 03:00 Asia/Shanghai**.
