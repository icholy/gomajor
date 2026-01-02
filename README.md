# gomajor

A tool for major version upgrades in Go modules.

`gomajor` helps you upgrade Go module dependencies to new major versions and automatically rewrites import paths in your code.

## Installation

```sh
go install github.com/icholy/gomajor@latest
```

## Why gomajor?

When a Go module releases a new major version (e.g., `v2`, `v3`), the import paths must change according to semantic import versioning. This tool automates the process of:

1. Upgrading the dependency in `go.mod`
2. Rewriting all import paths in your code to match the new major version

## Commands

### `gomajor list`

List available updates for your dependencies.

```sh
# List all available updates
gomajor list

# List updates for specific modules
gomajor list github.com/go-redis/redis

# Show only major version updates
gomajor list -major

# Include prerelease versions
gomajor list -pre

# Output as JSON
gomajor list -json
```

**Flags:**
- `-major` - Only show newer major versions
- `-pre` - Allow non-v0 prerelease versions
- `-cached` - Only fetch cached content from the module proxy (default: `true`)
- `-json` - Output in JSON format
- `-dir` - Working directory (default: `.`)

### `gomajor get`

Upgrade a module to a new major version and rewrite imports.

```sh
# Upgrade to the latest version
gomajor get github.com/go-redis/redis@latest

# Upgrade to a specific major version
gomajor get github.com/go-redis/redis@v7

# Upgrade all modules to their latest versions
gomajor get all
```

**Flags:**
- `-major` - Only get newer major versions
- `-pre` - Allow non-v0 prerelease versions
- `-cached` - Only fetch cached content from the module proxy (default: `true`)
- `-rewrite` - Only rewrite imports matching this regex (default: `.*`)
- `-dir` - Working directory (default: `.`)

### `gomajor path`

Modify your module's own path (useful when releasing a new major version of your module).

```sh
# Increment to the next major version
gomajor path -next

# Set a specific major version
gomajor path -version v3

# Change the module path entirely
gomajor path github.com/username/new-module-name
```

**Flags:**
- `-next` - Increment the module path version
- `-version` - Set the module path version
- `-dir` - Working directory (default: `.`)

### `gomajor version`

Print the gomajor version.

```sh
gomajor version
```

## Examples

### Upgrading a dependency

Before:
```go
import "github.com/go-redis/redis/v8"
```

```sh
gomajor get github.com/go-redis/redis@v9
```

After:
```go
import "github.com/go-redis/redis/v9"
```

### Releasing a new major version

When you're ready to release v2 of your own module:

```sh
gomajor path -next
```

This updates your `go.mod` and rewrites all internal imports to use the new version path.

## Important Notes

- This tool does not understand `replace` directives or nested modules
- By default, only cached content will be fetched from the module proxy (use `-cached=false` to bypass)
- If you have multiple major versions imported, **ALL** of them will be rewritten (use `-rewrite` flag to control this)
- The latest version will not be found if there are **gaps** between major version numbers
- The `path` command updates the module path but does not rewrite package names
- Modules matching `GOPRIVATE` are skipped

## License

See [LICENSE](LICENSE) file.
