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
		if !strings.HasPrefix(path, pkg.ModPrefix) {
			return "", false
		}
		return "", false
	})
	if err != nil {
		log.Fatal(err)
	}
}
