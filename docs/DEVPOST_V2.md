# CommonReel — a group-recorded choice for movie night

Status: reviewable English submission draft; not posted. Track: Alexa+.

## Inspiration

A few friends in one room still have different limits: time, available
subscriptions, genre vetoes and movies they have already watched. A personal
recommendation score does not settle that shared decision. CommonReel brings
those inputs into one explicit, editable record.

This target audience and pain point are hypotheses. We have not measured
group satisfaction or time saved, and no human group study has been run.

## What it does

Two to four adults join a local room on their phones or browser. They select
tonight's US platforms and runtime, record genre preferences and vetoes, and
optionally import Netflix Viewing Activity CSV or a reusable profile copy.

Go filters a 217-film verified catalog before ranking. Unknown or conflicting
runtime, vetoed genres, disallowed rewatch and unconfirmed subscription supply
are excluded. A local model uses real MCP tools to update explicit inputs,
recommend and explain. Go renders final film facts from tool results so
generated provider names cannot become availability claims.

The page proposes a candidate and records each member entry's response.
Everyone may act for someone else, so the result is called a group-recorded
choice. All entries must record acceptance for the same decision version;
changed inputs invalidate old responses. Saving refreshes current supply and
rejects concurrent or stale decisions. Profiles keep normalized history across
nights; room copies expire after 24 hours or on close.

## How we built it

The runtime is Go, SQLite WAL, the official MCP Go SDK, Streamable HTTP
2025-11-25, browser forms/SSE and local Ollama qwen3.5:4b. Page and MCP paths
share one business module. Separate room and decision versions prevent an
individual response from clearing other responses while preserving CAS.
Idempotency stores minimal receipts, never full historical room snapshots.

Wikidata CC0 metadata is joined by stable identifiers. Watchmode provides
bounded, cached US non-image supply facts under the free plan. CSV matching
runs locally. No paid model API, AWS service, movie image or playback system
is required. This is a web simulation with a working self-hosted MCP server;
it does not connect to an Alexa device.

## Challenges and what we learned

The first real model recommendation invented platform names after reading
numeric source IDs and omitted a requested explanation tool. That run is
retained as a failure. We moved factual output ownership to Go, added a
visible MCP grounding step, and reran the workflow. A model remains useful
for planning tools, while domain rules retain authority.

Wikidata SPARQL returned HTTP 429, so the metadata builder switched to the
official entity API with exact IMDb cross-checks. We also corrected our own
initial genre-ID mapping mistakes using official English labels. Explicit
unknowns and provenance mattered more than presenting a larger catalog.

## Evidence and limitations

The repository includes controlled HTTP/SQLite/MCP tests, real supplier
success and invalid-title failure, real local-model tool traces, and browser
screens. A genuine phone test remains NOT_RUN because the owner could not
test a phone. Multi-browser compatibility and human benefits are unmeasured.
Neutral and preference-adjusted scores are estimates, not acceptance.

## Submission links

- GitHub repository: [yfj898/CommonReel](https://github.com/yfj898/CommonReel).
- Public YouTube/Vimeo demo: PENDING — local 2:44 English edited demo is ready.
- Product feedback: PRODUCT_FEEDBACK_V2.md.
- Evidence and test scope: ACCEPTANCE.md and IMPLEMENTATION_STATUS.md.

Before final submission, the owner must review identity/eligibility,
registration and all final fields. Official requirements were rechecked on
2026-10-14: [competition rules](https://amazonappdev2026.devpost.com/rules).
