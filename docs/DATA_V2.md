# v2 data boundaries

Verified 2026-10-14. The v2 catalog is independent of the MovieLens user-kNN
experiment in DATA.md and LEGACY_PROTOTYPE.md.

| Source | Use | Distribution / limitations |
|---|---|---|
| Wikidata structured entity data | Stable QIDs, IMDb cross-check, English titles/aliases, minute-valued P2047 runtime and explicit genre tags | CC0 metadata in `assets/`; no encyclopedia prose or images |
| GroupLens ml-latest-small links | IMDb identifier seeds, cross-checked against Wikidata; title requests alone are never accepted as joins | Raw ZIP, ratings and research users stay out of the v2 bundle |
| Watchmode Free Developer API | App-only US source matching and subscription listings | Own key; 2,500-credit account plan observed; no image license inferred; no raw responses or caches published |
| Netflix CSV supplied by the owner | Known watched movies and pending titles, not likes or satisfaction | Optional; local matching; no raw file retained or external upload |
| qwen3.5:4b on Ollama | Local tool planning; Go owns constraints and final factual output | Model weights not bundled; observed digest in evidence; no cloud fallback |

Wikidata SPARQL returned real HTTP 429 during its service rate-limiting
incident. The build uses the official `wbgetentities` API and cached entity
responses instead. The manifest contains 650 seeds, 217 included films,
exclusions, stable identifiers, the genre map and content SHA-256. Runtime is
accepted only with unit Q7727; missing/conflicting minute values are excluded.
English labels for all 168 observed genre identifiers were verified against the official entity API. The fixed 217-film selection and explicit source-label map are checked in; this is not a complete genre ontology or a content safety classifier. Composite
comedy drama retains both tags; biography, monster, coming-of-age and kung fu
are not relabeled as comedy/animation/drama.

Watchmode non-image data stays in the ignored local database. Logical supply
validity is 24 hours. Rows older than 30 days are pruned on startup and each
cleanup sweep; account cancellation also requires operator cache removal.
Attribution appears on the page wherever supply is shown. Renting, buying,
free and TVE listings do not satisfy a selected subscription. A platform
listing does not verify personal account rights or add-on subscriptions.

The live report publishes counts, status codes, request time and a response
hash, rather than the licensed payload. A real invalid-title request returned
404. Timeout, 429 and quota-unknown scenarios are controlled tests; the free
quota was not deliberately exhausted. Production code cannot upgrade plans.

No real Netflix export or actual person's viewing history has been tested.
CSV fixtures are constructed. Exact matching is limited to verified catalog
titles/aliases and year variants; fuzzy text is never auto-added as watched.
Known-episode classification needs explicit reference evidence; unknown
series-like titles stay pending, and a colon alone proves nothing. The
production snapshot has no general episode database.

Deletion is verified through normal app interfaces and minimal receipts.
It is not forensic erasure of RAM, SSD blocks or backups; SQLite WAL and
operator backups remain an operational boundary. No full room/profile JSON
is stored in idempotency responses.

Primary sources: [Wikidata access](https://www.wikidata.org/wiki/Wikidata:Data_access),
[Wikidata licensing](https://www.wikidata.org/wiki/Wikidata:Licensing),
[Watchmode terms](https://api.watchmode.com/tc),
[Watchmode free key](https://api.watchmode.com/requestApiKey/),
[Netflix export](https://help.netflix.com/en/node/101917),
[Ollama tools](https://docs.ollama.com/capabilities/tool-calling).
