package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

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
	dir, err := TempModDir()
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)
	// find the module root
	cfg := &packages.Config{
		Mode:       packages.NeedName | packages.NeedModule,
		BuildFlags: []string{"-mod=readonly"},
		Dir:        dir,
	}
	pkgs, err := packages.Load(cfg, pkgpath)
	if err != nil {
		log.Fatal("failed to load package:", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}
	if len(pkgs) == 0 || pkgs[0].Module == nil {
		log.Fatalf("failed to find module: %s", pkgpath)
	}
	pkg := pkgs[0]
	// find the module path for the specified version
	modpath := WithPathMajor(pkg.Module.Path, semver.Major(version))
	fmt.Println(
		path.Join(modpath, strings.TrimPrefix(pkg.PkgPath, pkg.Module.Path)),
	)
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

func TempModDir() (string, error) {
	dir, err := ioutil.TempDir("", "gomajor_*")
	if err != nil {
		return "", err
	}
	cmd := exec.Command("go", "mod", "init", "temp")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(dir) // best effort
		return "", err
	}
	return dir, nil
}
