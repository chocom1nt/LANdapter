# Testing

## Run automated tests

```sh
make test
```

This runs the Go test suite and the frontend translation checks (`npm test` in `web/`). The frontend translation tests use Node's built-in test runner and do not require installing browser dependencies.

## Seed demo clients for UI/TUI smoke tests

With the Compose PostgreSQL service running and migrations applied:

```sh
make seed-demo-agents
```

This inserts six deterministic **database fixtures**, not real connected agents. They are useful for testing the client table, filtering, counts, and the TUI list. All fixtures are offline by design because they have no running WebSocket agents. The clients API reports online status from active connections, not stale database flags.

Remove only these fixtures with:

```sh
make clear-demo-agents
```

Do not run these commands against a production database.

## Build checks

```sh
cd web && npm ci && npm run build
```

Run `go test ./...` with the Go version declared in `go.mod`. Integration tests require their test dependencies and database environment to be configured separately.
