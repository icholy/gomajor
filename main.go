package main

import (
	"flag"
	"fmt"
	"go/token"
	"log"
	"os"
	"os/exec"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"

	"github.com/icholy/gomajor/internal/importpaths"
	"github.com/icholy/gomajor/internal/modproxy"
	"github.com/icholy/gomajor/internal/modupdates"
	"github.com/icholy/gomajor/internal/packages"
)

var help = `
GoMajor is a tool for major version upgrades

Usage:

    gomajor <command> [arguments]

The commands are:

    get     upgrade to a major version
    list    list available updates
    path    modify the module path
    help    show this help text
`

func main() {
	flag.Usage = func() {
		fmt.Println(help)
	}
	flag.Parse()
	var cmd func([]string) error
	switch flag.Arg(0) {
	case "get":
		cmd = getcmd
	case "list":
		cmd = listcmd
	case "update":
		cmd = updatecmd
	case "path":
		cmd = pathcmd
	case "help", "":
		flag.Usage()
	default:
		fmt.Fprintf(os.Stderr, "unrecognized subcommand: %s\n", flag.Arg(0))
		flag.Usage()
	}
	if err := cmd(flag.Args()[1:]); err != nil {
		log.Fatal(err)
	}
}

func listcmd(args []string) error {
	var dir string
	var opt modupdates.Options
	fset := flag.NewFlagSet("list", flag.ExitOnError)
	fset.StringVar(&dir, "dir", ".", "working directory")
	fset.BoolVar(&opt.Pre, "pre", false, "allow non-v0 prerelease versions")
	fset.BoolVar(&opt.Cached, "cached", true, "only fetch cached content from the module proxy")
	fset.BoolVar(&opt.Major, "major", false, "only show newer major versions")
	fset.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gomajor list")
		fset.PrintDefaults()
	}
	fset.Parse(args)
	dependencies, err := packages.Direct(dir)
	if err != nil {
		return err
	}
	opt.Modules = dependencies
	opt.OnErr = func(m module.Version, err error) {
		fmt.Fprintf(os.Stderr, "%s: failed: %v\n", m.Path, err)
	}
	for u := range modupdates.List(opt) {
		fmt.Printf("%s: %s [latest %v]\n", u.Module.Path, u.Module.Version, u.Latest)
	}
	return nil
}

func updatecmd(args []string) error {
	var dir string
	var opt modupdates.Options
	fset := flag.NewFlagSet("update", flag.ExitOnError)
	fset.StringVar(&dir, "dir", ".", "working directory")
	fset.BoolVar(&opt.Pre, "pre", false, "allow non-v0 prerelease versions")
	fset.BoolVar(&opt.Cached, "cached", true, "only fetch cached content from the module proxy")
	fset.BoolVar(&opt.Major, "major", false, "only show newer major versions")
	fset.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gomajor update")
		fset.PrintDefaults()
	}
	fset.Parse(args)
	dependencies, err := packages.Direct(dir)
	if err != nil {
		return err
	}
	opt.Modules = dependencies
	opt.OnErr = func(m module.Version, err error) {
		fmt.Fprintf(os.Stderr, "%s: failed: %v\n", m.Path, err)
	}
	for u := range modupdates.List(opt) {
		modprefix := packages.ModPrefix(u.Module.Path)
		spec := fmt.Sprintf("%s@%s", packages.JoinPath(modprefix, u.Latest, ""), u.Latest)
		if err := GoGet(dir, spec); err != nil {
			continue
		}
		if err := RewriteModule(dir, u.Module.Path, "", u.Latest); err != nil {
			fmt.Fprintf(os.Stderr, "%s: rewrite: %v\n", u.Module.Path, err)
			continue
		}
	}
	return nil
}

func getcmd(args []string) error {
	var dir string
	var rewrite, goget, pre, cached bool
	fset := flag.NewFlagSet("get", flag.ExitOnError)
	fset.BoolVar(&pre, "pre", false, "allow non-v0 prerelease versions")
	fset.BoolVar(&rewrite, "rewrite", true, "rewrite import paths")
	fset.BoolVar(&goget, "get", true, "run go get")
	fset.StringVar(&dir, "dir", ".", "working directory")
	fset.BoolVar(&cached, "cached", true, "only fetch cached content from the module proxy")
	fset.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gomajor get <pathspec>")
		fset.PrintDefaults()
	}
	fset.Parse(args)
	if fset.NArg() != 1 {
		return fmt.Errorf("missing package spec")
	}
	// split the package spec into its components
	pkgpath, query := packages.SplitSpec(fset.Arg(0))
	mod, err := modproxy.QueryPackage(pkgpath, cached)
	if err != nil {
		return err
	}
	// figure out what version to get
	var version string
	switch query {
	case "":
		version = mod.MaxVersion("", pre)
	case "latest":
		latest, err := modproxy.Latest(mod.Path, cached)
		if err != nil {
			return err
		}
		version = latest.MaxVersion("", pre)
		query = version
	default:
		if !semver.IsValid(query) {
			return fmt.Errorf("invalid version: %s", query)
		}
		// best effort to detect +incompatible versions
		if v := mod.MaxVersion(query, pre); v != "" {
			version = v
		} else {
			version = query
		}
	}
	// go get
	if goget {
		modprefix := packages.ModPrefix(mod.Path)
		_, pkgdir, _ := packages.SplitPath(modprefix, pkgpath)
		spec := packages.JoinPath(modprefix, version, pkgdir)
		if query != "" {
			spec += "@" + query
		}
		if err := GoGet(dir, spec); err != nil {
			return err
		}
	}
	// rewrite imports
	if !rewrite {
		return nil
	}
	return RewriteModule(dir, mod.Path, pkgpath, version)
}

func GoGet(dir, spec string) error {
	fmt.Println("go get", spec)
	cmd := exec.Command("go", "get", spec)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func RewriteModule(dir, modpath, pkgpath, version string) error {
	modprefix := packages.ModPrefix(modpath)
	_, pkgdir, _ := packages.SplitPath(modprefix, pkgpath)
	return importpaths.Rewrite(dir, func(pos token.Position, path string) (string, error) {
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
		fmt.Printf("%s %s\n", pos, newpath)
		return newpath, nil
	})
}

func pathcmd(args []string) error {
	var dir, version string
	var next, rewrite bool
	fset := flag.NewFlagSet("path", flag.ExitOnError)
	fset.BoolVar(&next, "next", false, "increment the module path version")
	fset.StringVar(&version, "version", "", "set the module path version")
	fset.BoolVar(&rewrite, "rewrite", true, "rewrite import paths")
	fset.StringVar(&dir, "dir", ".", "working directory")
	fset.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gomajor path [modpath]")
		fset.PrintDefaults()
	}
	fset.Parse(args)
	// find and parse go.mod
	name, err := packages.FindModFile(dir)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	file, err := modfile.ParseLax(name, data, nil)
	if err != nil {
		return err
	}
	// figure out the new module path
	modpath := fset.Arg(0)
	if modpath == "" {
		modpath = file.Module.Mod.Path
	}
	// find the current version if one wasn't provided
	if version == "" {
		var ok bool
		version, ok = packages.ModMajor(modpath)
		if !ok || version == "" {
			version = "v1"
		}
	}
	// increment the path version
	if next {
		version, err = modproxy.NextMajor(version)
		if err != nil {
			return err
		}
	}
	if !semver.IsValid(version) {
		return fmt.Errorf("invalid version: %q", version)
	}
	// create the new modpath
	modprefix := packages.ModPrefix(modpath)
	oldmodprefix := packages.ModPrefix(file.Module.Mod.Path)
	modpath = packages.JoinPath(modprefix, version, "")
	fmt.Printf("module %s\n", modpath)
	if !rewrite {
		return nil
	}
	// update go.mod
	cmd := exec.Command("go", "mod", "edit", "-module", modpath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	// rewrite import paths
	return importpaths.Rewrite(dir, func(pos token.Position, path string) (string, error) {
		_, pkgdir, ok := packages.SplitPath(oldmodprefix, path)
		if !ok {
			return "", importpaths.ErrSkip
		}
		newpath := packages.JoinPath(modprefix, version, pkgdir)
		if newpath == path {
			return "", importpaths.ErrSkip
		}
		fmt.Printf("%s %s\n", pos, newpath)
		return newpath, nil
	})
}
