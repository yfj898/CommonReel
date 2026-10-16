"""Encode an edited, genuine UI capture with offline English narration.
Uses versioned real browser screenshots declared in demo-captures.json and
offline SAPI WAV files. Private browser recordings are not required.
"""
import concurrent.futures, hashlib, json, subprocess, sys, wave
from pathlib import Path

root = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(root / ".runtime/video-deps"))
import imageio_ffmpeg

ffmpeg = imageio_ffmpeg.get_ffmpeg_exe()
work = root / ".runtime/demo-encode"
work.mkdir(parents=True, exist_ok=True)
parts = json.loads((root / "scripts/demo-narration.json").read_text(encoding="utf-8"))
evidence = root / "docs/evidence"
capture_manifest = json.loads((root / "scripts/demo-captures.json").read_text(encoding="utf-8"))
if len(capture_manifest["chapters"]) != len(parts):
  raise RuntimeError("Capture and narration chapter counts differ")
sources = [[evidence / name for name in chapter["files"]] for chapter in capture_manifest["chapters"]]


def quote(path):
  return str(path.resolve()).replace("\\", "/").replace("'", "'\\''")


def filter_path(path):
  return quote(path).replace(":", "\\:")


durations = []
for i in range(len(parts)):
  with wave.open(str(root / f".runtime/demo-audio/{i}.wav")) as wav:
    durations.append(wav.getnframes() / wav.getframerate())
if sum(durations) >= 175:
  raise RuntimeError("Narration needs to remain below three minutes with margin")


def encode(i):
  duration = durations[i]
  paths = sources[i]
  if not paths or any(not p.is_file() for p in paths):
    raise RuntimeError("Missing real capture source for chapter " + str(i))
  # Normalize dimensions before concat. Reinitializing the fps filter for a
  # differently sized screenshot can reset its timestamps and lose a hold.
  normalized = []
  for n, path in enumerate(paths):
    still = work / f"{i}-{n}-still.png"
    subprocess.run([
        ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-i", str(path),
        "-vf", "scale=1280:1080:force_original_aspect_ratio=decrease,pad=1280:1280:(ow-iw)/2:80:color=0x101217,setsar=1,format=rgb24",
        "-frames:v", "1", str(still),
    ], check=True)
    normalized.append(still)
  listing = work / f"{i}.ffconcat"
  lines = ["ffconcat version 1.0"]
  for path in normalized:
    lines.extend(
        ["file '" + quote(path) + "'", "duration " + str(duration / len(paths))]
    )
  lines.append("file '" + quote(normalized[-1]) + "'")
  listing.write_text("\n".join(lines), encoding="utf-8")
  title = work / f"{i}-title.txt"
  title.write_text(parts[i]["title"], encoding="utf-8")
  caption = work / f"{i}-caption.txt"
  caption.write_text(
      "Actual browser screenshots | Constructed group & CSV fixture\nEdited still-image demo | Phone NOT_RUN | No human-outcome claim",
      encoding="utf-8",
  )
  font = filter_path(Path("C:/Windows/Fonts/arial.ttf"))
  graph = f"drawbox=x=0:y=0:w=iw:h=72:color=0x101217:t=fill,drawtext=fontfile='{font}':textfile='{filter_path(title)}':fontcolor=0xb8df88:fontsize=28:x=24:y=20,drawbox=x=0:y=1160:w=iw:h=120:color=0x101217:t=fill,drawtext=fontfile='{font}':textfile='{filter_path(caption)}':fontcolor=0xf1f2f4:fontsize=20:line_spacing=8:x=24:y=1182,fps=24,setsar=1,format=yuv420p,tpad=stop_mode=clone:stop_duration={duration}"
  output = work / f"{i}.mp4"
  command = [
      ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-safe", "0",
      "-f", "concat", "-i", str(listing), "-i", str(root / f".runtime/demo-audio/{i}.wav"),
      "-vf", graph, "-t", str(duration), "-r", "24", "-fps_mode", "cfr",
      "-c:v", "libx264", "-preset", "fast", "-crf", "21", "-pix_fmt", "yuv420p",
      "-threads", "3", "-c:a", "aac", "-b:a", "96k", "-movflags", "+faststart",
      str(output),
  ]
  result = subprocess.run(command, capture_output=True, text=True)
  if result.returncode:
    raise RuntimeError("Chapter " + str(i) + ": " + result.stderr[-2000:])
  print("encoded chapter", i, round(duration, 2), flush=True)


with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
  list(pool.map(encode, range(len(parts))))
listing = work / "all.ffconcat"
listing.write_text(
    "ffconcat version 1.0\n"
    + "\n".join("file '" + quote(work / f"{i}.mp4") + "'" for i in range(len(parts))),
    encoding="utf-8",
)
destination = root / "dist/CommonReel-demo.mp4"
destination.parent.mkdir(exist_ok=True)
subprocess.run(
    [
        ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-safe", "0",
        "-f", "concat", "-i", str(listing), "-c", "copy", "-movflags", "+faststart",
        str(destination),
    ],
    check=True,
)
report = {
    "artifact": str(destination),
    "narration_seconds": round(sum(durations), 2),
    "date": "2026-10-17",
    "chapters": len(parts),
    "source_images": sum(len(chapter) for chapter in sources),
    "capture_manifest": "scripts/demo-captures.json",
    "source_sha256": {str(p.relative_to(root)).replace("\\", "/"): hashlib.sha256(p.read_bytes()).hexdigest() for chapter in sources for p in chapter},
    "method": "Actual October 15-17 browser screenshots listed per chapter; edited still-image holds; no generated product states",
    "narration": "Offline Microsoft Zira Desktop, original English project text, no third-party music",
    "phone": "NOT_RUN; narrow browser viewport only",
    "group": "constructed; CSV fixture; no human trial",
    "sha256": hashlib.sha256(destination.read_bytes()).hexdigest(),
    "bytes": destination.stat().st_size,
    "public_url": None,
}
(evidence / "demo-video-v2.json").write_text(
    json.dumps(report, indent=2), encoding="utf-8"
)
print(json.dumps(report), flush=True)
