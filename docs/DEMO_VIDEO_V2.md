# Current demo — 1:30, silent with English captions

The published file is **`dist/CommonReel-demo-silent-2026-10-21.mp4`**. It is H.264,
1920×1080, 24 fps, exactly 90 seconds, with **no audio track**. Seventeen short
English caption cues are burned into the picture; understanding the demo does
not require enabling a subtitle control. A matching
`dist/CommonReel-demo-silent.en.srt` and plain caption transcript are supplied.
**Published video:** [CommonReel on YouTube](https://youtu.be/VwcLSyXDnyE).
YouTube Studio confirmed public publication on October 21 and showed a 1:30
duration with no copyright issues. Publication verification is recorded in
[the publication report](evidence/youtube-publication-2026-10-21.json).

## The story follows the decision

| Start | Scene | What the picture establishes |
|---|---|---|
| 00:00 | Four people, one film | Shared limits and optional explicit favourites |
| 00:12 | Nothing fits | 81 minutes, Horror/Musical vetoes, zero eligible films |
| 00:20 | Ask a what-if | Actual local Qwen → MCP `runtime_preview`; 81→96 minutes, 0→19 eligible, six displayed; room unchanged |
| 00:40 | Apply, then sync | Explicit page action; second-origin LAN client changes from 81 to 96 minutes |
| 00:54 | Check the facts | Inside Out 2, 96 minutes, Disney+ US listing, supply time and preference evidence |
| 01:04 | Record and save | Four acceptance entries, supplier recheck, saved group-recorded choice |
| 01:18 | Keep the result | Saved record after reload; source and evidence location |

The owner requested removal of the synthetic voice and narration-led pauses.
The new edit is 72 seconds shorter than the former approximately 2:42 version.
It uses 83 unique frames selected from the 84 retained Edge Chromium captures,
plus the existing real screenshot of the completed structured preview. Model
waiting is removed and selected frames have short reading holds. This is an
edited demonstration, not a continuous timing benchmark. The real model turn
remains the recorded 9.99-second observation in the engineering report.

## Rebuild on Windows without speech tools

```powershell
python scripts/build-silent-demo.py
```

The explicit cuts, panel text and caption times are in
`scripts/demo-silent-cuts.json`. The builder uses the existing
`imageio-ffmpeg` dependency in `.runtime/video-deps`; no SAPI, voice model,
network request or audio file is used. On a new authoring machine, install
`imageio-ffmpeg==0.6.0` there explicitly. App use does not need this authoring
dependency. The source package includes all captured visual inputs.

`scripts/build-demo.py` and `scripts/render-demo-audio.ps1` still reproduce the
earlier narrated edit at `dist/CommonReel-demo-2026-10-21.mp4`. That file and its
old SRT/transcript are preserved as history; they are not the current upload.

## Acceptance evidence

- Complete decode passed; the output has one 1920×1080 video stream and no audio stream.
- The duration is 90.00 seconds. All 17 caption cues cover the full timeline.
- Twenty-one decoded frame samples were inspected, covering every caption
  midpoint, the 81/96 client states, save-time recheck and saved record. Timed
  captions continue to change while a captured image is held.
- Short captions and side-panel facts match the actual UI: 19 newly eligible,
  six displayed, no room mutation during preview, and four acceptance records.
- Published playback passed in signed-in Edge at 1080p HD, with readable
  burned-in English captions and a 1:30 duration. Separate signed-out playback
  remains unverified: the in-app browser could not resolve YouTube, and the
  independent watch/oEmbed fetch was throttled.
- [Machine report](evidence/demo-video-silent-2026-10-21.json),
  [preview frame](evidence/demo-silent-preview-2026-10-21.png),
  [saved-choice frame](evidence/demo-silent-choice-2026-10-21.png).

Members are constructed; the portrait client is Chromium emulation. Physical
phones and human-group outcomes remain NOT_RUN. Proxy recording is allowed,
so the result is a group-recorded choice. No movie footage, copyrighted music
or generated product states are included.

[YouTube publication details, title and description](YOUTUBE_UPLOAD.md).
