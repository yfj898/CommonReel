# Local English demo

2026-10-15: the frontend has been redesigned; see `FRONTEND_REDESIGN.md`.
The local video below records the October 14 interface. Re-record it before
using it as a demonstration of the current visual design.

Artifact: `dist/CommonReel-demo.mp4`; approximately 2:44, below three minutes.
Public YouTube/Vimeo URL: **PENDING**. Verification metadata:
`evidence/demo-video-v2.json`.

The video uses actual browser screencast frames and screenshots from the
running application, edited into chapters with shortened idle waits and
offline English narration. No generated product states, human testimonials,
external music, movie posters or third-party video footage are used.

The group and Netflix-format CSV are constructed. Three member entries,
cross-origin synchronization, real local model/MCP calls, CSV classification,
real supply filtering and page-controlled save were exercised. The narrow
view is browser emulation; no physical phone was tested. Captions and the
narration explicitly state these boundaries.

The narration is `scripts/demo-narration.json`. Windows SAPI creates local
WAV chapters through `scripts/render-demo-audio.ps1`; `scripts/build-demo.py`
encodes captured local frames with an isolated imageio-ffmpeg dependency.
These are authoring tools, not runtime application dependencies.

Seven chapters cover the shared record, room inputs, genuine MCP calls,
optional CSV, verified eligibility, versioned response/save flow and evidence
limits. Review the local video before public upload. Keep credentials,
recovery codes and live invitation material out of any future recording.
