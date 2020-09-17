package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sanity-io/litter"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/tools/go/packages"
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("missing package spec")
	}
	// split the package package and version
	spec := flag.Arg(0)
	pkgpath, version := SplitSpec(spec)
	// create a temporary module
	pkg, err := PackageWithVersion(pkgpath, version)
	if err != nil {
		log.Fatal(err)
	}
	litter.Dump(pkg)
}

type Package struct {
	Version   string
	PkgPath   string
	PkgDir    string
	ModPath   string
	ModPathV1 string
}

func PackageWithVersion(pkgpath string, version string) (*Package, error) {
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
		return nil, fmt.Errorf("failed to file module: %s", pkgpath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, pkg.Errors[0]
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}
	// remove the existing version if there is one
	modpathV1 := pkg.Module.Path
	if prefix, _, ok := module.SplitPathVersion(modpathV1); ok {
		modpathV1 = prefix
	}
	pkgdir := strings.TrimPrefix(pkg.PkgPath, pkg.Module.Path)
	pkgdir = strings.TrimPrefix(pkgdir, "/")
	// find the module path for the specified version
	modpath := JoinPathMajor(modpathV1, semver.Major(version))
	return &Package{
		Version:   version,
		PkgPath:   path.Join(modpath, pkgdir),
		PkgDir:    pkgdir,
		ModPath:   modpath,
		ModPathV1: modpathV1,
	}, nil
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
