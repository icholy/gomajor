package importpaths

import (
	"go/token"
	"sort"
)

// List returns a sorted list of all package imports
func List(dir string) ([]string, error) {
	pkgpaths := []string{}
	seen := map[string]struct{}{}
	err := Rewrite(dir, func(_ token.Position, path string) (string, error) {
		if _, ok := seen[path]; !ok {
			pkgpaths = append(pkgpaths, path)
			seen[path] = struct{}{}
		}
		return "", ErrSkip
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(pkgpaths, func(i, j int) bool { return pkgpaths[i] < pkgpaths[j] })
	return pkgpaths, nil
}
