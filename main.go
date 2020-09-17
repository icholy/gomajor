package main

import (
	"flag"
	"log"

	"github.com/sanity-io/litter"

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
	litter.Dump(pkg)
}
