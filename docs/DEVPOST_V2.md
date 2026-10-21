# CommonReel — Preview the trade-off. Record the choice.

**Tagline:** Preview a movie-night compromise without changing the room, then save a verified group-recorded choice.

**Track:** Alexa+ — self-hosted MCP server (Streamable HTTP, 2025-11-25) with a companion web simulation. No mini-challenge is claimed.
**Repository:** [yfj898/CommonReel](https://github.com/yfj898/CommonReel)
**Video:** Owner upload pending — `dist/CommonReel-demo-2026-10-21.mp4` (approximately 2:42, English).
**Built with:** Go, official Go MCP SDK, SQLite WAL, Ollama, qwen3.5:4b, JavaScript, Watchmode, Wikidata.

## Inspiration

A group can agree on “movie night” and still disagree on every title. One
person has limited time, another skips horror, and someone has already watched
the obvious suggestion. A personal recommendation score does not resolve that
shared decision. Our target is two to four adults choosing on the same couch.
This is the product hypothesis; we have not run a human group study.

The interesting question is not simply “what should we watch?” It is “what
would become possible if we changed this one limit?” CommonReel lets a group
explore that trade-off without turning a hypothetical question into a real edit.

## What it does

- A same-Wi-Fi invite or locally generated QR lets friends join with nicknames.
  No account is needed. Each person can optionally select 1–3 explicitly liked
  films; watched history stays distinct from liking.
- Shared runtime/platforms combine with member genre vetoes and known watched
  records. The bounded 247-film CC0 catalog includes 30 checked recent films.
  Unknown metadata, expired/unknown supply and rent/buy-only offers are excluded.
- A **read-only runtime preview** evaluates the full eligible set twice with
  the same Go rules and valid cached US supply. It reports exact before/after
  and newly eligible counts, then shows up to six new films with providers/times.
- Only **Apply this runtime** changes the room. A changed room revision,
  catalog/supply fact or validity interval rejects an outdated preview.
- Every member entry must record acceptance before save. Invite holders may
  record for others, so the saved outcome is a **group-recorded choice**.
  Save refreshes supply and rejects a stale decision.
- The bright English/Simplified Chinese interface preserves drafts. Optional
  Netflix CSV and recovery-code profiles retain normalized reusable inputs;
  room edits do not automatically overwrite a profile.

## How we built it

One Go domain module serves both HTTP and a real self-hosted MCP server using
the official Go SDK over Streamable HTTP, protocol **2025-11-25**. Eight tools
include `runtime_preview`; the local Ollama agent uses those actual tools.
The host's preview mode exposes only cache-only read tools and rejects writers.
Hypothetical wording also cannot authorize a write in the ordinary assistant.
Go rechecks and renders structured tool facts rather than trusting model prose.
Final responses, apply, save, delete and profile maintenance remain page actions.

SQLite WAL transactions combine `room_revision`, `decision_version` and minimal
idempotency receipts. Response-only writes retain other members' records;
changed decision inputs invalidate candidates/responses. Apply checks its
cache-bound basis inside the existing transaction, without another table.
Supplier refresh on save happens outside the write transaction, then CAS
rejects edits made during the network wait. SSE delivers current versions on
connection and reconnect. Capability secrets stay in headers and out of model
arguments; only normalized room inputs needed for the task enter the local model.

Wikidata QIDs/IMDb IDs, unique minute-valued runtimes and explicit genres are
reproducible from retained CC0 responses. Existing external `imdb:` watched
records still match when those films enter the catalog. Watchmode availability
is a separate time-limited source; a platform listing is not an account check.
The original group ranking formula remains 0.65 × minimum + 0.35 × mean.

## Challenges and learning

An earlier real local-model response invented provider names from numeric IDs
and omitted an explanation call. We retained that failed trace, moved factual
presentation into Go and logged model-requested versus guard-requested MCP calls
separately. This release extends that separation: a model can plan a what-if,
but the host limits authority and the actual tool object supplies the card.

Catalog expansion also exposed a concrete upgrade issue: old external watched
IDs could stop excluding a film after it acquired a Wikidata catalog ID. A
restart/expansion integration test reproduced it; canonical identity comparisons
now preserve the watched meaning. Full counterfactual evaluation avoids another
trap: subtracting “runtime filtered” totals would count films still blocked by
vetoes, watched inputs or supply.

## Demonstration and validation

The English video shows actual browser interactions in a constructed four-entry
room: **81 → 96 minutes, 0 → 19 eligible, 19 newly eligible, six displayed**,
explicit application, a second-origin LAN client updating, and **Inside Out 2**
saved after four acceptance records and supplier recheck. It uses edited real
screencast takes and reading holds, original offline narration and original
cover designs, rather than generated product states or movie footage.

Windows tests/vet/build, actual Linux WSL race tests (**38 top-level Go tests
and 17 subtests**) and **9 Node tests** passed. Chromium simulation covered
independent origins, 2/3/4-member workflows, 360×800 and 390×844 touch, a 319px
sidebar and 1280px desktop, reconnection, language/draft retention, and a real
unavailable-model form workflow. Supplier-change/expiry cases use controlled
adapters/clocks through the actual HTTP/MCP business seams.

**Physical phones and human group outcomes remain untested.** We claim no
satisfaction, decision-time improvement, personal consent or playback entitlement.
The earlier MovieLens research benchmark is separate from this live workflow.

## Materials and next step

- [Setup and run instructions](../README.md)
- [Tool feedback and recorded friction](PRODUCT_FEEDBACK_V2.md)
- [Current release and evidence classes](FOUR_IMPROVEMENTS_2026-10-21.md)
- [Catalog provenance](MODERN_CATALOG_SOURCES.md)
- [YouTube title/description and upload steps](YOUTUBE_UPLOAD.md)

This is copy-ready material, not a submitted entry. The owner will upload the
public video, paste its URL, review eligibility/rights and complete final rule
acceptance/submission. The [official rules](https://amazonappdev2026.devpost.com/rules)
were checked on 2026-10-21; the deadline is 2026-10-24 03:00 Asia/Shanghai.
