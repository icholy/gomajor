package packages

import (
	"path"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/tools/go/packages"
)

type Package struct {
	Version   string
	PkgDir    string
	ModPrefix string
}

func Direct(dir string) ([]*Package, error) {
	cfg := packages.Config{
		Mode:  packages.NeedModule,
		Dir:   dir,
		Tests: true,
	}
	pkgs, err := packages.Load(&cfg, "all")
	if err != nil {
		return nil, err
	}
	direct := []*Package{}
	seen := map[string]bool{}
	for _, pkg := range pkgs {
		if mod := pkg.Module; mod != nil && !mod.Indirect && mod.Replace == nil && !mod.Main && !seen[mod.Path] {
			seen[mod.Path] = true
			modprefix := pkg.Module.Path
			if prefix, _, ok := module.SplitPathVersion(modprefix); ok {
				modprefix = prefix
			}
			direct = append(direct, &Package{
				Version:   mod.Version,
				ModPrefix: modprefix,
			})
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

func (pkg Package) ModPath() string {
	return JoinPathMajor(pkg.ModPrefix, pkg.Version)
}

func (pkg Package) Path() string {
	return path.Join(pkg.ModPath(), pkg.PkgDir)
}

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
		modpath = JoinPathMajor(modprefix, major)
	}
	pkgdir = strings.TrimPrefix(pkgpath[len(modpath):], "/")
	return modpath, pkgdir, true
}

func (pkg Package) FindModPath(pkgpath string) (string, bool) {
	modpath, _, ok := SplitPath(pkg.ModPrefix, pkgpath)
	return modpath, ok
}

func SplitSpec(spec string) (path, version string) {
	parts := strings.SplitN(spec, "@", 2)
	if len(parts) == 2 {
		path = parts[0]
		version = parts[1]
	} else {
		path = spec
	}
	return
}

func JoinPathMajor(path, version string) string {
	version = strings.TrimPrefix(version, ".")
	version = strings.TrimPrefix(version, "/")
	major := semver.Major(version)
	if strings.HasPrefix(path, "gopkg.in") {
		return path + "." + major
	}
	if major == "" || major == "v0" || major == "v1" || strings.Contains(version, "+incompatible") {
		return path
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path + major
}
