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
	pkg, err := packages.PackageWithVersion(pkgpath, version)
	if err != nil {
		log.Fatal(err)
	}
	err = importpaths.Rewrite(".", func(name, path string) (string, bool) {
		if !strings.HasPrefix(path, pkg.ModPathV1) {
			return "", false
		}
		return "", false
	})
	if err != nil {
		log.Fatal(err)
	}
}
