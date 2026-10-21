# Current implementation — 2026-10-21

**LOCAL_COMPLETE / SIMULATION_PASS / VIDEO_PUBLIC / OWNER_DEVPOST_PENDING**

| Scope | Current result | Evidence |
|---|---|---|
| Catalog + cold start (P1) | 247 CC0 films; retained 217 + 30 recent; reproducible exact IDs/runtime/genres; optional 1–3 explicit likes; baseline markers; old IMDb watched matching | MODERN_CATALOG_SOURCES.md; catalog/upgrade tests |
| Runtime what-if (P1) | Complete before/after/new counts; cached facts only; version-bound explicit apply; eighth MCP tool and host-enforced read-only mode | MCP_V2.md; runtime/agent HTTP tests |
| Mobile simulation (P2) | Independent origins and 2/3/4 constructed-member workflows; 360×800 / 390×844 touch; 319px / 1280px; reconnect; drafts/language; unavailable-model forms | FOUR_IMPROVEMENTS_2026-10-21.md and machine report |
| Source, feedback, video (P0) | Current bright English/Chinese UI, setup/docs/feedback, reproducible source package, 1:30 silent real-interaction video with 17 burned-in English caption cues | DEVPOST_V2.md; DEMO_VIDEO_V2.md; YOUTUBE_UPLOAD.md |
| Physical phones / human outcomes | NOT_RUN, separately retained | No real phone or group participants |
| Public video / final submission | [YouTube video published](https://youtu.be/VwcLSyXDnyE); final Devpost acceptance/submission remains owner-operated | evidence/youtube-publication-2026-10-21.json |

Current checks: **38 top-level Go tests + 17 subtests** on Windows and Linux WSL
with race; vet/build; JavaScript syntax and **9 Node tests**. The final published
commit's GitHub Actions run is pinned in `dist/CommonReel-delivery.json` alongside
the source ZIP. Do not infer that an earlier commit's CI validates later edits.

Run `go run ./cmd/commonreel-v2` (default 8093). This machine's independent
acceptance instance uses 8096 and a different database; the older 8092 MovieLens
prototype remains a separate research entry point. Real local Qwen and MCP
calls, supplier requests, controlled faults, browser simulation, physical
phones and human outcomes are distinct evidence classes.

The original model hallucination and dated October 14–18 evidence remain
available as history. Current contracts and acceptance are [SPEC.md](SPEC.md),
[MCP_V2.md](MCP_V2.md), [ACCEPTANCE.md](ACCEPTANCE.md), and the
[October 21 report](FOUR_IMPROVEMENTS_2026-10-21.md).

The [official rules](https://amazonappdev2026.devpost.com/rules), checked
2026-10-21, still allow the Alexa+ web simulation and require a public English
YouTube/Vimeo video under three minutes. Deadline: October 23, 12:00 PDT =
**October 24, 03:00 Asia/Shanghai**. Final submission accepts the rules contract;
the owner performs that action.
