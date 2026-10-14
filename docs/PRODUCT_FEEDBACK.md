> Historical prototype document (2026-10-13). Current v2: [PRODUCT_FEEDBACK_V2.md](PRODUCT_FEEDBACK_V2.md).

# Actual product feedback and friction log

Observed through 2026-10-13. This file must describe tools actually used.
No AWS runtime service, Alexa device, Fire TV, Bee or Ring API was used.

## Amazon Alexa+ hackathon documentation

Used for selecting the primary track and defining a web simulation that
demonstrates MCP. The explicit protocol minimum and simulated-experience
allowance made it possible to build a complete local Go application.

The examples would be more useful with a tiny Go Streamable HTTP server and
a browser companion that includes initialize, tools/list and tools/call.
We would use the open MCP path again. Device authentication/onboarding was
not attempted, so this is not feedback on device integration.

## Official MCP Go SDK v1.8.0

Used for typed tools, schema inference, client/server tests, Streamable HTTP
and stdio. The transport handler and typed handlers kept the application
logic small. Explicit protocol restriction made the contest requirement
testable. A live HTTP test confirms version 2025-11-25 and discovers nine tools.

The web companion still needs a small client implementation. A maintained
minimal browser example, including stateless JSON-response handling, would
make this path easier to learn. We would build with the SDK again.

## MovieLens / GroupLens

Used for a local, noncommercial research benchmark and demo. File formats and
rating timestamps are easy to parse in Go. The README gives clear separate
license terms. The age of the data and lack of runtimes/streaming availability
limit product claims. We would use it again for reproducible offline tests,
not as an unlicensed commercial catalog.

Observed friction: Go's HTTPS downloader hit a TLS handshake timeout for the
official ZIP on this Windows network. Native Invoke-WebRequest retrieved the
same official archive. Severity: medium. Workaround: the fetch command accepts
an already downloaded official ZIP and records its observed hash. This is a
network-path observation, not a proven server or SDK defect.

## modernc SQLite

Used for persistent rooms and atomic idempotency receipts. A pure Go driver
made the Windows quickstart possible without a C compiler. Transaction and
restart tests passed. This prototype uses a single database connection and
serialized writes; scale beyond a local movie-night app remains untested.

## Honest submission boundary

The contest's optional friction log should include only relevant observed
issues. Do not invent Amazon API failures, model calls, credits, device tests,
human feedback, or endorsement. The current log supports a request for better
Go/browser MCP examples, not a claimed device-side bug.
