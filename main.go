package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/icholy/gomajor/internal/importpaths"
	"github.com/icholy/gomajor/internal/latest"
	"github.com/icholy/gomajor/internal/packages"
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
	var pre bool
	fset := flag.NewFlagSet("list", flag.ExitOnError)
	fset.BoolVar(&pre, "pre", false, "allow non-v0 prerelease versions")
	fset.StringVar(&dir, "dir", ".", "working directory")
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
		v, err := latest.Version(pkg.ModPath(), pre)
		if err != nil {
			// if the module root is not a package, no versions will be returned.
			// we fallback to trying to get newer module versions of the full package path.
			// if the newer major version doesn't contain the package subdirectory, no versions will be returned.
			v, err = latest.Version(pkg.Path(), pre)
			if err != nil {
				fmt.Printf("%s: failed: %v\n", pkg.ModPath(), err)
				continue
			}
		}
		if semver.Compare(v, pkg.Version) > 0 {
			fmt.Printf("%s: %s [latest %v]\n", pkg.ModPath(), pkg.Version, v)
		}
	}
	return nil
}

func get(args []string) error {
	var dir string
	var rewrite, goget, pre bool
	fset := flag.NewFlagSet("get", flag.ExitOnError)
	fset.BoolVar(&pre, "pre", false, "allow non-v0 prerelease versions")
	fset.BoolVar(&rewrite, "rewrite", true, "rewrite import paths")
	fset.BoolVar(&goget, "get", true, "run go get")
	fset.StringVar(&dir, "dir", ".", "working directory")
	fset.Parse(args)
	if fset.NArg() != 1 {
		return fmt.Errorf("missing package spec")
	}
	// figure out the correct import path
	pkgpath, version := packages.SplitSpec(fset.Arg(0))
	pkg, err := packages.Load(pkgpath)
	if err != nil {
		return err
	}
	// figure out what version to get
	if version == "latest" {
		version, err = latest.Version(pkg.Path(), pre)
		if err != nil {
			return err
		}
	}
	if version != "" && !semver.IsValid(version) {
		return fmt.Errorf("invalid version: %s", version)
	}
	// go get
	if goget {
		spec := pkg.Path()
		if version != "" {
			spec = pkg.PathWithVersion(version) + "@" + semver.Canonical(version)
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
		modpath, ok := pkg.FindModPath(path)
		if !ok {
			return "", importpaths.ErrSkip
		}
		pkgdir := strings.TrimPrefix(path, modpath)
		pkgdir = strings.TrimPrefix(pkgdir, "/")
		if pkg.PkgDir != "" && pkg.PkgDir != pkgdir {
			return "", importpaths.ErrSkip
		}
		newpath := packages.Package{
			Version:   version,
			PkgDir:    pkgdir,
			ModPrefix: pkg.ModPrefix,
		}.Path()
		if newpath == path {
			return "", importpaths.ErrSkip
		}
		fmt.Printf("%s: %s -> %s\n", name, path, newpath)
		return newpath, nil
	})
}
