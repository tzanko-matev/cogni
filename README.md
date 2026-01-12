# cogni
A Cognitive test suite

## Development

- Enter the dev shell with `nix develop` (or direnv).
- Rate limiter integration tests require TigerBeetle. In the Nix shell, `TB_BIN`
  is exported automatically; otherwise set `TB_BIN=/path/to/tigerbeetle` before
  running tests.
