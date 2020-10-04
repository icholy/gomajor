# GOMAJOR

```
$ gomajor help

GoMajor is an experimental tool for major version upgrades

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

### Features:

* Finds latest version.
* Rewrites your import paths.
* Lets you ignore SIV on the command line.

### Warning:

* `@v` suffix doesn't work for `+incompatible` versions (just use `go get`).
* If you have multiple major versions imported, **ALL** of them will be rewritten.
* `list` can miss newer versions if the subpackage structure changes.
* `@latest` and `@master` scrapes pkg.go.dev and will stop working at some point.
    * https://proxy.golang.org/ only allows listing minor versions.
    * https://github.com/golang/go/issues/36785
    * https://github.com/golang/go/issues/40323
    * https://github.com/golang/go/issues/40323