"""Rebuild the approved 30-film metadata addition from the retained official responses.

This offline command uses no API key or streaming-supply service. catalog_version
is the existing CommonReel per-film version contract, not a transfer checksum.
"""
import argparse
from decimal import Decimal, ROUND_CEILING
import hashlib
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1] / 'assets/catalog-sources/modern-2026-10-21'

def read(path):
    return json.loads(path.read_text(encoding='utf-8'))

def vals(entity, prop):
    return [c['mainsnak']['datavalue']['value']
            for c in entity.get('claims', {}).get(prop, [])
            if c.get('rank') != 'deprecated'
            and 'datavalue' in c.get('mainsnak', {})]

def facts(entity):
    runtimes, invalid_units = set(), []
    for value in vals(entity, 'P2047'):
        if not isinstance(value, dict) or not value.get('unit', '').endswith('/Q7727'):
            invalid_units.append(value)
            continue
        amount = Decimal(value['amount'])
        if not amount.is_finite() or amount <= 0:
            raise ValueError(f"{entity['id']}: invalid duration {value!r}")
        runtimes.add(amount)
    years = sorted({int(v['time'][1:5]) for v in vals(entity, 'P577')
                    if isinstance(v, dict) and re.match(r'^\+\d{4}-', v.get('time', ''))})
    return dict(title=entity.get('labels', {}).get('en', {}).get('value'),
                imdb_ids=sorted(set(vals(entity, 'P345'))),
                runtimes=sorted(runtimes), invalid_runtime_units=invalid_units,
                release_years=years,
                genre_ids=sorted({v['id'] for v in vals(entity, 'P136') if isinstance(v, dict)}),
                instance_ids=sorted({v['id'] for v in vals(entity, 'P31') if isinstance(v, dict)}))

def build(root, out):
    seeds = read(root / 'modern-seeds.json')
    base = read(root / 'base-catalog-identifiers.json')
    genre_map = read(root / 'base-genre-map.json')['genres']
    genre_map.update(read(root / 'modern-genre-map.json')['genres'])
    sources = read(root / 'raw-sources.json')
    entities, source_files, labels = {}, {}, {}
    for source in sources:
        payload = read(root / source['file'])
        for qid, entity in payload.get('entities', {}).items():
            if Path(source['file']).name.startswith('genre-instance-labels-'):
                labels[qid] = entity
            elif qid.startswith('Q'):
                entities[qid] = entity
                source_files[qid] = source['file']
    old_ids = {f['film_id'] for f in base}
    old_imdb = {f['imdb_id'] for f in base}
    if len(base) != 217 or len(old_ids) != 217 or len(old_imdb) != 217:
        raise ValueError('The retained base snapshot must have 217 unique film and IMDb identifiers')
    if len(seeds) != 30 or len({s['qid'] for s in seeds}) != 30:
        raise ValueError('The approved addition must contain exactly 30 unique QIDs')
    films, audits = [], []
    for seed in seeds:
        qid = seed['qid']
        entity = entities[qid]
        f = facts(entity)
        if f['imdb_ids'] != [seed['imdb_id']] or not re.fullmatch(r'tt\d{7,9}', seed['imdb_id']):
            raise ValueError(f'{qid}: IMDb claim differs from fixed identifier seed')
        if f['title'] != seed['title']:
            raise ValueError(f'{qid}: English title differs from fixed seed')
        if not f['release_years'] or min(f['release_years']) != seed['year'] or not 2015 <= seed['year'] <= 2025:
            raise ValueError(f'{qid}: earliest source release year differs or is outside 2015–2025')
        if len(f['runtimes']) != 1 or f['invalid_runtime_units']:
            raise ValueError(f'{qid}: runtime is absent, conflicting, or not in minutes')
        if not ({'Q11424', 'Q202866'} & set(f['instance_ids'])):
            raise ValueError(f'{qid}: no explicit film/animated-film instance claim')
        if not f['genre_ids'] or any(g not in genre_map for g in f['genre_ids']):
            raise ValueError(f'{qid}: one or more source genres lack a reviewed mapping')
        if any(genre_map[g]['label'] != labels[g]['labels']['en']['value'] for g in f['genre_ids']):
            raise ValueError(f'{qid}: a mapping label differs from its retained official source')
        if 'wikidata:' + qid in old_ids or seed['imdb_id'] in old_imdb:
            raise ValueError(f'{qid}: duplicates a retained base identifier')
        genres = sorted({tag for g in f['genre_ids'] for tag in genre_map[g]['tags']})
        film = dict(film_id='wikidata:' + qid, title=f['title'], year=seed['year'],
                    imdb_id=seed['imdb_id'],
                    runtime_minutes=int(f['runtimes'][0].to_integral_value(rounding=ROUND_CEILING)),
                    genres=genres,
                    aliases=[v['value'] for v in entity.get('aliases', {}).get('en', [])],
                    catalog_prior=3.0, source='https://www.wikidata.org/wiki/' + qid)
        film['catalog_version'] = hashlib.sha256(json.dumps(film, sort_keys=True).encode()).hexdigest()
        films.append(film)
        audits.append(dict(qid=qid, film_id=film['film_id'], raw_file=source_files[qid],
                           source=film['source'], **{**f, 'runtimes': [str(v) for v in f['runtimes']]},
                           genre_evidence=[dict(qid=g, label=labels[g]['labels']['en']['value'],
                                                tags=genre_map[g]['tags'], source='https://www.wikidata.org/wiki/'+g)
                                           for g in f['genre_ids']],
                           output_runtime_minutes=film['runtime_minutes']))
    if len({f['imdb_id'] for f in films}) != 30:
        raise ValueError('The addition must contain 30 unique IMDb identifiers')
    out.mkdir(parents=True, exist_ok=True)
    (out / 'modern-films.json').write_text(json.dumps(sorted(films, key=lambda f:f['film_id']), ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    (out / 'source-audit.json').write_text(json.dumps(audits, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(dict(base_count=len(base), addition_count=len(films), total_count=len(base)+len(films),
                          year_range=[min(f['year'] for f in films), max(f['year'] for f in films)],
                          genres=sorted({g for f in films for g in f['genres']}),
                          supply_calls=0, model_calls=0), ensure_ascii=False))

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source-dir', type=Path, default=ROOT)
    parser.add_argument('--out-dir', type=Path, default=ROOT)
    args = parser.parse_args()
    build(args.source_dir, args.out_dir)
