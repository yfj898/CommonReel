"""Build the caption-led, audio-free demo from retained real browser captures.

The former narrated edit remains reproducible with build-demo.py. This edit
uses its own explicit cut/caption manifest and never reads the narration WAVs.
"""
import concurrent.futures
import json
import subprocess
import sys
import textwrap
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / '.runtime/video-deps'))
import imageio_ffmpeg

FFMPEG = imageio_ffmpeg.get_ffmpeg_exe()
FPS = 24
MANIFEST = ROOT / 'scripts/demo-silent-cuts.json'
EDIT = json.loads(MANIFEST.read_text(encoding='utf-8'))
SOURCE = ROOT / EDIT['source_root']
WORK = ROOT / '.runtime/demo-silent-encode-20261021'
WORK.mkdir(exist_ok=True)
DESTINATION = ROOT / EDIT['output']
DESTINATION.parent.mkdir(exist_ok=True)
FONT = Path('C:/Windows/Fonts/arial.ttf')
BOLD = Path('C:/Windows/Fonts/arialbd.ttf')


def quote(path):
    return str(path.resolve()).replace('\\', '/').replace("'", "'\\''")


def filter_path(path):
    return quote(path).replace(':', '\\:')


def text_filter(path, x, y, size, color='0x223624', bold=False, enable=None):
    value = (f"drawtext=fontfile='{filter_path(BOLD if bold else FONT)}':"
             f"textfile='{filter_path(path)}':fontcolor={color}:fontsize={size}:"
             f"line_spacing=10:x={x}:y={y}")
    if enable:
        value += f":enable='{enable}'"
    return value


def text_file(name, value):
    path = WORK / (name + '.txt')
    path.write_text(value, encoding='utf-8', newline='\n')
    return path


def normalized_still(path):
    # The source concat is MJPEG at 1280x900. Normalize the PNG preview and
    # portrait phone captures so decoder/geometry changes cannot reset filters.
    output = WORK / (path.parent.name + '-' + path.stem + '.jpg')
    subprocess.run([
        FFMPEG, '-hide_banner', '-loglevel', 'error', '-y', '-i', str(path),
        '-vf', 'scale=1280:900:force_original_aspect_ratio=decrease,pad=1280:900:(ow-iw)/2:(oh-ih)/2:color=0xf4f5ef',
        '-frames:v', '1', '-q:v', '2', str(output)
    ], check=True)
    return output


def shot_frames(shot):
    if 'file' in shot:
        return [(normalized_still(ROOT / shot['file']), shot['seconds'])]
    meta = json.loads((SOURCE / shot['clip'] / 'frames.json').read_text(encoding='utf-8'))
    if meta['errors']:
        raise RuntimeError('Capture has recorded errors: ' + shot['clip'])
    chosen = [(i, meta['frames'][i]) for i in shot['frames']]
    if shot.get('timing') == 'even':
        durations = [shot['seconds'] / len(chosen)] * len(chosen)
    else:
        durations = [max(1 / FPS, chosen[n + 1][1]['timestamp'] - f['timestamp'])
                     if n + 1 < len(chosen) else 1 / FPS
                     for n, (_, f) in enumerate(chosen)]
        for n, (index, _) in enumerate(chosen):
            durations[n] += shot.get('holds', {}).get(str(index), 0)
        remaining = shot['seconds'] - sum(durations)
        if remaining < 0:
            raise RuntimeError('Cut is shorter than its selected captured motion: ' + shot['clip'])
        durations[-1] += remaining
    result = []
    for (_, frame), duration in zip(chosen, durations):
        path = SOURCE / shot['clip'] / frame['file']
        if shot['clip'] == '03-mobile-final':
            path = normalized_still(path)
        result.append((path, duration))
    return result


def encode_scene(index):
    scene = EDIT['scenes'][index]
    seconds = sum(s['seconds'] for s in scene['shots'])
    # Captions must cover the scene continuously: the film has no narration.
    cursor = 0
    for caption in scene['captions']:
        if caption['start'] != cursor or caption['end'] <= cursor:
            raise RuntimeError('Caption gap/overlap in ' + scene['id'])
        cursor = caption['end']
    if cursor != seconds:
        raise RuntimeError('Captions do not cover ' + scene['id'])
    frames = [f for shot in scene['shots'] for f in shot_frames(shot)]
    concat = WORK / (scene['id'] + '.ffconcat')
    lines = ['ffconcat version 1.0']
    for path, dt in frames:
        lines.extend(["file '" + quote(path) + "'", 'duration ' + str(dt)])
    lines.append("file '" + quote(frames[-1][0]) + "'")
    concat.write_text('\n'.join(lines), encoding='utf-8')

    filters = [
        'scale=1280:900:force_original_aspect_ratio=decrease',
        'pad=1280:900:(ow-iw)/2:(oh-ih)/2:color=0xf4f5ef',
        'setsar=1',
        # Expand held source images before timed text, so captions can change
        # during a reading hold rather than freezing on the last source frame.
        'fps=24',
        'pad=1920:1080:32:76:color=0xf4f5ef',
        'drawbox=x=1340:y=76:w=548:h=900:color=0xe5ecdf:t=fill',
        'drawbox=x=0:y=984:w=1920:h=96:color=0x234b36:t=fill',
    ]
    for step in range(len(EDIT['scenes'])):
        color = '0x2f6b4b' if step <= index else '0xdce3d6'
        filters.append(f'drawbox=x={32 + step * 266}:y=66:w=250:h=4:color={color}:t=fill')
    texts = [
        ('brand', 'COMMONREEL', 32, 22, 28, '0x2f6b4b', True),
        ('track', 'ALEXA+ WEB SIMULATION', 1390, 28, 20, '0x52614e', False),
        ('step', scene['step'], 1368, 112, 20, '0x2f6b4b', True),
        ('title', scene['title'], 1368, 166, 38, '0x223624', True),
        ('metric', scene['metric'], 1368, 340, 72, '0x2f6b4b', True),
        ('metric-label', scene['metric_label'], 1368, 430, 26, '0x294d35', False),
        ('details', scene['details'], 1368, 520, 26, '0x294d35', False),
        ('note', scene.get('note', ''), 1368, 756, 22, '0x52614e', False),
        ('provenance', 'Edited real captures\nConstructed group', 1368, 892, 22, '0x52614e', False),
    ]
    for name, value, x, y, size, color, bold in texts:
        if value:
            path = text_file(scene['id'] + '-' + name, value)
            filters.append(text_filter(path, x, y, size, color, bold))
    for n, caption in enumerate(scene['captions']):
        path = text_file(scene['id'] + f'-caption-{n}', textwrap.fill(caption['text'], 90))
        enable = f"gte(t,{caption['start']})*lt(t,{caption['end']})"
        filters.append(text_filter(path, '(w-text_w)/2', '1032-text_h/2', 34,
                                   '0xf8fbf6', enable=enable))
    filters.append('format=yuv420p')
    output = WORK / (scene['id'] + '.mp4')
    subprocess.run([
        FFMPEG, '-hide_banner', '-loglevel', 'error', '-y', '-safe', '0',
        '-f', 'concat', '-i', str(concat), '-vf', ','.join(filters),
        '-t', str(seconds), '-an', '-c:v', 'libx264', '-preset', 'fast',
        '-crf', '20', '-threads', '3', '-movflags', '+faststart', str(output)
    ], check=True)
    print('Encoded', scene['id'], seconds, 'seconds; no audio', flush=True)
    return output


def stamp(seconds):
    ms = round(seconds * 1000)
    return f'{ms // 3600000:02}:{ms // 60000 % 60:02}:{ms // 1000 % 60:02},{ms % 1000:03}'


def main():
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
        files = list(pool.map(encode_scene, range(len(EDIT['scenes']))))
    concat = WORK / 'all.ffconcat'
    concat.write_text('ffconcat version 1.0\n' + '\n'.join("file '" + quote(p) + "'" for p in files), encoding='utf-8')
    subprocess.run([FFMPEG, '-hide_banner', '-loglevel', 'error', '-y', '-safe', '0',
                    '-f', 'concat', '-i', str(concat), '-map', '0:v:0', '-an',
                    '-c:v', 'copy', '-movflags', '+faststart', str(DESTINATION)], check=True)
    srt, transcript, timeline = [], [], []
    offset = 0
    for scene in EDIT['scenes']:
        duration = sum(s['seconds'] for s in scene['shots'])
        timeline.append({'id': scene['id'], 'start': offset, 'seconds': duration})
        for c in scene['captions']:
            start, end = offset + c['start'], offset + c['end']
            srt.append(f"{len(srt) + 1}\n{stamp(start)} --> {stamp(end)}\n{c['text']}\n")
            transcript.append(f"{stamp(start)}  {c['text']}")
        offset += duration
    captions = DESTINATION.parent / 'CommonReel-demo-silent.en.srt'
    captions.write_text('\n'.join(srt), encoding='utf-8')
    (DESTINATION.parent / 'CommonReel-demo-silent-captions.txt').write_text('\n'.join(transcript) + '\n', encoding='utf-8')
    report = {
        'artifact': EDIT['output'], 'date': EDIT['date'], 'duration_seconds': offset,
        'size': [1920, 1080], 'fps': FPS, 'audio_tracks': 0,
        'captions': 'Short English captions burned into every scene; matching SRT supplied',
        'caption_cues': len(srt), 'caption_file': 'dist/' + captions.name,
        'cut_manifest': 'scripts/demo-silent-cuts.json', 'timeline': timeline,
        'method': EDIT['method'],
        'selected_frame_entries': sum(len(s.get('frames', [])) for p in EDIT['scenes'] for s in p['shots']),
        'unique_source_frames': len({(s['clip'], f) for p in EDIT['scenes'] for s in p['shots'] if 'clip' in s for f in s['frames']}),
        'terminal_capture': 'docs/evidence/four-improvements-demo-preview-19.png',
        'preview_observed': {'before_minutes': 81, 'after_minutes': 96, 'before_eligible': 0, 'after_eligible': 19, 'newly_eligible': 19, 'shown': 6},
        'saved_film': 'Inside Out 2', 'members': 'four constructed entries; proxy recording allowed',
        'mobile': 'Chromium emulation; physical phones NOT_RUN', 'human_trials': 'NOT_RUN',
        'previous_narrated_edit': 'dist/CommonReel-demo-2026-10-21.mp4',
        'bytes': DESTINATION.stat().st_size, 'public_url': None, 'upload_owner': 'User will upload to YouTube'
    }
    (ROOT / 'docs/evidence/demo-video-silent-2026-10-21.json').write_text(json.dumps(report, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(report), flush=True)


if __name__ == '__main__':
    main()
