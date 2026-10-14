# CommonReel — a film everyone can say yes to

Go MCP service and a working Alexa+ web simulation for the **Amazon Build,
Ship, Shape Developer Hackathon 2026**. Started on 2026-10-13 during the
submission window. Primary track: **Alexa+**.

CommonReel takes each member's likes, explicit genre vetoes and optional
anonymous rating history. It builds a shared shortlist, explains every
member's estimated match, and saves a choice with its room revision.

The companion uses real **MCP 2025-11-25 / Streamable HTTP** calls. Its local
conversation router is explicitly scripted, not an LLM or a live Alexa+
connection. Any compatible agent can connect to the same MCP tools.

## Run in five minutes

Requires Go 1.26.4+; no Node build, Docker, Redis or CGO is needed for the app.

Read [the separate dataset terms](docs/DATA.md), then:

```sh
go run ./cmd/fetch-data
go run ./cmd/commonreel
```

Open **http://127.0.0.1:8092/** and click **Try a three-person night**.
MovieLens is downloaded directly from GroupLens to gitignored local storage.
The application refuses to silently replace missing data with invented data.

On Windows, the quickstart is:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start.ps1
```

If the Go download client times out on your network, Windows' native download
can be used without changing global networking settings:

```powershell
New-Item -ItemType Directory -Path data\ml-1m -Force | Out-Null
Invoke-WebRequest 'https://files.grouplens.org/datasets/movielens/ml-1m.zip' -OutFile data\ml-1m\ml-1m.zip -TimeoutSec 120
go run ./cmd/fetch-data -archive data/ml-1m/ml-1m.zip
```

For a completely offline, **invented-data** smoke demo only:

```sh
go run ./cmd/commonreel -demo -db data/invented-demo.db
```

## What to try

1. Start the three-person example. Names and groups are fictional; optional
   histories are real, anonymous MovieLens research profiles.
2. Compare **Balance the group**, **Group average**, and **Catalog popularity**.
3. Edit a member's vetoes. No recommended film can contain a vetoed genre.
4. Ask **Why the first pick?** or **We can rewatch something**.
5. Choose a film, reload, and see its durable receipt.
6. Copy the room link into another browser tab. Changes propagate through SSE.

Room links are random local collaboration capabilities. Anyone with the link
can edit that room. The app is loopback-only; an authenticated hosted build
would be separate deployment work. No playback rights, streaming availability,
age classification, movie runtime, or real Alexa+ device access is asserted.

## Go architecture

- Official MCP Go SDK v1.8.0, explicitly restricted to protocol 2025-11-25.
- A stateless Streamable HTTP endpoint; application state lives in SQLite.
- Atomic compare-and-swap room revisions prevent lost concurrent updates.
- Idempotency keys bind an exact request to its original durable response.
- Bounded SSE subscribers and a 64-profile prediction cache.
- Localhost host checks, cross-origin protection and a 64 KiB MCP body limit.
- A global chronological training split: no future rating influences profiles,
  neighbors, priors or personalized predictions.

Collaborative estimates use centered user-user cosine similarity, minimum four
co-rated films, overlap shrinkage, 40 positive neighbors, and prior blending.
The default group rank is **0.65 × lowest member estimate + 0.35 × average**,
after hard veto and optional previously-rated-film exclusion.

## MCP integration

Endpoint: `http://127.0.0.1:8092/mcp`

Tools: `catalog_info`, `create_room`, `get_room`, `set_member`, `remove_member`,
`set_policy`, `shortlist`, `save_choice`, `search_movies`.

Mutations require the current room revision and an operation key. The room ID
belongs in tool arguments. A save is revalidated against the current shortlist.
See [protocol notes](docs/MCP.md) and run a genuine network probe:

```sh
go run ./cmd/mcp-check
```

The same binary can serve stdio MCP: `go run ./cmd/commonreel -stdio`.

## Observed evaluation

MovieLens 1M: **1,000,209 ratings**, **6,040 users**, **3,883 films**.
Global chronological split: **800,168 training / 200,041 held-out ratings**.
Fixed-seed sample: **200 of 843 eligible users**, **10,291 positive held-out
items**, each ranked against **1,991–3,135** unseen training-vocabulary films.

| Individual ranking | Popularity baseline | Personalized user-kNN |
|---|---:|---:|
| Recall@10 | 0.02554 | 0.03311 |
| NDCG@10 | 0.13161 | 0.15658 |
| Hit rate@10 | 0.520 | 0.560 |

These are one local offline run, **not group satisfaction or production
impact**. The algorithm was specified before this held-out run; no test-set
parameter tuning was performed. This is an engineering baseline, not a novel
recommendation algorithm or a state-of-the-art claim.

```sh
go test -count=1 ./...
go vet ./...
go run ./cmd/evaluate -users 200
```

Full protocol, dataset hashes, sample IDs, latency scope and limitations are
in [the machine-readable report](docs/evidence/evaluation.json).

## Competition materials

- [English Devpost draft](docs/DEVPOST.md)
- [2:45 demo storyboard](docs/DEMO_VIDEO.md)
- [Truthful tool feedback and friction log](docs/PRODUCT_FEEDBACK.md)
- [Submission readiness](docs/READINESS.md)
- [Evidence-backed Chinese resume bullets](docs/RESUME_ZH.md)

Registration, repository publication, public English video upload, and final
submission remain separate external milestones. No award is guaranteed.

Code is MIT licensed. MovieLens has separate research-only terms and is not
redistributed with this repository. No GroupLens or Amazon endorsement is implied.
