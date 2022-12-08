package importpaths

import (
	"fmt"
	"go/token"

	"github.com/icholy/gomajor/internal/packages"
)

func RewriteModuleVersion(dir, modpath, pkgpath, version string) error {
	modprefix := packages.ModPrefix(modpath)
	_, pkgdir, _ := packages.SplitPath(modprefix, pkgpath)
	return Rewrite(dir, func(pos token.Position, path string) (string, error) {
		_, pkgdir0, ok := packages.SplitPath(modprefix, path)
		if !ok {
			return "", ErrSkip
		}
		if pkgdir != "" && pkgdir != pkgdir0 {
			return "", ErrSkip
		}
		newpath := packages.JoinPath(modprefix, version, pkgdir0)
		if newpath == path {
			return "", ErrSkip
		}
		fmt.Printf("%s %s\n", pos, newpath)
		return newpath, nil
	})
}
