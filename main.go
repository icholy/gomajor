package main

import (
	"flag"
	"fmt"
	"log"
	"path"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/tools/go/packages"
)

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

func WithPathMajor(path, major string) string {
	// remove the existing version if there is one
	if prefix, _, ok := module.SplitPathVersion(path); ok {
		path = prefix
	}
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

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("missing package spec")
	}
	// split the package package and version
	spec := flag.Arg(0)
	pkgpath, version := SplitSpec(spec)
	// find the module root
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedModule,
	}
	pkgs, err := packages.Load(cfg, pkgpath)
	if err != nil {
		log.Fatal(err)
	}
	pkg := pkgs[0]
	// find the module path for the specified version
	modpath := WithPathMajor(pkg.Module.Path, semver.Major(version))
	fmt.Println(
		path.Join(modpath, strings.TrimPrefix(pkg.PkgPath, pkg.Module.Path)),
	)
}
