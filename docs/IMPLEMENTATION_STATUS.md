# Spec execution — updated 2026-10-17

Overall: **PARTIAL — local implementation and reviewable artifacts delivered;
public video/final submission and physical-phone evidence remain open.**
Source of truth: SPEC.md. Detailed evidence: ACCEPTANCE.md.

| Gate | Current state | Evidence |
|---|---|---|
| G0 real dependencies | PASS_WITH_GUARD | Free key configured locally; real US API 200 and invalid-title 404; fixed 217-film CC0 catalog. Real local Qwen/MCP update and recommendation loops, with explicit Go grounding for omitted explanation. |
| G1 business workflow | PASS for tested HTTP/MCP scenarios | WAL/CAS, dual versions, 20 concurrent retries, competing clients, strict eligibility, responses and save freshness/races; real three-entry browser save. |
| G2 profiles/lifecycle | PASS at normal interfaces | CSV limits/matching/report, reusable copy/update boundaries, delete withdrawal, model cancellation, exact 24h access gate. No forensic-erasure claim. |
| G3 mobile/agent | Browser + local agent PASS; phone NOT_RUN | One IAB engine, localhost and private-address origins with separate browser storage; same-room SSE update and invite-fragment switch verified. Actual 360px viewport had no overflow. Owner cannot test a phone now. |
| G4 materials | CODE_PUBLIC / VIDEO_LOCAL / SUBMISSION_PENDING | Source: [yfj898/CommonReel](https://github.com/yfj898/CommonReel). English README/draft, run instructions, feedback, acceptance, source ZIP and 2:44 edited English demo. No public video URL or final submission. |

Current entry: `go run ./cmd/commonreel-v2`, port 8093, separate
`data/commonreel-v2.db`. The older 8092 prototype, database and offline
MovieLens evaluation remain independent. New functionality is not inferred
from old prototype evidence.

The model's first ungrounded recommendation failed: it added unverified
provider names and omitted an explanation call. That run remains in
`evidence/agent-real-v2-before-grounding.json`. Final text now comes from
actual tool facts. `origin=model` and `origin=grounding_guard` distinguish who
requested each real SDK call; no model-only success rate is claimed.

Real-run task timings are individual observations, not a benchmark. The
latest two-task report records a no-change member update and a recommendation
with a guarded explanation. The earlier policy run proves an actual policy
mutation; the browser demo proves an actual member mutation.

`go test -count=1 ./...`, `go vet ./...`, the v2 build and JavaScript syntax check passed again for the October 16/17 enhancement. The historical machine-readable test report enumerates pass events, including parent/subtest events. Windows `go test -race` remains NOT_RUN because its CGO compiler is unavailable. Separately, [Linux CI run 38753594156](https://github.com/yfj898/CommonReel/actions/runs/38753594156) passed race tests, vet and all-package build for the previous published commit `55896f6`; the new publication has its own CI result. No human group trials or satisfaction/time-saving results exist.

Open external steps: upload the local demo to a public YouTube/Vimeo URL, and review/complete
Devpost registration and final submission. No account agreement or final
submission was accepted on the owner's behalf.

Deadline rechecked against the [official rules](https://amazonappdev2026.devpost.com/rules):
2026-10-23 12:00 PDT = **2026-10-24 03:00 Asia/Shanghai**.


October 16/17 follow-up: the visible header assistant entry and protocol badge now expose the actual MCP path. UI traces show `origin=model` separately from `origin=grounding_guard`, refusal codes, the configured model and elapsed seconds. Actual model turns took 6.87s for a member mutation and 25.57s for recommendation/explanation on this machine; these are observations, not a benchmark. See `evidence/agent-ui-refresh.json`. The CSV fixture, post-import eligibility, acceptance gate and supply-refresh save were rerun in the browser. The 360px assistant dialog had client/scroll width 306/306; physical phone remains NOT_RUN.

The demo was rebuilt from 13 real screenshot appearances of the dark interface, with offline English narration. It is an edited still-image demo, not a continuous interaction recording. Its exact source list and checksums are committed; private `.runtime/demo-recording` frames are no longer required.
