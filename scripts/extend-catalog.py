"""Merge 30 reviewed recent films without rewriting the retained 217 film objects.
Outputs go to data/; copy the two catalog files to assets/ after verification.
"""
import argparse,hashlib,importlib.util,json,time
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--base-file',type=Path,default=ROOT/'assets/catalog-v2.json')
parser.add_argument('--modern-only',action='store_true')
args=parser.parse_args()
spec=importlib.util.spec_from_file_location('modern_catalog',ROOT/'scripts/build-modern-catalog.py')
modern=importlib.util.module_from_spec(spec);spec.loader.exec_module(modern)
source=ROOT/'assets/catalog-sources/modern-2026-10-21'
data=ROOT/'data';data.mkdir(exist_ok=True)
modern.build(source,data/'modern')
def read(p):return json.loads(p.read_text(encoding='utf-8'))
base_ids=set(read(ROOT/'assets/catalog-base-selection.json'))
base=[f for f in read(args.base_file) if f['film_id'] in base_ids]
assert len(base)==217 and {f['film_id'] for f in base}==base_ids
addition=read(data/'modern/modern-films.json')
films=sorted(base+addition,key=lambda f:f['film_id'])
assert len(films)==247 and len({f['film_id'] for f in films})==247 and len({f['imdb_id'] for f in films})==247
mapping=read(ROOT/'assets/genre-map.json');mapping['genres'].update(read(source/'modern-genre-map.json')['genres'])
(ROOT/'assets/genre-map.json').write_text(json.dumps(mapping,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
(ROOT/'assets/catalog-selection.json').write_text(json.dumps([f['film_id'] for f in films],indent=2)+'\n',encoding='utf-8')
payload=json.dumps(films,ensure_ascii=False,indent=2).encode()
(data/'catalog-v2.json').write_bytes(payload)
manifest=read(ROOT/'assets/catalog-v2.manifest.json')
manifest.update(catalog_count=247,film_ids=[f['film_id'] for f in films],sha256=hashlib.sha256(payload).hexdigest(),created_utc=time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime()),genre_mapping={qid:v['tags'] for qid,v in mapping['genres'].items()},genre_map_sha256=hashlib.sha256((ROOT/'assets/genre-map.json').read_bytes()).hexdigest())
manifest.update(seed_count=680,legacy_seed_count=650,identifier_seed='650 legacy GroupLens IMDb identifiers plus 30 fixed recent Wikidata/IMDb tuples',excluded_metadata_scope='legacy 650-seed audit; modern candidate decisions are retained separately',genre_mapping_evidence='Legacy 168 source labels plus two reviewed recent genre QIDs; official English labels retained in source snapshots')
manifest['modern_addition']={'seed_count':30,'count':30,'source':'assets/catalog-sources/modern-2026-10-21','years':[2015,2024],'builder':'scripts/build-catalog-entities.py --modern-only','base_identifiers_preserved':217,'supply_included':False}
(data/'catalog-v2.manifest.json').write_text(json.dumps(manifest,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'catalog_count':len(films),'retained_count':len(base),'added_count':len(addition),'supply_calls':0}))
