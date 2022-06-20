package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icholy/gomajor/internal/tempmod"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/tools/go/packages"
)

// ModPrefix returns the module path with no SIV
func ModPrefix(modpath string) string {
	prefix, _, ok := module.SplitPathVersion(modpath)
	if !ok {
		prefix = modpath
	}
	return prefix
}

// ModMajor returns the major version in vN format
func ModMajor(modpath string) (string, bool) {
	_, major, ok := module.SplitPathVersion(modpath)
	if ok {
		major = strings.TrimPrefix(major, "/")
		major = strings.TrimPrefix(major, ".")
	}
	return major, ok
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
	if major, ok := ModMajor(pkgpath[:modpathlen]); ok {
		modpath = JoinPath(modprefix, major, "")
	}
	pkgdir = strings.TrimPrefix(pkgpath[len(modpath):], "/")
	return modpath, pkgdir, true
}

// SplitSpec splits the path/to/package@query format strings
func SplitSpec(spec string) (path, query string) {
	parts := strings.SplitN(spec, "@", 2)
	if len(parts) == 2 {
		path = parts[0]
		query = parts[1]
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

// FindModFile recursively searches up the directory structure until it
// finds the go.mod, reaches the root of the directory tree, or encounters
// an error.
func FindModFile(dir string) (string, error) {
	var err error
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		name := filepath.Join(dir, "go.mod")
		_, err := os.Stat(name)
		if err == nil {
			return name, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		if dir == "" || dir == "." || dir == "/" {
			break
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("cannot find go.mod")
}

func loadModFile(dir string) (*modfile.File, error) {
	name, err := FindModFile(dir)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return modfile.ParseLax(name, data, nil)
}

// Direct returns a list of all modules that are direct dependencies
func Direct(dir string) ([]module.Version, error) {
	file, err := loadModFile(dir)
	if err != nil {
		return nil, err
	}
	var mods []module.Version
	for _, req := range file.Require {
		if !req.Indirect {
			mods = append(mods, req.Mod)
		}
	}
	return mods, nil
}

// IsInternal returns true if the provided package path is internal.
func IsInternal(pkgpath string) bool {
	switch {
	case strings.HasSuffix(pkgpath, "/internal"):
		return true
	case strings.Contains(pkgpath, "/internal/"):
		return true
	case pkgpath == "internal", strings.HasPrefix(pkgpath, "internal/"):
		return true
	}
	return false
}

type Package = packages.Package

// LoadModulePackages all packages in a module.
func LoadModulePackages(mod module.Version) ([]*Package, error) {
	temp, err := tempmod.Create("")
	if err != nil {
		return nil, err
	}
	defer temp.Delete()
	if err := temp.ExecGo("get", "-t", mod.String()); err != nil {
		return nil, err
	}
	cfg := &packages.Config{
		Dir:  temp.Dir,
		Mode: packages.LoadTypes | packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps,
	}
	pkgs, err := packages.Load(cfg, fmt.Sprintf("%s...", mod.Path))
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("found no packages for %s", mod)
	}
	for _, pkg := range pkgs {
		if len(pkg.Errors) != 0 {
			return nil, pkg.Errors[0]
		}
	}
	if err := temp.Delete(); err != nil {
		return nil, err
	}
	return pkgs, nil
}

// LoadPackage loads a single package
func LoadPackage(dir, pkgpath string) (*Package, error) {
	cfg := &packages.Config{
		Dir:  dir,
		Mode: packages.LoadTypes | packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps,
	}
	pkgs, err := packages.Load(cfg, pkgpath)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("found no packages for %s", pkgpath)
	}
	if len(pkgs[0].Errors) != 0 {
		return nil, pkgs[0].Errors[0]
	}
	return pkgs[0], nil
}
