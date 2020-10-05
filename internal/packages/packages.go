package packages

import (
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/tools/go/packages"
)

// Direct returns a list of all modules that are direct dependencies
func Direct(dir string) ([]*packages.Module, error) {
	cfg := packages.Config{
		Mode:  packages.NeedModule,
		Dir:   dir,
		Tests: true,
	}
	pkgs, err := packages.Load(&cfg, "all")
	if err != nil {
		return nil, err
	}
	direct := []*packages.Module{}
	seen := map[string]bool{}
	for _, pkg := range pkgs {
		if mod := pkg.Module; mod != nil && !mod.Indirect && mod.Replace == nil && !mod.Main && !seen[mod.Path] {
			seen[mod.Path] = true
			direct = append(direct, mod)
		}
	}
	return direct, nil
}

// ModPrefix returns the module path with no SIV
func ModPrefix(modpath string) string {
	prefix, _, ok := module.SplitPathVersion(modpath)
	if !ok {
		prefix = modpath
	}
	return prefix
}

// SplitPath splits the package path into the module path and the package subdirectory.
// It requires the a module path prefix to figure this out.
func SplitPath(modprefix, pkgpath string) (modpath, pkgdir string, ok bool) {
	if !strings.HasPrefix(pkgpath, modprefix) {
		return "", "", false
	}
	modpathlen := len(modprefix)
	if strings.HasPrefix(pkgpath[modpathlen:], "/") {
		modpathlen++
	}
	if idx := strings.Index(pkgpath[modpathlen:], "/"); idx >= 0 {
		modpathlen += idx
	} else {
		modpathlen = len(pkgpath)
	}
	modpath = modprefix
	if _, major, ok := module.SplitPathVersion(pkgpath[:modpathlen]); ok {
		modpath = JoinPath(modprefix, major, "")
	}
	pkgdir = strings.TrimPrefix(pkgpath[len(modpath):], "/")
	return modpath, pkgdir, true
}

// SplitSpec splits the path/to/package@target format strings
func SplitSpec(spec string) (path, target string) {
	parts := strings.SplitN(spec, "@", 2)
	if len(parts) == 2 {
		path = parts[0]
		target = parts[1]
	} else {
		path = spec
	}
	return
}

// JoinPath creates a full package path given a module prefix, version, and package directory.
func JoinPath(modprefix, version, pkgdir string) string {
	version = strings.TrimPrefix(version, ".")
	version = strings.TrimPrefix(version, "/")
	major := semver.Major(version)
	pkgpath := modprefix
	switch {
	case strings.HasPrefix(pkgpath, "gopkg.in"):
		pkgpath += "." + major
	case major != "" && major != "v0" && major != "v1" && !strings.Contains(version, "+incompatible"):
		if !strings.HasSuffix(pkgpath, "/") {
			pkgpath += "/"
		}
		pkgpath += major
	}
	if pkgdir != "" {
		pkgpath += "/" + pkgdir
	}
	return pkgpath
}
