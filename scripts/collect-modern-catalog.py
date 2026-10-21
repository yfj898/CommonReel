"""Read-only official Wikidata candidate collection. Never calls supply services."""
import concurrent.futures
import datetime
import json
from pathlib import Path
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[1] / 'assets/catalog-sources/modern-2026-10-21'
RAW = ROOT / 'raw'
RAW.mkdir(exist_ok=True)
TITLES = [
    'Inside Out (2015 film)', 'The Martian (film)', 'Zootopia', 'Arrival (film)',
    'La La Land', 'Moana (2016 film)', 'Hidden Figures', 'Coco (2017 film)',
    'Get Out', 'Dunkirk (2017 film)', 'The Shape of Water', 'Paddington 2',
    'Spider-Man: Into the Spider-Verse', 'Black Panther (film)', 'Knives Out',
    'Parasite (2019 film)', 'Little Women (2019 film)', 'Ford v Ferrari',
    'Soul (2020 film)', 'Palm Springs (2020 film)', 'The Mitchells vs. the Machines',
    'Dune (2021 film)', 'CODA (2021 film)', 'Encanto',
    'Everything Everywhere All at Once', 'Top Gun: Maverick',
    'Puss in Boots: The Last Wish', 'Past Lives (film)', 'The Holdovers',
    'Inside Out 2', 'Dune: Part Two', 'The Wild Robot', 'Flow (2024 film)',
    'The Grand Budapest Hotel', 'The Big Short (film)', 'Spotlight (film)',
    'Room (2015 film)', 'Brooklyn (film)', 'The Nice Guys',
    'Hunt for the Wilderpeople', 'Sing Street', 'Captain Fantastic (film)',
    'The Florida Project', 'Lady Bird (film)', 'Logan (film)',
    'Baby Driver', 'The Farewell (2019 film)', 'Jojo Rabbit',
    'Klaus (film)', 'Sound of Metal', 'Wolfwalkers',
    'Minari (film)', 'Nomadland (film)', 'Luca (2021 film)',
    'The Worst Person in the World (film)', 'The Banshees of Inisherin',
    'Marcel the Shell with Shoes On (2021 film)', 'Turning Red',
    'Guillermo del Toro\'s Pinocchio', 'Aftersun',
    'The Boy and the Heron', 'American Fiction (film)',
    'Robot Dreams (film)', 'Nimona (film)', 'Anatomy of a Fall',
    'Godzilla Minus One', 'Thelma (2024 film)', 'A Real Pain',
    'Wallace & Gromit: Vengeance Most Fowl', 'Memoir of a Snail',
    'Sing Sing (2023 film)', 'The Life of Chuck',
]

def download(index, names):
    path = RAW / f'candidate-batch-{index:02}.json'
    args = dict(action='wbgetentities', sites='enwiki', titles='|'.join(names),
                props='claims|labels|aliases|descriptions', languages='en',
                format='json', redirects='yes')
    url = 'https://www.wikidata.org/w/api.php?' + urllib.parse.urlencode(args)
    if not path.exists():
        req = urllib.request.Request(url, headers={'User-Agent': 'CommonReel/0.3 (noncommercial hackathon; source verification)'})
        with urllib.request.urlopen(req, timeout=90) as res:
            payload = res.read()
        parsed = json.loads(payload)
        if 'error' in parsed or not isinstance(parsed.get('entities'), dict):
            raise ValueError(parsed)
        path.write_bytes(payload)
    data = json.loads(path.read_text(encoding='utf-8'))
    return dict(index=index, names=names, file=path.relative_to(ROOT).as_posix(), source=url,
                entities=len(data['entities']), acquired_utc=datetime.datetime.fromtimestamp(path.stat().st_mtime, datetime.timezone.utc).isoformat())

if __name__ == '__main__':
    batches = [(i // 15, TITLES[i:i+15]) for i in range(0, len(TITLES), 15)]
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
        futures = [pool.submit(download, i, names) for i, names in batches]
        results = []
        for future in concurrent.futures.as_completed(futures):
            row = future.result()
            print(row['index'], row['entities'], flush=True)
            results.append(row)
    # This title's enwiki redirect does not resolve through wbgetentities's sites request.
    path = RAW / 'inside-out-direct.json'
    url = 'https://www.wikidata.org/w/api.php?action=wbgetentities&ids=Q6144664&props=claims%7Clabels%7Caliases%7Cdescriptions&languages=en&format=json'
    if not path.exists():
        with urllib.request.urlopen(urllib.request.Request(url, headers={'User-Agent':'CommonReel/0.3'}), timeout=90) as res:
            payload = res.read()
        parsed = json.loads(payload)
        if 'error' in parsed or 'Q6144664' not in parsed.get('entities', {}):
            raise ValueError(parsed)
        path.write_bytes(payload)
    results.append(dict(file=path.relative_to(ROOT).as_posix(), source=url, ids=['Q6144664'],
                        acquired_utc=datetime.datetime.fromtimestamp(path.stat().st_mtime, datetime.timezone.utc).isoformat()))
    entities = {}
    for source in results:
        entities.update(json.loads((ROOT / source['file']).read_text(encoding='utf-8'))['entities'])
    genre_instance_ids = set()
    for qid in json.loads((ROOT / 'selection-qids.json').read_text(encoding='utf-8')):
        for prop in ['P136', 'P31']:
            for claim in entities[qid]['claims'].get(prop, []):
                value = claim.get('mainsnak', {}).get('datavalue', {}).get('value')
                if claim.get('rank') != 'deprecated' and isinstance(value, dict) and 'id' in value:
                    genre_instance_ids.add(value['id'])
    genre_instance_ids = sorted(genre_instance_ids)
    for start in range(0, len(genre_instance_ids), 40):
        ids = genre_instance_ids[start:start+40]
        path = RAW / f'genre-instance-labels-{start//40}.json'
        args = dict(action='wbgetentities', ids='|'.join(ids), props='labels|descriptions',
                    languages='en|mul', format='json')
        url = 'https://www.wikidata.org/w/api.php?' + urllib.parse.urlencode(args)
        if not path.exists():
            with urllib.request.urlopen(urllib.request.Request(url, headers={'User-Agent':'CommonReel/0.3'}), timeout=90) as res:
                payload = res.read()
            parsed = json.loads(payload)
            if 'error' in parsed or not isinstance(parsed.get('entities'), dict):
                raise ValueError(parsed)
            path.write_bytes(payload)
        results.append(dict(file=path.relative_to(ROOT).as_posix(), source=url, ids=ids,
                            acquired_utc=datetime.datetime.fromtimestamp(path.stat().st_mtime, datetime.timezone.utc).isoformat()))
    (ROOT / 'raw-sources.json').write_text(json.dumps(sorted(results, key=lambda x:x['file']), ensure_ascii=False, indent=2), encoding='utf-8')
    print('complete', flush=True)
