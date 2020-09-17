# GOMAJOR

> This is an experimental tool for upgrading major versions

### Example:

```
$ gomajor github.com/go-redis/redis@latest
go get github.com/go-redis/redis/v8@v8.1.1
foo.go: github.com/go-redis/redis -> github.com/go-redis/redis/v8
bar.go: github.com/go-redis/redis -> github.com/go-redis/redis/v8
```

### Features:

* Finds latest version.
* Rewrites your import paths.
* Let's you ignore SIV on the command line.

### Warning:

* This tool has no dry-run or undo feature. Commit before running.
* If you have multiple major versions imported, ALL of them will be rewritten.
* `@latest` scrapes pkg.go.dev and will stop working at some point.
* `@master` is not supported.
* gopkg.in imports are not supported.
