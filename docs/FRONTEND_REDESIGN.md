# Frontend redesign — 2026-10-15

The owner requested an easier frontend and chose a dark movie interface close
to JustWatch / Letterboxd. The main visual reference is the
[JustWatch US catalog](https://www.justwatch.com/us): clearly separated service
filters, restrained dark surfaces and a movie-card grid. Its
[provider-filter documentation](https://support.justwatch.com/article/understanding-provider-filters-in-the-just-watch-app)
also informed keeping subscription supply distinct from buy/rent.
[NightPick](https://nightpick.app/) provides a relevant group-room flow;
[Letterboxd](https://letterboxd.com/) was considered for film-focused hierarchy.
These are interaction/design references. No third-party application code,
branding, poster assets, reviews or account integrations were copied.

The Go-embedded web client remains dependency-free at runtime. The redesign
uses three workspaces: room setup, film shortlist, and group-recorded choice.
Desktop navigation becomes a compact horizontal step bar on narrow screens.
Profile, optional history import, assistant, data sources and member editing
open on demand. Streaming-service chips and runtime presets update the form;
only an explicit save changes the shared policy. Shortlist cards show actual
metadata and selected US subscription listings, with separate estimated
preference scores. Covers are original CSS typography and abstract shapes,
not official film posters.

Product authority remains unchanged: anyone with an invite may record for
another member; an acceptance record is not proof of personal consent.
All member entries must record acceptance before saving. Changed decision
inputs discard the displayed shortlist when the active proposal is invalidated.
An in-flight shortlist from another room or older decision is discarded.
Backend CAS, hard filtering, deletion, expiry and capability checks remain
authoritative. Profile recovery codes and invitation capabilities are never
included in public screenshots.

Validation is recorded separately in `evidence/frontend-redesign.json`.
Browser actions use the running Go server and real supplier-backed candidates.
The group and CSV inputs are constructed. A narrow viewport is not a physical
phone test, and the earlier owner response still makes that test NOT_RUN.
No improvement in satisfaction or decision time has been measured.

The October 14 demo video captures the previous visual design. It remains
historical evidence of that workflow and needs a new capture before it can
represent this redesign in a final competition submission.


October 17 follow-up: the local demo has now been rebuilt from declared dark-interface screenshots, including the enhanced assistant and current save flow. See `DEMO_VIDEO_V2.md` and `evidence/demo-video-v2.json`. It is explicitly a still-image edit; public upload remains pending. The original October 14 video note above describes the historical state at redesign time.
