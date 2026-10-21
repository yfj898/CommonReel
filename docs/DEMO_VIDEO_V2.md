# Current English demo — approximately 2:42

Artifact: `dist/CommonReel-demo-2026-10-21.mp4` (H.264/AAC, 1920×1080, 24 fps).
Companion files: `dist/CommonReel-demo.en.srt` and
`dist/CommonReel-demo-narration.txt`. The owner chose to upload to YouTube
personally; the public URL is **OWNER_PENDING**.

The October 21 bright bilingual version uses **84 actual Edge Chromium
screencast frames**, retained with timestamps in `evidence/demo-2026-10-21`, and
one real terminal screenshot for reading the structured preview. The edited
takes and terminal-frame holds align offline English narration. It is an
edited interaction demonstration, not an uncut timing benchmark or the earlier
October 17 still-image-only demo. No generated product states, movie posters,
movie footage or third-party music are used.

| Start | Chapter |
|---|---|
| 00:00 | Four constructed entries and optional explicit likes |
| 00:22 | 81-minute limit, three US platforms, Horror/Musical vetoes and no match |
| 00:44 | Actual local Qwen → MCP runtime_preview; 0→19 eligible, 19 new, six displayed |
| 01:16 | Explicit apply and independent LAN client receiving 96 minutes |
| 01:39 | Checked Inside Out 2 listing and partial preference evidence |
| 02:00 | Member-entry records, supply refresh and saved group choice |
| 02:21 | Reload/language continuity and separated simulation evidence |

Rebuild on Windows using the existing authoring tools:

```powershell
powershell.exe -NoProfile -File scripts/render-demo-audio.ps1
python scripts/build-demo.py
```

Narration uses offline Microsoft Zira Desktop. FFmpeg comes from the existing
`imageio-ffmpeg` authoring dependency in `.runtime/video-deps`; it is not a Go
runtime dependency. If reproducing on a new authoring machine, install
`imageio-ffmpeg==0.6.0` explicitly into that directory. The committed frame and
capture manifests provide all visual inputs. App use does not need these tools.

Decoded key scenes were inspected at 5, 30, 65, 80, 92, 109, 133 and 153 seconds.
Audio is present throughout the narration, with measured maximum -2.4 dB before
final terminal-capture encoding; no clipping was observed. The current machine
report is `evidence/demo-video-v2.json`. Subtitle times follow the generated
narration approximately; the owner can review them while uploading.

The room and all members are constructed. The 390×844 client is Chromium
emulation. Physical phones and human-group outcomes remain NOT_RUN.
[Upload copy and final owner actions](YOUTUBE_UPLOAD.md).
