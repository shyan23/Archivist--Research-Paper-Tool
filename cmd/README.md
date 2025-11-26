# Archivist CLI Commands

This directory contains the CLI application entry points and command definitions.

## Structure

```
cmd/
├── main/                    # Main CLI application
│   ├── main.go             # Entry point
│   └── commands/           # Cobra command definitions
│       ├── root.go         # Root command
│       ├── process.go      # Process papers
│       ├── search.go       # Search functionality
│       ├── chat.go         # Interactive chat
│       ├── index.go        # Indexing operations
│       ├── list.go         # List papers
│       ├── cache.go        # Cache management
│       ├── models.go       # Model management
│       └── other.go        # Utility commands
│
└── graph-init/             # Knowledge graph initialization tool
    └── main.go             # Standalone graph setup utility
```

## Building

### Main CLI Application

```bash
# Build main archivist binary
go build -o archivist cmd/main/main.go

# Or using Makefile
make build
```

### Graph Initialization Tool

```bash
# Build graph-init utility
go build -o graph-init cmd/graph-init/main.go
```

## Available Commands

### Main Application (`archivist`)

Run `./archivist --help` to see all commands:

- **process** - Process PDF papers and generate LaTeX reports
- **search** - Search through processed papers
- **chat** - Interactive chat with your paper library
- **index** - Build and manage search indexes
- **list** - List processed papers
- **cache** - Manage Redis cache
- **models** - Manage AI models

### Graph Initialization (`graph-init`)

Standalone utility to initialize the knowledge graph:

```bash
./graph-init
```

This sets up:
- Neo4j schema (constraints, indexes)
- Qdrant collection
- Initial graph structure

## Command Structure

Commands use the Cobra CLI framework:

```go
// cmd/main/commands/root.go
var rootCmd = &cobra.Command{
    Use:   "archivist",
    Short: "Research Paper Helper",
    Long:  "AI-powered research paper processing and analysis",
}
```

Each command is defined in its own file for modularity.

## Adding New Commands

1. Create new file in `cmd/main/commands/`
2. Define command using Cobra
3. Register in `root.go`

Example:

```go
// cmd/main/commands/mycommand.go
package commands

import "github.com/spf13/cobra"

var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "My new command",
    Run: func(cmd *cobra.Command, args []string) {
        // Implementation
    },
}

func init() {
    rootCmd.AddCommand(myCmd)
}
```

## Graph Commands Integration

Knowledge graph commands are integrated in the main CLI:

- `archivist process --with-graph` - Process with graph building
- `archivist search` - Hybrid search (vector + graph)
- `archivist graph stats` - Show graph statistics
- `archivist graph rebuild` - Rebuild knowledge graph
- `archivist cite show` - Citation analysis
- `archivist similar` - Find similar papers

See `cmd/main/commands/` for implementation details.

## Dependencies

Commands depend on:
- `internal/` - Core logic and implementations
- `pkg/` - Shared utilities
- Third-party: Cobra, Viper for CLI framework

## Testing

```bash
# Test commands
go test ./cmd/...

# Run specific command
./archivist process --help
./archivist search --help
```

## Environment Variables

Commands can use environment variables from:
- `.env` file
- `config/config.yaml`
- System environment

Priority: CLI flags > env vars > config file > defaults

## Notes

- Main entry point: `cmd/main/main.go`
- Commands are modular and can be refactored
- Graph-init is separate for standalone graph setup
- All commands support `--help` flag

For detailed usage, see main project documentation.
