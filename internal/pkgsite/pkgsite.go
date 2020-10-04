package pkgsite

import (
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/andybalholm/cascadia"
	"golang.org/x/mod/semver"
	"golang.org/x/net/html"
)

// ErrNoVersions is returned when no versions are found for a package path
var ErrNoVersions = errors.New("no versions found")

// Latest returns the latest version of the package
// If pre is false, non-v0 pre-release versions are omitted
func Latest(pkgpath string, pre bool) (string, error) {
	versions, err := Versions(pkgpath)
	if err != nil {
		return "", err
	}
	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) > 0
	})
	for _, v := range versions {
		if pre || semver.Major(v) == "v0" || semver.Prerelease(v) == "" {
			return v, nil
		}
	}
	return "", ErrNoVersions
}

// Versions returns all versions of a package
func Versions(pkgpath string) ([]string, error) {
	url := fmt.Sprintf("https://pkg.go.dev/%s?tab=versions", pkgpath)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New(res.Status)
	}
	// extract versions from the html
	var versions []string
	doc, err := html.Parse(res.Body)
	if err != nil {
		return nil, err
	}
	sel := cascadia.MustCompile(".Versions-item>a")
	for _, node := range cascadia.QueryAll(doc, sel) {
		walk(node, func(n *html.Node) {
			if n.Type == html.TextNode {
				if v := n.Data; semver.IsValid(v) {
					versions = append(versions, v)
				}
			}
		})
	}
	if len(versions) == 0 {
		return nil, ErrNoVersions
	}
	return versions, nil
}

// walk the node using depth first
func walk(n *html.Node, f func(*html.Node)) {
	f(n)
	if n.FirstChild != nil {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, f)
		}
	}
}
