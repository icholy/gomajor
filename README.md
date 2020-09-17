# GOMAJOR

> This is an experimental tool for upgrading major versions

Features:

* Let's you ignore SIV on the command line.
* Automatically rewrites your import paths.

Example:

```
gomajor github.com/go-redis/redis@v8.0.0
```

Warning:

* This tool has no dry-run or undo feature. Commit before running.
* If you have multiple major versions imported, ALL of them will be rewritten.
* `@master` is not supported.
* `@latest` scrapes pkg.go.dev
* gopkg.in imports are not supported.
