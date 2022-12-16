# GOMAJOR

```
$ gomajor help

GoMajor is a tool for major version upgrades

Usage:

    gomajor <command> [arguments]

The commands are:

    get     upgrade to a major version
    list    list available updates
    path    modify the module path
    version print the gomajor version
    help    show this help text
```

### List Updates

```
$ gomajor list
github.com/go-redis/redis: v6.15.9+incompatible [latest v8.1.3]
```

### Update and Rewrite Imports

```
$ gomajor get github.com/go-redis/redis@latest
go get github.com/go-redis/redis/v8@v8.1.3
foo.go:4:2 github.com/go-redis/redis/v8
bar.go:5:2 github.com/go-redis/redis/v8
```

**Note:** Use `gomajor get all` to update all modules.

### Increment Module Path Version

```
$ gomajor path -next
module github.com/go-redis/redis/v9
bench_test.go:11:2 github.com/go-redis/redis/v9
cluster.go:15:2 github.com/go-redis/redis/v9/internal
cluster.go:16:2 github.com/go-redis/redis/v9/internal/hashtag
# etc ...
```

### Change Module Path

```
$ gomajor path goredis.io
module goredis.io
bench_test.go:11:2 goredis.io
cluster.go:15:2 goredis.io/internal
cluster.go:16:2 goredis.io/internal/hashtag
# etc ...
```

### Features:

* Finds latest version.
* Rewrites your import paths.
* Lets you ignore SIV on the command line.
* Update your module's major version.

### Warning:

* This tool does not understand `replace` directives or nested modules.
* By default, only cached content will be fetched from the module proxy (See `-cached` flag).
* If you have multiple major versions imported, **ALL** of them will be rewritten.
* The latest version will not be found if there are **gaps** between major version numbers.
* The `path` command does not rewrite package names.
* Modules matching `GOPRIVATE` are skipped.