package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/icholy/gomajor/internal/importpaths"
	"github.com/icholy/gomajor/internal/modproxy"
	"github.com/icholy/gomajor/internal/packages"
	"golang.org/x/mod/semver"
)

var help = `
GoMajor is an experimental tool for major version upgrades

Usage:

    gomajor <command> [arguments]

The commands are:

    get     upgrade to a major version
    list    list available updates
    help    show this help text
`

func main() {
	flag.Usage = func() {
		fmt.Println(help)
	}
	flag.Parse()
	switch flag.Arg(0) {
	case "get":
		if err := get(flag.Args()[1:]); err != nil {
			log.Fatal(err)
		}
	case "list":
		if err := list(flag.Args()[1:]); err != nil {
			log.Fatal(err)
		}
	case "help", "":
		flag.Usage()
	default:
		fmt.Printf("unrecognized subcommand: %s\n", flag.Arg(0))
		flag.Usage()
	}
}

func list(args []string) error {
	var dir string
	var pre, cached, major bool
	fset := flag.NewFlagSet("list", flag.ExitOnError)
	fset.BoolVar(&pre, "pre", false, "allow non-v0 prerelease versions")
	fset.StringVar(&dir, "dir", ".", "working directory")
	fset.BoolVar(&cached, "cached", true, "only fetch cached content from the module proxy")
	fset.BoolVar(&major, "major", false, "only show newer major versions")
	fset.Parse(args)
	direct, err := packages.Direct(dir)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, pkg := range direct {
		if seen[pkg.ModPrefix] {
			continue
		}
		seen[pkg.ModPrefix] = true
		mod, err := modproxy.Latest(pkg.ModPath(), cached)
		if err != nil {
			fmt.Printf("%s: failed: %v\n", pkg.ModPath(), err)
			continue
		}
		v := mod.Latest(pre)
		if major && semver.Compare(semver.Major(v), semver.Major(pkg.Version)) <= 0 {
			continue
		}
		if semver.Compare(v, pkg.Version) <= 0 {
			continue
		}
		fmt.Printf("%s: %s [latest %v]\n", pkg.ModPath(), pkg.Version, v)
	}
	return nil
}

func get(args []string) error {
	var dir string
	var rewrite, goget, pre, cached bool
	fset := flag.NewFlagSet("get", flag.ExitOnError)
	fset.BoolVar(&pre, "pre", false, "allow non-v0 prerelease versions")
	fset.BoolVar(&rewrite, "rewrite", true, "rewrite import paths")
	fset.BoolVar(&goget, "get", true, "run go get")
	fset.StringVar(&dir, "dir", ".", "working directory")
	fset.BoolVar(&cached, "cached", true, "only fetch cached content from the module proxy")
	fset.Parse(args)
	if fset.NArg() != 1 {
		return fmt.Errorf("missing package spec")
	}
	// figure out the correct import path
	pkgpath, version := packages.SplitSpec(fset.Arg(0))
	pkg, err := modproxy.Package(pkgpath, pre, cached)
	if err != nil {
		return err
	}
	// figure out what version to get
	switch version {
	case "":
		mod, ok, err := modproxy.Query(pkg.ModPath(), cached)
		if err != nil {
			return err
		}
		if ok {
			version = mod.Latest(pre)
		}
	case "latest":
		mod, err := modproxy.Latest(pkg.ModPath(), cached)
		if err != nil {
			return err
		}
		version = mod.Latest(pre)
		pkg.Version = version
	case "master":
		mod, err := modproxy.Latest(pkg.ModPath(), cached)
		if err != nil {
			return err
		}
		pkg.Version = mod.Latest(pre)
	default:
		if !semver.IsValid(version) {
			return fmt.Errorf("invalid version: %s", version)
		}
		pkg.Version = version
	}
	// go get
	if goget {
		spec := pkg.Path()
		if version != "" {
			spec += "@" + version
		}
		fmt.Println("go get", spec)
		cmd := exec.Command("go", "get", spec)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	// rewrite imports
	if !rewrite {
		return nil
	}
	return importpaths.Rewrite(dir, func(name, path string) (string, error) {
		_, pkgdir, ok := packages.SplitPath(pkg.ModPrefix, path)
		if !ok {
			return "", importpaths.ErrSkip
		}
		if pkg.PkgDir != "" && pkg.PkgDir != pkgdir {
			return "", importpaths.ErrSkip
		}
		newpath := packages.JoinPath(pkg.ModPrefix, pkg.Version, pkgdir)
		if newpath == path {
			return "", importpaths.ErrSkip
		}
		fmt.Printf("%s: %s -> %s\n", name, path, newpath)
		return newpath, nil
	})
}
