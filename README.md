# GOMAJOR

```
$ gomajor help

GoMajor is a tool for major version upgrades

Usage:

    gomajor <command> [arguments]

The commands are:

    get     upgrade to a major version
    list    list available updates
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
foo.go: github.com/go-redis/redis -> github.com/go-redis/redis/v8
bar.go: github.com/go-redis/redis -> github.com/go-redis/redis/v8
```

### Increment Module Path Version

```
$ gomajor path -next
module github.com/go-redis/redis/v9
bench_test.go: github.com/go-redis/redis/v8 -> github.com/go-redis/redis/v9
cluster.go: github.com/go-redis/redis/v8/internal -> github.com/go-redis/redis/v9/internal
cluster.go: github.com/go-redis/redis/v8/internal/hashtag -> github.com/go-redis/redis/v9/internal/hashtag
cluster.go: github.com/go-redis/redis/v8/internal/pool -> github.com/go-redis/redis/v9/internal/pool
cluster.go: github.com/go-redis/redis/v8/internal/proto -> github.com/go-redis/redis/v9/internal/proto
cluster.go: github.com/go-redis/redis/v8/internal/rand -> github.com/go-redis/redis/v9/internal/rand
# etc ...
```

### Features:

* Finds latest version.
* Rewrites your import paths.
* Lets you ignore SIV on the command line.

### Warning:

* By default, only cached content will be fetched from the module proxy (See `-cached` flag).
* If you have multiple major versions imported, **ALL** of them will be rewritten.
* The latest version will not be found if there are **gaps** between major version numbers.