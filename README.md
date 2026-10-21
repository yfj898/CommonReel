# CommonReel · A group-recorded movie choice

A Go application and self-hosted MCP server for the **Alexa+ track** of
Amazon Build, Ship, Shape 2026. The companion web app uses a **real local
Ollama model** and the official Go MCP SDK over **Streamable HTTP,
protocol 2025-11-25**. It is a web simulation, not a certified Alexa device integration.

Two to four adults on the same Wi-Fi share tonight's runtime and streaming
platforms, combine genre vetoes and known watched films, review a bounded
shortlist, and save a choice only after every member entry records acceptance.
Anyone holding the room invite may act for another member: this is a
**group-recorded choice**, not proof of personal consent.

## Run

Requires Go 1.26.4+, Ollama for the local assistant, and a free Watchmode API
key for current US subscription supply. The 247-film Wikidata CC0 catalog is
included in `assets/`. No Node build, Redis, Docker or CGO is needed.

1. Request your own free key at [Watchmode API](https://api.watchmode.com/requestApiKey/).
   Review the [API terms](https://api.watchmode.com/tc). Put it in ignored
   `.env.local`; never publish it or paste it into a model:

   ```dotenv
   WATCHMODE_API_KEY=your-own-free-key
   OLLAMA_ENDPOINT=http://127.0.0.1:11434
   OLLAMA_MODEL=qwen3.5:4b
   ```

2. Start Ollama locally, disable cloud use, and explicitly pull the model:

   ```powershell
   $env:OLLAMA_NO_CLOUD = '1'
   ollama serve
   # In another terminal:
   ollama pull qwen3.5:4b
   ```

3. From the repository root:

   ```sh
   go run ./cmd/commonreel-v2
   ```

   Open **http://127.0.0.1:8093/**. Create a room, add members and apply limits.
   Model failure leaves the forms usable. Missing supply never becomes a
   successful recommendation. `/health` reports each dependency separately.

For phones, select the computer's actual private Wi-Fi IP explicitly:

```powershell
.\scripts\start-v2.ps1 -LanIP 192.168.1.25
# Equivalent:
go run ./cmd/commonreel-v2 -lan 192.168.1.25
```

The same process listens on loopback and that private IP. Its invitation
QR/link uses the LAN address. Phones must use the same Wi-Fi; no firewall
rules or public tunnels are created. If a network blocks client-to-client
traffic, use a suitable private network rather than a public tunnel.

## The workflow

Use the language selector in the header to switch between **English** and
**简体中文**. The first visit follows the browser language; your selection is
remembered on this browser origin. Switching preserves the room, open dialogs
and unsaved form values. Names, film titles, imported history and model replies
retain their original text. Chinese genre labels are converted to the existing
canonical API values when submitted. Translation strings live in
`internal/workflow/web/i18n.js`; no translation API or frontend build is needed.

1. Share an invite; each person joins with a nickname. No account is needed.
2. Select US platforms available on tonight's device and the runtime limit.
3. Optionally select 1–3 explicitly liked films in each member editor, then
   set genre vetoes and watched inputs. Netflix Viewing Activity CSV can be skipped.
   Candidate badges distinguish explicit preference evidence from a catalog baseline.
4. Find up to six candidates. Unknown runtime, conflicting metadata,
   unconfirmed/expired supply, rentals and purchases do not qualify.
5. If time is the obstacle, use **Try a longer runtime**. It compares the full
   current and hypothetical eligible sets using valid cached supply only. No
   supplier request or room edit occurs until you explicitly apply. Other
   constraints remain fixed; unknowns stay excluded. A stale preview is rejected.
6. Propose a candidate; record accept/reject for every member entry.
   Changes to decision inputs revoke old responses immediately.
7. Save with the page button. Supply refresh occurs outside the database
   write transaction; concurrent changes reject a stale save. The same
   decision reuses its existing anonymous summary.

A profile recovery code can maintain normalized history and stable likes
across rooms. Applying it creates an independent tonight copy. Raw CSV is
never stored; ambiguous and unmatched titles stay pending. External IMDb
movie IDs can be confirmed explicitly for history and do not extend the
candidate catalog. Room changes never automatically write back to a profile.
Deleting a profile withdraws linked active copies and invalidates decisions.
Room copies are cleared on close or creation + 24 hours; profiles persist
until their recovery-code holder deletes them.

## Engineering design

- One `internal/workflow` module serves page HTTP and MCP business rules.
- Real SQLite WAL transactions, independent `room_revision` and
  `decision_version`, durable minimal idempotency receipts.
- Capability secrets are generated with browser secure random bytes,
  transmitted in headers and stored as hashes. IDs grant no access alone.
- Deterministic hard filtering precedes ranking. Default estimate is
  0.65 × minimum + 0.35 × mean; sparse users get a neutral 3/5 prior unless
  they supply explicit ratings or positive preferences. Watched is not liked.
- US subscription availability is cached for 24 hours, capped at 30 days
  retention. Each round probes at most 30 movies, global concurrency 4,
  45-second probe deadline and a configurable process request cap.
- The model plans bounded tool calls (6 calls / 90 seconds). Go renders
  factual recommendation text from tool results. An explanation omitted by
  the model can be fetched through MCP by a clearly logged grounding guard.
  No generated platform names or runtime claims are trusted.
- SSE sends immediate version snapshots on every connection, including
  reconnects. Credentials never go into SSE URL queries.
- Explicit local/private listeners, allowed Hosts and same-origin protection.
  The assistant remains loopback-only. Recovery codes cannot be recovered by email.

## Reproduce the 30-film catalog addition

The old 217 film objects and IDs are retained. The addition contains 30 movies
from 2015–2024, within the approved 2015–2025 range. Exact Wikidata QIDs/IMDb
IDs, unique minute-valued runtimes and source genres are retained with the
CC0 responses. Subscription supply is not part of this metadata snapshot.

```sh
python scripts/build-catalog-entities.py --modern-only
# Review data/catalog-v2.json and its manifest, then copy both to assets/.
```

This command uses included source responses; no API key, MovieLens download
or network request is needed. See [catalog provenance](docs/MODERN_CATALOG_SOURCES.md).
Previously confirmed `imdb:` watched IDs still exclude the corresponding film
after it enters the candidate catalog.

## MCP

Endpoint: `http://127.0.0.1:8093/mcp`.
Headers: `Authorization: Bearer <room-invite-token>` and
`X-CommonReel-Session: <session-id>`. The app's MCP client binds both; the
model never receives the secret or chooses another room.

Tools: `catalog_info`, `get_room`, `set_member`, `set_policy`, `shortlist`,
`search_movies`, `explain_candidate`, `runtime_preview` (**8 tools**).
The assistant has a host-enforced read-only runtime mode; it can never apply
a preview. Hypothetical wording does not authorize a write even in the ordinary mode. Responses, save, deletion, profile
maintenance and room creation are page operations. See [MCP contract](docs/MCP_V2.md).

## Evidence and limits

```sh
go test -count=1 ./...
go vet ./...
# Frontend checks use Node.js; no npm packages or build step are needed.
node --check internal/workflow/web/app.js
node --test internal/workflow/web_test.cjs
# Explicit opt-in only: these consume free API credits and use the local model.
python scripts/live-v2-probe.py --supplier --model
```

The live probe expects the local constructed probe-room credentials in ignored
`.runtime/live-probe.json`; ordinary users need not run it. No external calls,
model downloads or API fees run as part of `go test`.

See [execution status](docs/IMPLEMENTATION_STATUS.md),
[acceptance evidence](docs/ACCEPTANCE.md),
[data and licensing](docs/DATA_V2.md), and [catalog provenance](assets/README.md).
Review the [English submission draft](docs/DEVPOST_V2.md) and
[local demo details](docs/DEMO_VIDEO_V2.md) before external publication.
Real supplier requests, controlled faults, real model runs, browser checks,
physical phones and human outcomes are reported separately. An initial real
model answer invented provider names; that failed run is retained beside the
subsequent Go-grounded result. No satisfaction or time-saving claim is made.

The current release passed Windows Go tests, vet/build, Linux WSL race tests
(**38 top-level Go tests and 17 subtests**) and **9 Node tests**. Independent
loopback/LAN origins completed 2-, 3- and 4-member constructed workflows.
Chromium mobile emulation covered 360×800, 390×844, touch, 319px and 1280px
layouts, stale previews, disconnect/reconnect, draft-preserving language
changes, and a real unavailable-model form workflow. Supplier changes/expiry
are controlled HTTP/MCP tests, not a claim about a real phone.

Physical phone testing is **NOT_RUN**. There have been **no human group trials**. Source repository:
[yfj898/CommonReel](https://github.com/yfj898/CommonReel). The current **2:42 English real-interaction demo** is
`dist/CommonReel-demo-2026-10-21.mp4`, with a subtitle file and
[YouTube upload copy](docs/YOUTUBE_UPLOAD.md). The owner chose to upload it
personally; the public URL and final Devpost submission remain open.
The [release report](docs/FOUR_IMPROVEMENTS_2026-10-21.md) separates each evidence class.
Remote checks: [Go checks](https://github.com/yfj898/CommonReel/actions/workflows/go.yml).

The earlier MovieLens user-kNN prototype and its offline evaluation remain
independent: [legacy prototype](docs/LEGACY_PROTOTYPE.md). Its research metrics
are not live-user recommendation or group-choice results. The old entry point
uses port 8092; v2 uses 8093 and a separate `data/commonreel-v2.db`.

Code: MIT. Wikidata structured metadata: CC0. Vendored QRCode.js: MIT, with
its license beside the script. Watchmode data and model weights have their
own terms; raw supplier caches, credentials, CSV, model weights and databases
are not redistributed.
