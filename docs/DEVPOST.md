> Historical prototype document (2026-10-13). Current v2: [DEVPOST_V2.md](DEVPOST_V2.md).

# Devpost draft — CommonReel

Project name: CommonReel
Tagline: A movie night with room for every taste.
Primary track: Alexa+ — self-hosted MCP and an explicitly labeled web simulation.
Mini challenges: none claimed. No AWS integration or extra open-source
contribution has been implemented or verified.

## Inspiration

Choosing a movie with friends is a small negotiation. A high average rating
does not tell us whether one person will be miserable. We wanted an assistant
that remembers each person's boundaries and makes the compromise visible.

## What it does

CommonReel creates a shared movie-night room. Each person adds favorite genres,
hard genre vetoes, and optionally an anonymous research rating profile. The
service filters candidates, estimates each person's enjoyment, and balances
the least enthusiastic person's estimate with the group's average. A companion
web app demonstrates a guided Alexa+ conversation using actual MCP calls.

People can inspect the explanation, change a constraint, compare the average
and popularity baselines, and save a film. The saved choice keeps its source
revision and explanation. A second local tab sees changes through live events.

## How we built it

Go owns the MCP endpoint, data loader, recommendations, SQLite room state,
optimistic concurrency, exact-request idempotency and live room updates. The
official Go MCP SDK serves Streamable HTTP at protocol 2025-11-25. The browser
calls those tools instead of using a separate mutation API.

Collaborative estimates use centered user-user similarity with overlap
shrinkage and prior blending. Every genre veto is enforced before ranking.
The default score gives 65% weight to the lowest member estimate and 35% to
the average. This is an explainable engineering choice, not a new recommender
algorithm claim.

## What we measured

We downloaded MovieLens 1M from GroupLens for a local research benchmark.
A single global chronological split gives 800,168 training and 200,041 test
ratings. On a fixed-seed sample of 200 eligible users, personalized NDCG@10
was 0.1566 versus 0.1316 for the popularity baseline. Recall@10 was 0.0331
versus 0.0255. Each user was ranked against 1,991–3,135 unseen eligible films.

These are individual offline results. Constructed group replay and
reliability tests are separate evidence classes. We have not measured human
group satisfaction or production conversion.

## Challenges and learning

Real HTTP MCP integration exposed the need to treat room persistence separately
from transport sessions. Exact-request idempotency matters when the client is
unsure whether a choice was saved. Optimistic revisions ensure an old shortlist
cannot silently override a new veto. Data licensing also shaped the project:
raw MovieLens files are downloaded locally and excluded from the repository.

## Scope and provenance

All project source was newly created on 2026-10-13. Third-party dependencies
are the official MCP Go SDK and modernc SQLite; MovieLens is separate data.
AI-assisted development was used and should be disclosed in the final entry.
The companion is scripted, not a live Alexa+ device or an LLM integration.
No streaming availability, playback rights, or commercial dataset permission
is claimed. Fictional demo names represent anonymous research profiles.

## Links to complete before submission

Repository URL: pending publication.
Public English demo video (<3 minutes): pending recording and upload.
Working build: repository quickstart; the judge can run it locally.
Product feedback: see PRODUCT_FEEDBACK.md.

Official contest: https://amazonappdev2026.devpost.com/
Deadline: 2026-10-24 03:00 Asia/Shanghai.
