# GOMAJOR

> A tool for major version upgrades

## Installation

```sh
go install github.com/icholy/gomajor@latest
```

## Commands

* `get` - Upgrade to a major version
* `list` - List available updates
* `path` - Modify the module path

Usage format is as follows: `gomajor <command> [arguments]`

## Usage

#### List Updates

```
gomajor list
```

#### Update a module to its latest version

```
gomajor get github.com/go-redis/redis@latest
```

#### Switch a module to a specific version

```
gomajor get github.com/go-redis/redis@v7
```

### Update all mobules to their latest version

```
gomajor get all
```

#### Increment module path version

```
gomajor path -next
```

#### Change module path version

```
gomajor path -version v3
```

#### Change module path

```
gomajor path goredis.io
```

### Warning:

* This tool does not understand `replace` directives or nested modules.
* By default, only cached content will be fetched from the module proxy (See `-cached` flag).
* If you have multiple major versions imported, **ALL** of them will be rewritten.
* The latest version will not be found if there are **gaps** between major version numbers.
* The `path` command does not rewrite package names.
* Modules matching `GOPRIVATE` are skipped.
