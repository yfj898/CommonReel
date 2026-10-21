"""Encode the current real-interaction demo with existing offline SAPI/FFmpeg tools.
All selected browser screencast frames and their timestamps are in docs/evidence.
Cuts and terminal-frame reading holds align narration; timings are not a benchmark.
"""
import concurrent.futures,json,subprocess,sys,textwrap,wave
from pathlib import Path
root=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(root/'.runtime/video-deps'))
import imageio_ffmpeg
ffmpeg=imageio_ffmpeg.get_ffmpeg_exe()
work=root/'.runtime/demo-encode-20261021';work.mkdir(exist_ok=True)
parts=json.loads((root/'scripts/demo-narration.json').read_text(encoding='utf-8'))
manifest=json.loads((root/'scripts/demo-captures.json').read_text(encoding='utf-8'))
source=root/manifest['source_root'];destination=root/'dist/CommonReel-demo-2026-10-21.mp4';destination.parent.mkdir(exist_ok=True)
def quote(p):return str(p.resolve()).replace("\\","/").replace("'","'\\''")
def filter_path(p):return quote(p).replace(':','\\:')
font=filter_path(Path('C:/Windows/Fonts/arial.ttf'))
bold=filter_path(Path('C:/Windows/Fonts/arialbd.ttf'))
voiced=[]
for i,p in enumerate(parts):
 with wave.open(str(root/f'.runtime/demo-audio/{i}.wav')) as w:d=w.getnframes()/w.getframerate()
 if d>p['seconds']:raise RuntimeError(f'Chapter {i} narration {d:.2f}s exceeds {p["seconds"]}s')
 voiced.append(d)

def stream_clip(name,seconds,label):
 meta=json.loads((source/name/'frames.json').read_text(encoding='utf-8'))
 if meta['errors'] or len(meta['frames'])<2:raise RuntimeError('Incomplete capture '+name)
 frames=meta['frames'];listing=work/(name+'.ffconcat');lines=['ffconcat version 1.0']
 for i,f in enumerate(frames):
  dt=max(.04,frames[i+1]['timestamp']-f['timestamp']) if i+1<len(frames) else .12
  if i==0:dt+=1.5
  lines += ["file '"+quote(source/name/f['file'])+"'",'duration '+str(dt)]
 if name=='02-readonly-agent':
  terminal=work/'02-terminal.jpg'
  subprocess.run([ffmpeg,'-hide_banner','-loglevel','error','-y','-i',str(root/'docs/evidence/four-improvements-demo-preview-19.png'),'-vf','scale=1280:900:force_original_aspect_ratio=decrease,pad=1280:900:(ow-iw)/2:(oh-ih)/2:color=0xf4f5ef','-frames:v','1','-q:v','2',str(terminal)],check=True)
  lines += ["file '"+quote(terminal)+"'",'duration 0.12',"file '"+quote(terminal)+"'"]
 else:lines += ["file '"+quote(source/name/frames[-1]['file'])+"'"]
 listing.write_text('\n'.join(lines),encoding='utf-8')
 out=work/(name+'.mp4')
 graph=f'scale=1280:900:force_original_aspect_ratio=decrease,pad=1280:900:(ow-iw)/2:(oh-ih)/2:color=0xf4f5ef,setsar=1,fps=24,tpad=stop_mode=clone:stop_duration={seconds},format=yuv420p'
 subprocess.run([ffmpeg,'-hide_banner','-loglevel','error','-y','-safe','0','-f','concat','-i',str(listing),'-vf',graph,'-t',str(seconds),'-an','-c:v','libx264','-preset','fast','-crf','20','-threads','3',str(out)],check=True)
 return out

def encode(i):
 p=parts[i];duration=p['seconds']
 if p.get('phone_clip'):
  segments=[(p['clips'][0],8,'Loopback client'),(p['phone_clip'],duration-8,'LAN client')]
 else:segments=[(p['clips'][0],duration,'Actual browser capture')]
 media=[stream_clip(name,seconds,label) for name,seconds,label in segments]
 listing=work/f'{i}-visuals.ffconcat';listing.write_text('ffconcat version 1.0\n'+'\n'.join("file '"+quote(v)+"'" for v in media),encoding='utf-8')
 title=work/f'{i}-title.txt';title.write_text(p['title'],encoding='utf-8')
 note=work/f'{i}-note.txt';note.write_text(p['note'],encoding='utf-8')
 footer=work/f'{i}-footer.txt';footer.write_text('Actual browser interaction | Edited with reading holds | Constructed members',encoding='utf-8')
 graph=f"pad=1920:1080:32:84:color=0xf4f5ef,drawbox=x=1340:y=84:w=548:h=900:color=0xe5ecdf:t=fill,drawtext=fontfile='{bold}':text='COMMONREEL':fontcolor=0x2f6b4b:fontsize=27:x=32:y=25,drawtext=fontfile='{font}':text='ALEXA+ WEB SIMULATION':fontcolor=0x52614e:fontsize=20:x=1390:y=34,drawtext=fontfile='{bold}':textfile='{filter_path(title)}':fontcolor=0x223624:fontsize=27:x=1368:y=160,drawtext=fontfile='{font}':textfile='{filter_path(note)}':fontcolor=0x294d35:fontsize=27:line_spacing=22:x=1368:y=260,drawtext=fontfile='{font}':textfile='{filter_path(footer)}':fontcolor=0x52614e:fontsize=22:x=32:y=1020,format=yuv420p"
 out=work/f'{i}.mp4'
 subprocess.run([ffmpeg,'-hide_banner','-loglevel','error','-y','-safe','0','-f','concat','-i',str(listing),'-i',str(root/f'.runtime/demo-audio/{i}.wav'),'-vf',graph,'-af','apad','-t',str(duration),'-r','24','-c:v','libx264','-preset','fast','-crf','20','-threads','3','-c:a','aac','-b:a','128k','-movflags','+faststart',str(out)],check=True)
 print('Encoded chapter',i,duration,flush=True)
 return out
with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:files=list(pool.map(encode,range(len(parts))))
listing=work/'all.ffconcat';listing.write_text('ffconcat version 1.0\n'+'\n'.join("file '"+quote(f)+"'" for f in files),encoding='utf-8')
subprocess.run([ffmpeg,'-hide_banner','-loglevel','error','-y','-safe','0','-f','concat','-i',str(listing),'-c','copy','-movflags','+faststart',str(destination)],check=True)
# Approximate sentence-level subtitle timing follows the actual offline narration.
def stamp(t):
 ms=round(t*1000);return f'{ms//3600000:02}:{ms//60000%60:02}:{ms//1000%60:02},{ms%1000:03}'
srt=[];offset=0
for i,p in enumerate(parts):
 words=p['text'].split();chunks=[words[j:j+13] for j in range(0,len(words),13)];done=0
 for chunk in chunks:
  start=offset+voiced[i]*done/len(words);done+=len(chunk);end=offset+voiced[i]*done/len(words)
  srt.append(f'{len(srt)+1}\n{stamp(start)} --> {stamp(end)}\n'+textwrap.fill(' '.join(chunk),width=64)+'\n')
 offset+=p['seconds']
(destination.parent/'CommonReel-demo.en.srt').write_text('\n'.join(srt),encoding='utf-8')
(destination.parent/'CommonReel-demo-narration.txt').write_text('\n\n'.join(p['title']+'\n'+p['text'] for p in parts),encoding='utf-8')
report={'artifact':'dist/'+destination.name,'date':'2026-10-21','duration_seconds':sum(p['seconds'] for p in parts),'size':[1920,1080],'fps':24,'chapters':len(parts),'source_frames':sum(len(json.loads((source/n/'frames.json').read_text())['frames']) for n in {n for p in parts for n in p['clips']+([p['phone_clip']] if p.get('phone_clip') else [])}),'terminal_captures':['docs/evidence/four-improvements-demo-preview-19.png'],'capture_manifest':'scripts/demo-captures.json','method':manifest['method'],'narration':'Offline Microsoft Zira Desktop; original English project text; no third-party music or footage','preview_observed':{'before_minutes':81,'after_minutes':96,'before_eligible':0,'after_eligible':19,'newly_eligible':19,'shown':6,'unknown_under_proposed_limit':0},'saved_film':'Inside Out 2','members':'four constructed entries; proxy recording allowed','mobile':'Chromium emulation, not a physical phone','human_trials':'NOT_RUN','public_url':None,'upload_owner':'User will upload to YouTube','bytes':destination.stat().st_size}
(root/'docs/evidence/demo-video-v2.json').write_text(json.dumps(report,indent=2)+'\n',encoding='utf-8')
print(json.dumps(report),flush=True)
