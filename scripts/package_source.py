"""Package source, excluding private supplier data and local state.
Default keeps the original working-tree behavior. --ref exports a committed
Git tree for release delivery, so platform line-ending conversion cannot make
the package differ from the published source.
"""
import argparse,hashlib,io,json,subprocess,zipfile
from pathlib import Path
root=Path(__file__).resolve().parents[1]
git=['git','-c','safe.directory='+str(root).replace('\\','/')]
parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--ref',help='explicit committed revision for a release package')
args=parser.parse_args()
secrets=[]
envfile=root/'.env.local'
if envfile.exists():
 for line in envfile.read_text(encoding='utf-8-sig').splitlines():
  if line.strip().startswith('WATCHMODE_API_KEY='):
   value=line.split('=',1)[1].strip().strip('"\'')
   if value:secrets.append(value.encode())
if args.ref:
 raw=subprocess.check_output(git+['archive','--format=zip',args.ref],cwd=root)
 with zipfile.ZipFile(io.BytesIO(raw)) as z:
  entries=[(name,z.read(name)) for name in z.namelist() if not name.endswith('/')]
 commit=subprocess.check_output(git+['rev-parse',args.ref],cwd=root,text=True).strip()
else:
 paths=sorted(set(subprocess.check_output(git+['ls-files','--cached','--others','--exclude-standard','-z'],cwd=root).decode('utf-8').split('\0')))
 entries=[]
 for name in paths:
  if not name:continue
  path=(root/name).resolve()
  if not path.is_relative_to(root):raise RuntimeError('Unexpected package path: '+name)
  entries.append((name,path.read_bytes()))
 commit=None
for name,content in entries:
 path=(root/name).resolve()
 if not path.is_relative_to(root) or name.split('/')[0] in {'data','bin','.runtime','dist','.git'}:raise RuntimeError('Unexpected package path: '+name)
 if path.name in {'.env','.env.local'} or path.suffix.lower() in {'.db','.exe','.csv','.log','.zip'}:raise RuntimeError('Private/binary artifact blocked: '+name)
 if any(value in content for value in secrets):raise RuntimeError('Credential detected in source: '+name)
if args.ref:
 files=dict(entries)
 manifest=json.loads(files['assets/catalog-v2.manifest.json'])
 catalog=files['assets/catalog-v2.json']
 if hashlib.sha256(catalog).hexdigest()!=manifest['sha256'] or len(json.loads(catalog))!=manifest['catalog_count']:raise RuntimeError('Committed catalog does not match its required loader manifest')
 if hashlib.sha256(files['assets/genre-map.json']).hexdigest()!=manifest['genre_map_sha256']:raise RuntimeError('Committed genre mapping differs from its manifest')
output=root/'dist/CommonReel-source.zip';output.parent.mkdir(exist_ok=True)
with zipfile.ZipFile(output,'w',zipfile.ZIP_DEFLATED) as z:
 for name,content in entries:z.writestr('CommonReel/'+name,content)
print(json.dumps({'path':str(output),'files':len(entries),'bytes':output.stat().st_size,'commit':commit},indent=2))
