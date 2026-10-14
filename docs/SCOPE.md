# Confirmed product scope — 2026-10-13

The user confirmed the full requirements in Q31. [PRODUCT_REQUIREMENTS.md](PRODUCT_REQUIREMENTS.md) is the authoritative implementation scope; [PRODUCT_DISCOVERY.md](PRODUCT_DISCOVERY.md) records the decisions.

[SPEC.md](SPEC.md) defines the implementation contracts, state changes, interfaces and acceptance scenarios. [SPEC_TRACEABILITY.md](SPEC_TRACEABILITY.md) maps the confirmed requirements to those scenarios. New acceptance scenarios start as NOT_RUN.

The target is 2–4 adults selecting a film together through phones on the same Wi-Fi, using a verified catalog of roughly 200–300 films, US subscription availability, optional Netflix CSV import, reusable profiles maintained by recovery codes, and a zero-fee local agent through Go MCP. Members may edit or decide for others; the saved result is a group-recorded choice, not proof of personal consent.

New inputs invalidate acceptance records. Sessions expire after 24 hours or close, reusable profiles persist until deletion, and personal-data-free choice summaries may remain. Model failure falls back to forms; final acceptance/save/delete actions are page controls. New functionality is not yet implemented or verified.

## Existing prototype scope and evidence

The following describes the earlier implemented prototype and is retained for its evidence boundary.

# CommonReel

Started 2026-10-13 during the Amazon Build, Ship, Shape submission window.
Primary track: Alexa+, self-hosted MCP / explicitly labeled web simulation.
Deadline: 2026-10-24 03:00 Asia/Shanghai (2026-10-23 12:00 PDT).

Problem: the most popular movie can still be a poor choice for one person.
CommonReel balances each member's predicted enjoyment, honors explicit vetoes,
and explains the compromise before the group saves a choice.

Core path: create room -> add preferences -> shortlist -> inspect tradeoffs ->
change a constraint -> save a choice with its room revision.

Go owns the MCP tools, data loader, scoring, persistent room state, optimistic
concurrency, idempotency, live room events, and evaluation. The web companion
is a guided Alexa+ simulation using actual Streamable HTTP MCP calls. A
scripted local assistant must be labeled as scripted. It is not a live Alexa+
device integration, and does not establish streaming rights or availability.

Data: MovieLens 1M, downloaded from GroupLens into gitignored local storage.
No demographic files, posters, trailers, streaming catalogs, or raw ratings
are bundled. A tiny invented catalog is available only with an explicit demo
flag and cannot be used as real-data evidence.

Validation: chronological split, popularity and personalized baselines,
held-out single-user ranking, separate simulated group-constraint tests,
durability/idempotency/concurrency tests, and a real MCP HTTP round trip.
Human group satisfaction requires a human study and remains unmeasured.

External milestones: Devpost registration, repository publication, English
video under three minutes, truthful product feedback, final submission.
Local implementation does not establish any of these external milestones.
