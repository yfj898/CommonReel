# Reproducible 247-film metadata snapshot

`catalog-v2.json` contains the retained **217** Wikidata film objects plus
**30** checked recent films (2015–2024). All original film and IMDb identifiers
are preserved. The loader-required checksum and count are in the manifest.

The legacy audit used 650 GroupLens IMDb seeds. The addition uses 30 fixed
Wikidata/IMDb/title/year tuples and retained official CC0 responses under
`catalog-sources/modern-2026-10-21`. A unique positive P2047 minute value and
reviewed explicit source genre tags are required; conflicting films were
excluded, not resolved by choosing a convenient duration.

From the project root, rebuild without network/key/MovieLens download:

```sh
python scripts/build-catalog-entities.py --modern-only
```

Review `data/catalog-v2.json` and its manifest, then copy both here. The command
also synchronizes the full selection and genre map. `catalog-base-selection.json`
retains the original 217 identifier boundary. The original full legacy builder
is still available with the separate local MovieLens ZIP.

[Per-film provenance and exclusions](../docs/MODERN_CATALOG_SOURCES.md).
Metadata is CC0. Movie posters, research ratings/users, personal history and
Watchmode raw responses/caches are not included. Subscription supply is checked
separately at runtime and remains time limited.
