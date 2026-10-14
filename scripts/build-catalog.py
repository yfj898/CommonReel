"""Build a CC0 catalog from a bounded, IMDb-seeded Wikidata query.
MovieLens links seed identifiers only; output labels/runtime/genre come from Wikidata.
Run explicitly; no supplier API credits or images. Respect endpoint backoff.
"""
import csv, hashlib, io, json, pathlib, re, time, urllib.parse, urllib.request, zipfile

root = pathlib.Path(__file__).resolve().parents[1]
data = root / 'data'
data.mkdir(exist_ok=True)
with zipfile.ZipFile(data / 'ml-latest-small.zip') as z:
    links = list(csv.DictReader(io.TextIOWrapper(z.open('ml-latest-small/links.csv'))))
ids = ['tt' + x['imdbId'] for x in links[:400]]
query = '''SELECT ?film ?imdb ?title ?duration ?genre ?genreLabel ?date WHERE {
VALUES ?imdb { %s }
?film wdt:P345 ?imdb; rdfs:label ?title . FILTER(LANG(?title)="en")
OPTIONAL { ?film wdt:P2047 ?duration }
OPTIONAL { ?film wdt:P136 ?genre . ?genre rdfs:label ?genreLabel . FILTER(LANG(?genreLabel)="en") }
OPTIONAL { ?film wdt:P577 ?date }
}''' % ' '.join('"' + x + '"' for x in ids)
url = 'https://query.wikidata.org/sparql?' + urllib.parse.urlencode({'query': query, 'format': 'json'})
rawpath = data / 'wikidata-bounded-source.json'
if not rawpath.exists():
    for attempt in range(3):
        try:
            req = urllib.request.Request(url, headers={'User-Agent': 'CommonReel/0.2 (noncommercial movie prototype)', 'Accept': 'application/sparql-results+json'})
            with urllib.request.urlopen(req, timeout=55) as res:
                rawpath.write_bytes(res.read())
            break
        except Exception as exc:
            print('Wikidata attempt', attempt + 1, type(exc).__name__, str(exc)[:100], flush=True)
            if attempt == 2:
                raise
            time.sleep(61)
raw = json.loads(rawpath.read_text(encoding='utf-8'))
films = {}
for r in raw['results']['bindings']:
    qid = r['film']['value'].rsplit('/', 1)[-1]
    f = films.setdefault(qid, {'film_id': 'wikidata:' + qid, 'title': r['title']['value'], 'imdb_id': r['imdb']['value'], 'runtime_minutes': 0, 'genres': [], 'year': 0, 'catalog_prior': 3.0, 'source': 'https://www.wikidata.org/wiki/' + qid, '_durations': set()})
    if 'duration' in r:
        f['_durations'].add(round(float(r['duration']['value'])))
    if 'date' in r:
        y = int(r['date']['value'][:4])
        if not f['year'] or y < f['year']: f['year'] = y
    if 'genreLabel' in r:
        label = r['genreLabel']['value'].lower()
        for needle, genre in [('comedy','Comedy'),('drama','Drama'),('horror','Horror'),('thriller','Thriller'),('science fiction','Sci-Fi'),('fantasy','Fantasy'),('romance','Romance'),('romantic','Romance'),('action','Action'),('adventure','Adventure'),('animation','Animation'),('animated','Animation'),('documentary','Documentary'),('crime','Crime'),('mystery','Mystery'),('war','War'),('western','Western'),('musical','Musical'),('family','Family')]:
            if needle in label and genre not in f['genres']: f['genres'].append(genre)
out = []
missing = {'runtime': 0, 'runtime_conflict': 0, 'genre': 0}
for f in sorted(films.values(), key=lambda x: x['film_id']):
    durations = f.pop('_durations')
    f['runtime_conflict'] = len(durations) > 1
    f['runtime_minutes'] = next(iter(durations)) if len(durations) == 1 else 0
    f['genres'].sort()
    for k, bad in [('runtime', not durations), ('runtime_conflict', len(durations)>1), ('genre', not f['genres'])]: missing[k] += int(bad)
    if f['runtime_minutes'] > 0 and f['genres'] and not f['runtime_conflict'] and len(out) < 250:
        f['catalog_version'] = hashlib.sha256(json.dumps(f, sort_keys=True).encode()).hexdigest()
        out.append(f)
payload = json.dumps(out, ensure_ascii=False, indent=2).encode()
(data/'catalog-v2.json').write_bytes(payload)
manifest = {'source': 'Wikidata structured data CC0', 'identifier_seed': 'GroupLens ml-latest-small links.csv; IMDb identifiers only', 'seed_count': len(ids), 'mapped_count': len(films), 'catalog_count': len(out), 'excluded_metadata': missing, 'sha256': hashlib.sha256(payload).hexdigest(), 'source_sha256': hashlib.sha256(rawpath.read_bytes()).hexdigest(), 'film_ids': [f['film_id'] for f in out], 'query': query, 'created_utc': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()), 'watchmode_data_included': False}
(data/'catalog-v2.manifest.json').write_text(json.dumps(manifest, indent=2), encoding='utf-8')
print(json.dumps({k:v for k,v in manifest.items() if k not in ('film_ids','query')}), flush=True)
