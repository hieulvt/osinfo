# osinfo

Production-ready CLI tool written in Go to inspect host operating system, CPU, memory, and disk volume telemetry.

## Project Layout

```text
.
├── cmd/osinfo/main.go            # Application entrypoint & signal handling
├── internal/
│   ├── cli/                      # Cobra command tree & flags
│   ├── collector/                # Domain models & gopsutil system collectors
│   ├── formatter/                # Table and JSON output formatters
│   └── version/                  # Build-time metadata (ldflags)
├── Dockerfile                    # Multi-stage static container build
├── Makefile                      # Build, test, vet, and cross-compile targets
└── go.mod / go.sum
```

## Commands

```bash
osinfo                  # Show all system info (same as `osinfo all`)
osinfo all              # Show OS, CPU, RAM, and Disk volumes
osinfo os               # Show OS, platform, kernel, arch, uptime
osinfo cpu              # Show CPU model, cores, clock, usage
osinfo ram              # Show RAM and swap usage (aliases: mem, memory)
osinfo disk             # Show mounted disk volumes (alias: disks)
osinfo disk --path /    # Show disk usage for a specific path
osinfo version          # Show build and Go runtime version
```

### Global Flags

- `-o, --output string`: Output format (`table` or `json`, default `table`)
- `--timeout duration`: Collection timeout (default `5s`)

## Local Development

```bash
# Run directly without building a binary
go run ./cmd/osinfo all

# Run tests with race detector & vet
make vet test-race

# Build local binary into ./bin/osinfo
make build
./bin/osinfo all

# Install into $GOPATH/bin (or $HOME/go/bin) so `osinfo` works globally
make install
```

## Production Build & Deployment

```bash
# Build stripped static binaries for Linux/macOS/Windows into ./dist/
make build-all

# Install system-wide on a Linux server
sudo install -m 0755 bin/osinfo /usr/local/bin/osinfo
osinfo all -o json

# Build and run via Docker
docker build -t osinfo:0.1.0 .
docker run --rm osinfo:0.1.0 all -o json
```

