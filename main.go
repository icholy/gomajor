package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"golang.org/x/mod/semver"

	"github.com/icholy/gomajor/internal/importpaths"
	"github.com/icholy/gomajor/internal/modproxy"
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
	for _, dep := range direct {
		modprefix := packages.ModPrefix(dep.Path)
		if seen[modprefix] {
			continue
		}
		seen[modprefix] = true
		mod, err := modproxy.Latest(dep.Path, cached)
		if err != nil {
			fmt.Printf("%s: failed: %v\n", mod.Path, err)
			continue
		}
		v := mod.MaxVersion(pre)
		if major && semver.Compare(semver.Major(v), semver.Major(dep.Version)) <= 0 {
			continue
		}
		if semver.Compare(v, dep.Version) <= 0 {
			continue
		}
		fmt.Printf("%s: %s [latest %v]\n", dep.Path, dep.Version, v)
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
	pkgpath, target := packages.SplitSpec(fset.Arg(0))
	mod, err := modproxy.QueryPackage(pkgpath, cached)
	if err != nil {
		return err
	}
	modprefix := packages.ModPrefix(mod.Path)
	// try infer the target from SIV
	if target == "" {
		if modpath, _, ok := packages.SplitPath(modprefix, pkgpath); ok && modpath != mod.Path {
			if major, ok := packages.ModMajor(modpath); ok {
				target = major
			}
		}
	}
	// figure out what version to get
	var version string
	switch target {
	case "":
		version = mod.MaxVersion(pre)
	case "latest":
		latest, err := modproxy.Latest(mod.Path, cached)
		if err != nil {
			return err
		}
		version = latest.MaxVersion(pre)
		target = version
	case "master", "default":
		latest, err := modproxy.Latest(mod.Path, cached)
		if err != nil {
			return err
		}
		version = latest.MaxVersion(pre)
	default:
		if !semver.IsValid(target) {
			return fmt.Errorf("invalid version: %s", target)
		}
		// best effort to detect +incompatible versions
		if v := mod.BestMatch(target); v != "" {
			version = v
		} else {
			version = target
		}
	}
	// split up the path
	_, pkgdir, _ := packages.SplitPath(modprefix, pkgpath)
	// go get
	if goget {
		spec := packages.JoinPath(modprefix, version, pkgdir)
		if target != "" {
			spec += "@" + target
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
		_, pkgdir0, ok := packages.SplitPath(modprefix, path)
		if !ok {
			return "", importpaths.ErrSkip
		}
		if pkgdir != "" && pkgdir != pkgdir0 {
			return "", importpaths.ErrSkip
		}
		newpath := packages.JoinPath(modprefix, version, pkgdir0)
		if newpath == path {
			return "", importpaths.ErrSkip
		}
		fmt.Printf("%s: %s -> %s\n", name, path, newpath)
		return newpath, nil
	})
}
