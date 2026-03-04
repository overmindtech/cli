# Snapshot Source

A discovery source that serves items from a snapshot file or URL, enabling local testing with fixed data and deterministic re-runs of v6 investigation jobs.

## Overview

The snapshot source loads a snapshot file (JSON or protobuf format) at startup and responds to NATS discovery queries (GET, LIST, SEARCH) with items from that snapshot. This enables:

- **Local testing**: Run backend services (gateway, api-server, NATS) locally with consistent snapshot data
- **Deterministic v6 re-runs**: Re-run change analysis and blast radius calculations with the same snapshot data
- **Consistent exploration**: Query the same fixed data set repeatedly for debugging and testing

## Features

- **Snapshot loading**: Loads snapshots from local files or HTTP(S) URLs (JSON or protobuf format)
- **Format detection**: Automatically detects JSON (`.json`) or protobuf (`.pb`) format
- **Wildcard scope support**: Single adapter handles all types and scopes in the snapshot
- **Full query support**: Implements GET, LIST, and SEARCH query methods
- **In-memory indexing**: Fast lookups by type, scope, GUN, or query string
- **Comprehensive tests**: Unit tests for loader, index, and adapter components

## Usage

### Configuration

The snapshot source requires a snapshot file or URL to be specified:

**Environment variables:**
- `SNAPSHOT_SOURCE` or `SNAPSHOT_PATH` or `SNAPSHOT_URL` - Path to snapshot file or HTTP(S) URL
- Standard discovery engine config (NATS connection, auth, etc.)

**Command-line flags:**
```bash
--snapshot-source <path-or-url>  # Path to snapshot file or URL (required)
--log <level>                     # Log level (default: info)
--json-log <bool>                 # JSON logging (default: true)
--health-check-port <port>        # Health check port (default: 8089)
```

### Running Locally

#### Option 1: With backend services (recommended)

1. Start backend services (gateway, api-server, NATS) in devcontainer or via docker-compose
2. Run the snapshot source:

```bash
ALLOW_UNAUTHENTICATED=true \
SNAPSHOT_SOURCE=/workspace/services/api-server/service/changeanalysis/testdata/snapshot.json \
NATS_SERVICE_HOST=nats \
NATS_SERVICE_PORT=4222 \
go run ./sources/snapshot/main.go --log=debug --json-log=false
```

#### Option 2: Using VS Code launch configuration

Use the provided launch configurations in `.vscode/launch.json`:

- **"snapshot-source (with backend)"**: For use when backend services are running
- **"snapshot-source (standalone)"**: For standalone debugging with local NATS

Update the `SNAPSHOT_SOURCE` environment variable in the launch config to point to your snapshot file.

#### Option 3: Load snapshot from URL

```bash
ALLOW_UNAUTHENTICATED=true \
SNAPSHOT_SOURCE=https://gateway-host/area51/snapshots/{uuid}/json \
NATS_SERVICE_HOST=nats \
NATS_SERVICE_PORT=4222 \
go run ./sources/snapshot/main.go
```

### Query Behavior

The snapshot source implements a **wildcard scope adapter** that handles all types and scopes:

- **LIST**: Returns all items in the snapshot (or filtered by scope if scope != "*")
- **GET**: Finds an item by its globally unique name (GUN) or unique attribute value
- **SEARCH**: Searches items by regex pattern on globally unique name

Example queries via the gateway:
```
LIST *.*                           # Returns all 179 items in test snapshot
GET *.* <globally-unique-name>     # Gets specific item by GUN
SEARCH *.* <regex-pattern>         # Finds items matching pattern
```

## Implementation Details

### Architecture

```
sources/snapshot/
├── main.go                 # Entrypoint
├── cmd/
│   └── root.go            # Cobra CLI setup, viper config
└── adapters/
    ├── loader.go          # Snapshot loading (file/URL)
    ├── index.go           # In-memory indexing
    ├── adapter.go         # Discovery adapter implementation
    └── main.go            # Adapter initialization
```

### Snapshot Index

The source builds in-memory indices for efficient querying:

- **By GUN**: Map of `GloballyUniqueName` → `*Item` for fast GET lookups
- **By type/scope**: Nested map for filtering by type and scope
- **All items**: Full list for wildcard LIST queries

### Adapter Strategy

The snapshot source uses **Option B from the design doc**: a single adapter with wildcard type (`*`) and wildcard scope (`*`). This adapter:

- Reports `Type() = "*"` and `Scopes() = ["*"]`
- Implements `WildcardScopeAdapter` interface
- Handles all query types (GET, LIST, SEARCH) across all types and scopes in the snapshot

This differs from "one adapter per (type, scope)" because the gateway's query expansion expects adapters to report specific types. The wildcard approach lets us serve any item from the snapshot regardless of type or scope.

## Testing

Run unit tests:
```bash
cd sources/snapshot/adapters
go test -v
```

Test snapshot loading:
```bash
cd sources/snapshot
go run main.go --snapshot-source=/path/to/snapshot.json --help
```

Verify with real snapshot:
```bash
cd sources/snapshot
go test -run TestLoadSnapshotFromFile -v ./adapters
```

## Example: Using with v6 Investigations

1. Download a snapshot from Area 51 or use an existing test snapshot
2. Start backend services locally (gateway, api-server, NATS)
3. Start the snapshot source pointing at your snapshot file
4. Run a v6 investigation - it will query from the snapshot instead of live sources
5. Re-run with the same snapshot for consistent, deterministic results

## Troubleshooting

**Error: "snapshot has no items"**
- Verify the snapshot file is valid protobuf and contains items
- Check file path or URL is correct

**Error: "api-key must be set"**
- Set `ALLOW_UNAUTHENTICATED=true` for local testing
- Or provide a valid API key via `API_KEY` env var

**Error: "could not connect to NATS"**
- Verify NATS is running at the configured host/port
- Check `NATS_SERVICE_HOST` and `NATS_SERVICE_PORT` are correct

## Related Documentation

- **Linear issue**: [ENG-2577](https://linear.app/overmind/issue/ENG-2577)
- **Snapshot protobuf**: `sdp/snapshots.proto`
- **Discovery engine**: `go/discovery/`
- **Test snapshots**: 
  - JSON format (recommended): `services/api-server/service/changeanalysis/testdata/snapshot.json`
  - Protobuf format (legacy): `services/api-server/service/changeanalysis/testdata/snapshot.pb`
