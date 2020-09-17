# GOMAJOR

> This is an experimental tool for upgrading major versions

Example:

```
$ gomajor github.com/go-redis/redis@latest
go get github.com/go-redis/redis/v8@v8.1.1
foo.go: github.com/go-redis/redis -> github.com/go-redis/redis/v8
bar.go: github.com/go-redis/redis -> github.com/go-redis/redis/v8
```

Features:

* Let's you ignore SIV on the command line.
* Automatically rewrites your import paths.

Warning:

* This tool has no dry-run or undo feature. Commit before running.
* If you have multiple major versions imported, ALL of them will be rewritten.
* `@master` is not supported.
* `@latest` scrapes pkg.go.dev
* gopkg.in imports are not supported.
