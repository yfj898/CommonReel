> Historical prototype document (2026-10-13). Current v2: [MCP_V2.md](MCP_V2.md).

# MCP transport evidence

CommonReel uses github.com/modelcontextprotocol/go-sdk v1.8.0. The SDK is the
official Model Context Protocol Go implementation. Server protocol support is
explicitly restricted to **2025-11-25**, meeting the contest's minimum version.

The HTTP endpoint is **/mcp**, using the SDK's Streamable HTTP handler with
stateless requests and JSON responses. Stateless transport does not mean
stateless application: rooms and mutation receipts persist in SQLite.

The web companion sends initialize, notifications/initialized, and actual
tools/call JSON-RPC messages with both supported Accept types and the
MCP-Protocol-Version header. It does not bypass tools through a mutation REST
API. A client can call tools/list to discover the nine typed tools and schemas.

The assistant in the companion is a **guided scripted simulation**. It routes
four documented intents into real tools. Unsupported input does not mutate a
room. This is not general language understanding, live Alexa+, or LLM inference.
External MCP agents can use the same service for their own planning.

Rooms are independent of an HTTP transport session. Members share a random
room capability in tool arguments. On a mutation, read current revision and
generate an operation_key (8–128 bytes). Reuse it only for an identical retry;
the database rejects key reuse with different arguments. Stale revisions fail
without changing state. save_choice rechecks eligibility before recording
the selected movie, explanation and source revision.

Integration test covers initialize at 2025-11-25, tool discovery, create,
member edit, policy change, shortlist, save, identical save retry, stale save
rejection, and receipt readback over real HTTP transport. Synthetic fixtures
isolate transport/reliability behavior; they are not a MovieLens quality test.

Live local probe: go run ./cmd/mcp-check -endpoint http://127.0.0.1:8092/mcp

For a compatible MCP host:

```json
{
  "mcpServers": {
    "commonreel": {
      "url": "http://127.0.0.1:8092/mcp"
    }
  }
}
```

Exact host configuration keys vary by client. The service is local-only and
has no device-side Alexa authentication or certification. Do not expose it
through a public tunnel without implementing and verifying authentication.
