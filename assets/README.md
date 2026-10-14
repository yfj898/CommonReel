# Catalog snapshot

`catalog-v2.json` contains 217 movie records derived from Wikidata structured
data (CC0). The adjacent manifest records the SHA-256, stable identifiers,
source, builder and exclusions. IMDb identifiers from GroupLens
`ml-latest-small/links.csv` were used only to verify joins; no ratings, user
profiles, images, encyclopedia prose or Watchmode responses are bundled here.

Runtime values were accepted only from Wikidata P2047 values whose unit is
Q7727 (minute). Missing/conflicting values and missing canonical genre tags
were excluded. The mapping report is bounded to 650 seed records, not all
cinema. It does not imply complete alias or remake coverage.

Rebuild explicitly with `python scripts/build-catalog-entities.py` after
downloading `ml-latest-small.zip` from GroupLens into local `data/`. Review the
new manifest, then copy its two generated catalog files into this directory.
Supplier availability is always separate, private and time limited.
