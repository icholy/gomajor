# GOMAJOR

> This is an experimental tool for upgrading major versions

Features:

* Let's you ignore SIV on the command line.
* Automatically rewrites your import paths.

Example:

```
gomajor github.com/go-redis/redis@v8.0.0
```

**Note:** this will work even though the major version isn't specified in the package path.

