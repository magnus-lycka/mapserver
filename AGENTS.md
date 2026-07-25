# Repository Guidelines

## Project Overview

This repository contains a Go-based real-time map server for Minetest/Luanti. It reads map blocks from a Minetest world database, renders them into map tiles, extracts searchable map objects, and serves an embedded Vue 2 web client over HTTP and WebSocket.

The Go module is named `mapserver`. The supported world database backends are SQLite and PostgreSQL.

## Architecture

- `main.go` parses parameters and configuration, creates the application context, starts background rendering, and serves HTTP.
- `app/setup.go` is the composition root. Add and connect application-level dependencies there.
- `db/` reads Minetest map blocks. Backend-specific implementations live in `db/sqlite/` and `db/postgres/` behind `db.DBAccessor`.
- `mapblockaccessor/` adds caching and queries for initial or changed blocks.
- `mapblockrenderer/` renders vertical ranges of map blocks into images.
- `tilerenderer/` creates map tiles and stores them in the filesystem-backed `tiledb/`.
- `tilerendererjob/` runs initial and incremental rendering jobs.
- `mapobject/` extracts POIs, signs, travelnets, and other objects when map blocks are rendered.
- `mapobjectdb/` stores map objects and settings. Keep its SQLite and PostgreSQL implementations aligned.
- `eventbus/` connects rendered blocks and tiles to synchronous listeners.
- `web/` contains HTTP handlers, WebSocket support, and Prometheus endpoints.
- `public/` contains the Vue 2 frontend. Its generated Rollup bundle is embedded into the Go binary by `public/embed.go`.
- `coords/` and `types/` contain shared coordinate conversions and domain types.

The main data flow is:

```text
world database -> MapBlockAccessor -> MapBlockRenderer -> TileRenderer -> tile storage/API
```

Rendered map-block events synchronously trigger map-object extraction. Rendering progress and map-object changes are sent to browser clients through the web event bus and WebSocket endpoint.

## Development Guidelines

- Keep changes focused. Do not combine feature work or bug fixes with unrelated modernization or broad refactoring.
- Follow the existing package boundaries and wire new dependencies through `app.Setup`.
- Preserve both SQLite and PostgreSQL behavior when changing database interfaces, queries, or schemas.
- Add map-object database migrations under both backend-specific `migrations/` directories when applicable.
- Use `logrus` and the package-local logger conventions already present in the affected package.
- Fatal startup and background-job errors frequently use `panic` in the existing code. Do not change the wider error-handling model as part of an unrelated change.
- The event bus invokes listeners synchronously while holding a read lock. Avoid slow, blocking, or re-entrant listener behavior.
- Tile files and generated frontend bundles are runtime artifacts with established locations and formats; preserve compatibility unless a migration is explicitly part of the task.
- Run `gofmt` on all changed Go files.

## Configuration Caveats

- Configuration is read from `MT_CONFIG_PATH`, defaulting to `mapserver.json`.
- Unless `MT_READONLY=true`, startup writes the completed configuration through `Config.Save()` to `mapserver.json` in the current working directory, regardless of the input configuration path.
- Treat local, untracked `mapserver.json` files as user-owned environment configuration. Do not modify or commit them unless explicitly requested.
- `world.mt` selects the Minetest map backend. Application setup also performs database migrations, so integration-style startup is not read-only.

## Testing

Run the complete Go test suite after Go changes:

```sh
go test ./...
```

Also run static analysis:

```sh
go vet ./...
```

If the normal Go build cache is not writable in a restricted environment, use a temporary cache outside the repository, for example:

```sh
GOCACHE=/tmp/mapserver-go-build go test ./...
GOCACHE=/tmp/mapserver-go-build go vet ./...
```

Tests are package-local and commonly use temporary SQLite databases plus fixtures from `testutils/testdata`. Add focused tests close to the changed code and follow the surrounding test style. Rendering benchmarks exist in `mapblockrenderer/` and `tilerenderer/`, but are not part of the normal CI test command.

For frontend changes, install dependencies and run both checks from `public/`:

```sh
npm ci
npm run jshint
npm run bundle
```

Commit the regenerated `public/js/bundle.js` and source map when frontend source changes require them.

## CI and Build

GitHub Actions verifies Go tests and coverage, frontend JSHint, frontend bundling, and container builds. The production Docker build first creates the frontend bundle and then builds a static Go binary with `CGO_ENABLED=0`.

The development environment is defined in `docker-compose.yml`. See `doc/dev.md` for SQLite and PostgreSQL startup commands and `doc/incrementalrendering.md` for the rendering model.

Before handing off a change, report which checks were run and any checks that could not be run. Do not claim frontend verification if dependencies were unavailable.
