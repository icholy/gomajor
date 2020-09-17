package packages

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/tools/go/packages"
)

type Package struct {
	PkgDir    string
	ModPrefix string
}

func Load(pkgpath string) (*Package, error) {
	// create temp module directory
	dir, err := TempModDir()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	// find the module root
	cfg := &packages.Config{
		Dir:  dir,
		Mode: packages.NeedName | packages.NeedModule,
	}
	pkgs, err := packages.Load(cfg, pkgpath)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("failed to find module: %s", pkgpath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, pkg.Errors[0]
	}
	// remove the existing version if there is one
	modprefix := pkg.Module.Path
	if prefix, _, ok := module.SplitPathVersion(modprefix); ok {
		modprefix = prefix
	}
	pkgdir := strings.TrimPrefix(pkg.PkgPath, pkg.Module.Path)
	pkgdir = strings.TrimPrefix(pkgdir, "/")
	return &Package{
		PkgDir:    pkgdir,
		ModPrefix: modprefix,
	}, nil
}

func (pkg Package) ModPath(version string) string {
	return JoinPathMajor(pkg.ModPrefix, semver.Major(version))
}

func (pkg Package) Path(version string) string {
	return path.Join(pkg.ModPath(version), pkg.PkgDir)
}

func (pkg Package) FindModPath(pkgpath string) (string, bool) {
	if !strings.HasPrefix(pkgpath, pkg.ModPrefix) {
		return "", false
	}
	prefixlen := len(pkg.ModPrefix)
	if s := pkgpath[prefixlen:]; s != "" && s[0] == '/' {
		prefixlen++
	}
	idx := strings.Index(pkgpath[prefixlen:], "/")
	if idx < 0 {
		return pkg.ModPrefix, true
	}
	if prefix, major, ok := module.SplitPathVersion(pkgpath[:prefixlen+idx]); ok && major != "" {
		return JoinPathMajor(prefix, major), true
	}
	return pkg.ModPrefix, true
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

func JoinPathMajor(path, major string) string {
	major = strings.TrimPrefix(major, "/")
	if strings.HasPrefix(path, "gopkg.in/") {
		if major == "" {
			major = "v1"
		}
		return path + "." + major
	}
	if major == "v0" || major == "v1" || major == "" {
		return path
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path + major
}

func TempModDir() (string, error) {
	dir, err := ioutil.TempDir("", "gomajor_*")
	if err != nil {
		return "", err
	}
	modfile := "module temp"
	modpath := filepath.Join(dir, "go.mod")
	err = ioutil.WriteFile(modpath, []byte(modfile), os.ModePerm)
	if err != nil {
		_ = os.RemoveAll(dir) // best effort
		return "", err
	}
	return dir, nil
}
