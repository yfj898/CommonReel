> Historical prototype document (2026-10-13). Current v2: [DATA_V2.md](DATA_V2.md).

# Dataset and license boundary

Primary source: https://grouplens.org/datasets/movielens/1m/
ZIP: https://files.grouplens.org/datasets/movielens/ml-1m.zip
Terms: https://files.grouplens.org/datasets/movielens/ml-1m-README.txt

This build uses MovieLens solely as a local noncommercial research prototype
and offline evaluation. The terms prohibit redistribution without separate
permission, commercial or revenue-bearing use without permission, and any
implication of GroupLens endorsement. A public repository contains source and
fetch instructions, not MovieLens raw files or a rating-derived catalog export.
Do not turn the research demo into a commercial service or redistribute data
without the dataset owner's authorization. Participation in a prize contest
does not establish permission for commercial use.

Only movies.dat, ratings.dat and README are extracted. users.dat demographics
are not extracted. Room names and demo names are fictional; data profiles
are anonymous research histories, not actual invited movie-night participants.

Observed official ZIP download on 2026-10-13:

- Bytes: 5,917,549
- SHA-256: a6898adb50b9ca05aa231689da44c217cb524e7ebd39d264c56e2832f2c54e20
- movies.dat SHA-256: 0140fc2356357c1a851d0f52e893a1e4d3696df632c4141cea8d5bc3d621f0b9
- ratings.dat SHA-256: 506d64ca44484487c11dc2d9a28de5c54948213e6b96285e298afe28d6ea4e0f

These are observed hashes, not a publisher-signed checksum. The downloader
records the source and archive provenance; it rejects oversized files,
extracts only fixed allowlisted filenames, and refuses to overwrite extracted
data. The native Windows HTTPS client retrieved the official archive after
Go's HTTP client encountered a TLS handshake timeout on this network.

Reproduction uses a global chronological 80/20 timestamp split. Ties at the
cutoff remain in training. Movie priors, user means, cosine neighborhoods and
recommendations use training ratings exclusively. Held-out relevant items have
rating >= 4, were not previously rated, and have >= 10 training ratings. Users
need >= 20 training ratings and >= 5 eligible held-out positives. Candidate
sets contain every eligible unseen film, not sampled negatives.

No plot, poster, trailer, runtime, content rating, streaming price, or current
availability is included in MovieLens 1M, and CommonReel does not fabricate any.
Genre vetoes are enforced against the dataset's genre tags. Those tags are
manually entered and can be incomplete; tag compliance is not a verified
content-safety classification or a guarantee that a person will feel comfortable.

Citation: F. Maxwell Harper and Joseph A. Konstan. 2015. The MovieLens Datasets:
History and Context. ACM Transactions on Interactive Intelligent Systems 5, 4,
Article 19. https://doi.org/10.1145/2827872
