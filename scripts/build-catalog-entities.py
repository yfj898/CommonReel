"""Official Wikidata entity API fallback when SPARQL is rate limited.
Identifier joins must match IMDb exactly; titles are never trusted as identifiers.
"""
import csv, hashlib, io, json, pathlib, re, time, urllib.parse, urllib.request, zipfile, sys, subprocess
if "--modern-only" in sys.argv:
    subprocess.run([sys.executable,str(pathlib.Path(__file__).with_name("extend-catalog.py")),"--modern-only"],check=True)
    raise SystemExit(0)
from decimal import Decimal, ROUND_CEILING
root=pathlib.Path(__file__).resolve().parents[1];data=root/'data';cache=data/'wikidata-entities';cache.mkdir(exist_ok=True)
with zipfile.ZipFile(data/'ml-latest-small.zip') as z:
    links={r['movieId']:'tt'+r['imdbId'] for r in csv.DictReader(io.TextIOWrapper(z.open('ml-latest-small/links.csv'),encoding='utf-8'))}
    movies=list(csv.DictReader(io.TextIOWrapper(z.open('ml-latest-small/movies.csv'),encoding='utf-8')))[:650]
expected={links[r['movieId']] for r in movies}
titles=[]
for m in movies:
    title=re.sub(r' \(\d{4}\)$','',m['title'])
    title=re.sub(r'^(.*), (The|A|An)$',r'\2 \1',title)
    if title not in titles:titles.append(title)
films={};missing={'imdb_mismatch_or_missing':0,'runtime':0,'runtime_conflict':0,'genre':0}
genre_map=json.loads((root/'assets/genre-map.json').read_text(encoding='utf-8'))['genres']
genre_ids={qid:v['tags'] for qid,v in genre_map.items()}
selection_path=root/'assets/catalog-selection.json'
selection=set(json.loads(selection_path.read_text(encoding='utf-8'))) if selection_path.exists() else None
for start in range(0,len(titles),40):
    names=titles[start:start+40];path=cache/(str(start)+'.json')
    if not path.exists():
        url='https://www.wikidata.org/w/api.php?'+urllib.parse.urlencode({'action':'wbgetentities','sites':'enwiki','titles':'|'.join(names),'props':'claims|labels|aliases','languages':'en','format':'json','redirects':'yes'})
        try:
            with urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':'CommonReel/0.2 (noncommercial hackathon prototype)'}),timeout=40) as res:path.write_bytes(res.read())
        except Exception as exc:print('batch',start,type(exc).__name__,flush=True);continue
        time.sleep(1)
    entities=json.loads(path.read_text(encoding='utf-8')).get('entities',{})
    for qid,e in entities.items():
        claims=e.get('claims',{})
        def vals(prop):return [v['mainsnak']['datavalue']['value'] for v in claims.get(prop,[]) if v.get('rank')!='deprecated' and 'datavalue' in v.get('mainsnak',{})]
        imdbs=[v for v in vals('P345') if isinstance(v,str) and v in expected]
        if len(imdbs)!=1:missing['imdb_mismatch_or_missing']+=1;continue
        runtimes=set()
        for v in claims.get('P2047',[]):
            if v.get('rank')=='deprecated':continue
            d=v.get('mainsnak',{}).get('datavalue',{}).get('value',{})
            if isinstance(d,dict) and d.get('unit','').endswith('/Q7727'):
                try:runtimes.add(Decimal(d['amount']))
                except (KeyError,ValueError):pass
        genres=sorted({tag for v in vals('P136') if isinstance(v,dict) and v.get('id') in genre_ids for tag in (genre_ids[v['id']] if isinstance(genre_ids[v['id']],list) else [genre_ids[v['id']]])})
        if len(runtimes)!=1:missing['runtime_conflict' if runtimes else 'runtime']+=1;continue
        if not genres:missing['genre']+=1;continue
        years=[]
        for v in vals('P577'):
            try:years.append(int(v['time'][1:5]))
            except (KeyError,TypeError,ValueError):pass
        title=e.get('labels',{}).get('en',{}).get('value')
        if not title:continue
        f={'film_id':'wikidata:'+qid,'title':title,'year':min(years) if years else 0,'imdb_id':imdbs[0],'runtime_minutes':int(next(iter(runtimes)).to_integral_value(rounding=ROUND_CEILING)),'genres':genres,'aliases':[x['value'] for x in e.get('aliases',{}).get('en',[])],'catalog_prior':3.0,'source':'https://www.wikidata.org/wiki/'+qid}
        f['catalog_version']=hashlib.sha256(json.dumps(f,sort_keys=True).encode()).hexdigest();films[qid]=f
    print('batch',start,'verified films',len(films),flush=True)
    if selection is None and len(films)>=250:break
out=sorted([f for f in films.values() if selection is None or f['film_id'] in selection],key=lambda f:f['film_id'])[:250];payload=json.dumps(out,ensure_ascii=False,indent=2).encode();(data/'catalog-v2.json').write_bytes(payload)
manifest={'source':'Wikidata entity API, structured data CC0','identifier_seed':'GroupLens ml-latest-small IMDb identifiers; title requests are accepted only after exact IMDb cross-check','seed_count':len(movies),'catalog_count':len(out),'excluded_metadata':missing,'sha256':hashlib.sha256(payload).hexdigest(),'film_ids':[f['film_id'] for f in out],'created_utc':time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime()),'watchmode_data_included':False,'genre_mapping':genre_ids,'genre_map_sha256':hashlib.sha256((root/'assets/genre-map.json').read_bytes()).hexdigest(),'identifier_seed_sha256':hashlib.sha256((data/'ml-latest-small.zip').read_bytes()).hexdigest(),'genre_mapping_evidence':'Official English labels for all 168 observed genre IDs; composite source labels normalize into each explicit known tag; remaining source labels retained as their own tags; not a complete genre ontology','builder':'scripts/build-catalog-entities.py','runtime_units':'Wikidata P2047 unit Q7727 (minute); distinct source values conflict; fractional minutes round upward, never downward'}
(data/'catalog-v2.manifest.json').write_text(json.dumps(manifest,indent=2),encoding='utf-8');print('catalog_count',len(out),flush=True)


subprocess.run([sys.executable,str(root/"scripts/extend-catalog.py"),"--base-file",str(data/"catalog-v2.json")],check=True)
