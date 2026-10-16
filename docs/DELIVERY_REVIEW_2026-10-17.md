# Review of the supplied delivery changes — 2026-10-17

The supplied material combines the existing approved user stories/specification with five proposed delivery fixes. The scope and shared-room semantics remain those in `SPEC.md`; this work applies the supported delivery changes rather than restarting the requirements interview.

| Proposal | Decision and implementation |
|---|---|
| Downgrade checkout/setup-go from v7 | Rejected factual premise. Official checkout v7.0.1 and setup-go v7.0.0 are published, and the existing Linux workflow passed. Keep the current v7 references and race/vet/build commands. |
| Organize feedback into five official questions | Applied, retaining the real failed model run, supplier limits, simulation scope and evidence links. Removed an unmeasured 30-minute onboarding claim. |
| Strengthen the Devpost story | Applied: shared decision versions, observed provider-name hallucination, Go grounding and transport binding lead the explanation. Concurrent collisions are a controlled test case, not a human-study finding. No claim of automatic Open Source mini-challenge eligibility. |
| Rebuild dark-interface video | Applied with committed real screenshots and a per-chapter source manifest. Missing declared captures fail explicitly. Normalize dimensions before concat so different-size screenshots keep their intended duration. The video explicitly identifies itself as a still-image edit. |
| Expose assistant and trace origins | Applied: header shortcut, simulation/protocol badge, model/guard origin, refusal code, model and elapsed time. A missing origin is `unreported`, not automatically attributed to the model. Existing backend metadata is reused. |
| Recompute source hashes | Applied using LF-normalized UTF-8 file bytes, preserving `canonical_source_sha256` semantics rather than hashing platform-specific CRLF bytes. |

## Current checks

- `go test -count=1 ./...`: PASS; no domain behavior was changed.
- `go vet ./...`: PASS.
- `go build -o .runtime/commonreel-submission.exe ./cmd/commonreel-v2`: PASS.
- `node --check internal/workflow/web/app.js`: PASS.
- Real browser model turn: explicit Alex Horror veto mutation, 6.87 seconds; real `get_room` and `set_member` calls.
- Real recommendation turn: 25.57 seconds; four model-origin calls and one `grounding_guard` explanation. This is a single run, not a model success-rate or latency benchmark.
- Constructed CSV: 4 rows, 1 reliable movie match, 1 duplicate, 2 pending; a colon alone does not mark an episode.
- Latest page shortlist: 5 qualifying films after import; known watched Home Alone excluded.
- Response gate: one acceptance leaves Save disabled; two acceptances allow a real supplier refresh and save; completed button stays disabled.
- Desktop: viewport/document width 1039/1039. Narrow: 360/345. Assistant dialog: client/scroll width 306/306. No horizontal overflow observed.
- Video: full decode and all 13 image-hold samples inspected. See `evidence/demo-video-v2.json` for duration and digest.

Phone remains NOT_RUN per the owner's earlier response. No human group study, satisfaction result or measured time-saving claim exists. Public YouTube/Vimeo upload and final Devpost submission remain pending.

## Primary sources checked

- [Official competition rules](https://amazonappdev2026.devpost.com/rules): five feedback questions, Alexa+ web simulation path, public English demo and separate Open Source contribution requirements.
- [checkout v7.0.1](https://github.com/actions/checkout/releases/tag/v7.0.1) and [setup-go v7.0.0](https://github.com/actions/setup-go/releases/tag/v7.0.0): published official versions; no downgrade justified.
