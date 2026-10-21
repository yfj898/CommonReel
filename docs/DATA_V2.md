# Current data boundaries — 2026-10-21

The v2 metadata catalog contains **247 films**, independent of the MovieLens
user-kNN research experiment. The first 217 film objects/IDs are retained;
30 additional films from 2015–2024 meet the approved 2015–2025 boundary.

| Source | Use | Distribution / limitations |
|---|---|---|
| Wikidata structured entities | Exact QID/IMDb IDs, English titles/aliases, unique positive minute-valued P2047 runtime, explicit P136 genre tags | CC0 records and reviewed source responses included; no posters, trailers or encyclopedia prose |
| GroupLens ml-latest-small links | 650 legacy identifier seeds for exact joins | Legacy audit retained; raw ratings, research users and ZIP not distributed |
| Fixed modern seeds | 30 exact QID/IMDb/title/year tuples, checked against retained official responses | Included under assets/catalog-sources/modern-2026-10-21 |
| Watchmode Free API | US selected subscription listings | Own free key, bounded requests, private local cache; no account entitlement claim or paid upgrade |
| Optional Netflix CSV | Explicit known watched / pending titles | Local normalization only; watched is not liked; raw CSV is not retained |
| Ollama + qwen3.5:4b | Free loopback tool planning | Weights not bundled; no cloud fallback or history upload |

The combined manifest identifies 680 fixed input seeds: 650 legacy and 30 modern.
Legacy exclusion counters refer only to the original 650-seed audit; the modern
candidate audit separately records selection and conflicts. Exact film IDs,
versions and the loader-required content checksum bind the final catalog.

Rebuild the recent addition offline with
`python scripts/build-catalog-entities.py --modern-only`. It preserves the
retained base objects, generates `data/catalog-v2.json` and its manifest, and
synchronizes the reviewed selection/genre files. Review and copy both generated
catalog files to `assets/`. No API key or new download is needed. The full legacy
build remains available with its original local MovieLens ZIP requirement.
[Modern provenance](MODERN_CATALOG_SOURCES.md) lists every included film and conflict.

Types are source labels, not a content-safety classifier. P31 animated-film
identity does not automatically create an Animation genre. `Apocalyptic` and
`Post-Apocalyptic` remain different tags. Both new UI labels preserve canonical
English API values. A catalog addition does not prove subscription supply;
previous `imdb:` watched identifiers still refer to their newly cataloged film.

Watchmode data remains in the ignored database: logical supply validity is
24 hours, retention at most 30 days. Cached offers are separately checked for
US + selected provider + `sub`; rent/buy/free/TVE do not qualify. Preview reads
only currently valid cache facts and never refreshes supply. Find films probes
at most 30 new titles per round; save refreshes the chosen film. A changed or
expired fact invalidates a saved preview basis. The page attributes Watchmode.

No real person's Netflix history or human group outcomes were tested. CSV
fixtures are constructed, reliable exact matches become watched, ambiguous or
unmatched titles stay pending, and a colon does not identify an episode. Profile
delete/withdrawal is verified through normal app interfaces; forensic erasure of
RAM, SSD blocks or backups is not claimed.

Sources: [Wikidata access](https://www.wikidata.org/wiki/Wikidata:Data_access),
[Wikidata CC0](https://www.wikidata.org/wiki/Wikidata:Licensing),
[Watchmode terms](https://api.watchmode.com/tc),
[Netflix export](https://help.netflix.com/en/node/101917),
[Ollama tool calling](https://docs.ollama.com/capabilities/tool-calling).
