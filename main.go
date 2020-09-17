package main

import (
	"flag"
	"log"
	"strings"

	"github.com/icholy/gomajor/importpaths"
	"github.com/icholy/gomajor/packages"
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("missing package spec")
	}
	pkgpath, version := packages.SplitSpec(flag.Arg(0))
	pkg, err := packages.Load(pkgpath)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(pkg.Path(version))
	err = importpaths.Rewrite(".", func(name, path string) (string, bool) {
		modpath, ok := pkg.FindModPath(path)
		if !ok {
			return "", false
		}
		pkgdir := strings.TrimPrefix(path, modpath)
		pkgdir = strings.TrimPrefix(pkgdir, "/")
		if pkg.PkgDir != "" && pkg.PkgDir != pkgdir {
			return "", false
		}
		return packages.Package{
			PkgDir:    pkgdir,
			ModPrefix: pkg.ModPrefix,
		}.Path(version), true
	})
	if err != nil {
		log.Fatal(err)
	}
}
