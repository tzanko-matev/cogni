# cogni
A Cognitive test suite

## Development

- Enter the dev shell with `nix develop` (or direnv).
- Rate limiter integration tests require TigerBeetle. In the Nix shell, `TB_BIN`
  is exported automatically; otherwise set `TB_BIN=/path/to/tigerbeetle` before
  running tests.

## Rate limiter

- Example server config: `cmd/ratelimiterd/config.yaml`
- Example limits registry: `cmd/ratelimiterd/limits.json`
- Run the server: `go run ./cmd/ratelimiterd --config cmd/ratelimiterd/config.yaml`
- Run a load test: `go run ./cmd/ratelimiter-loadtest --mode=http --limits cmd/ratelimiterd/limits.json`
