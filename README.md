# gomajor

[![Go Reference](https://pkg.go.dev/badge/github.com/icholy/gomajor.svg)](https://pkg.go.dev/github.com/icholy/gomajor)
[![Go Report Card](https://goreportcard.com/badge/github.com/icholy/gomajor)](https://goreportcard.com/report/github.com/icholy/gomajor)

A tool for major version upgrades of Go modules.

## Overview

`gomajor` helps you manage major version upgrades in your Go projects. It automates the process of upgrading dependencies to new major versions by rewriting import paths and updating your `go.mod` file.

## Installation

```sh
go install github.com/icholy/gomajor@latest
```

## Quick Start

```sh
# List available major version updates
gomajor list

# Upgrade a specific module to its latest major version
gomajor get github.com/go-redis/redis@latest

# Upgrade all modules to their latest major versions
gomajor get all
```

## Commands

### `gomajor list`

List available major version updates for your dependencies.

**Example:**
```sh
$ gomajor list
github.com/go-redis/redis/v8 [v6.15.9] [v8.11.5]
github.com/example/foo/v3 [v2.1.0] [v3.0.0]
```

### `gomajor get <module>[@version]`

Upgrade a module to a new major version. This command:
- Rewrites import paths in your code to use the new major version
- Updates `go.mod` with the new version
- Runs `go mod tidy` to clean up

**Examples:**

Update to the latest major version:
```sh
gomajor get github.com/go-redis/redis@latest
```

Update to a specific major version:
```sh
gomajor get github.com/go-redis/redis@v7
```

Update all modules to their latest major versions:
```sh
gomajor get all
```

**Flags:**
- `-cached` - Only use cached content from the module proxy (default: `true`)
- `-rewrite` - Rewrite import paths for all major versions of the module (default: `true`)

### `gomajor path [options]`

Modify the module path in your `go.mod` file. Useful when publishing a new major version of your own module.

**Examples:**

Increment to the next major version:
```sh
gomajor path -next
```

Change to a specific major version:
```sh
gomajor path -version v3
```

Change the module path entirely:
```sh
gomajor path goredis.io
```

## How It Works

When you upgrade to a new major version, `gomajor`:

1. **Analyzes** your code to find all imports of the specified module
2. **Rewrites** import paths to use the new major version suffix (e.g., `/v8`)
3. **Updates** your `go.mod` file with the new version
4. **Tidies** your module dependencies

### Example

Before:
```go
import "github.com/go-redis/redis/v6"
```

After running `gomajor get github.com/go-redis/redis@v8`:
```go
import "github.com/go-redis/redis/v8"
```

## Important Considerations

### Limitations

- **`replace` directives**: This tool does not understand `replace` directives in `go.mod`
- **Nested modules**: Nested module structures are not supported
- **Private modules**: Modules matching `GOPRIVATE` are skipped
- **Version gaps**: The latest version will not be found if there are gaps between major version numbers (e.g., v1, v3, v5 with no v2 or v4)
- **Package names**: The `path` command rewrites the module path but does not rewrite package names in your code

### Rewriting Behavior

By default, when you upgrade a module, `gomajor` will rewrite **ALL** import paths for that module, even if you have multiple major versions imported. Use the `-rewrite=false` flag to disable this behavior if needed.

### Module Proxy Caching

By default, `gomajor` only fetches cached content from the module proxy to avoid unnecessary network requests. Use the `-cached=false` flag if you need to fetch the latest information.

## Common Use Cases

### Upgrading All Dependencies

To upgrade all your dependencies to their latest major versions:

```sh
gomajor get all
```

### Publishing a New Major Version

When releasing a new major version of your module:

```sh
# Increment the major version in go.mod
gomajor path -next

# Update your code and publish
git tag v2.0.0
git push origin v2.0.0
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

See [LICENSE](LICENSE) file for details.
