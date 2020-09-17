package latest

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/andybalholm/cascadia"
	"github.com/hashicorp/go-version"
	"golang.org/x/net/html"
)

// Version returns the latest version of the modpath
func Version(modpath string) (string, error) {
	vv, err := versions(modpath)
	if err != nil {
		return "", err
	}
	return highest(vv)
}

// parse all the versions from the pkg.go.dev versions tab
func parse(r io.Reader) ([]string, error) {
	var versions []string
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	sel := cascadia.MustCompile(".Versions-item>a")
	for _, node := range cascadia.QueryAll(doc, sel) {
		walk(node, func(n *html.Node) {
			if n.Type == html.TextNode {
				versions = append(versions, n.Data)
			}
		})
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

// get all the versions for a module
func versions(modpath string) ([]string, error) {
	url := fmt.Sprintf("https://pkg.go.dev/%s?tab=versions", modpath)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", res.Status)
	}
	return parse(res.Body)
}

// highest returns the highest semver version string
func highest(versions []string) (string, error) {
	if len(versions) == 0 {
		return "", errors.New("no versions")
	}
	var newest string
	var newestVer *version.Version
	for _, s := range versions {
		v, err := version.NewVersion(s)
		if err != nil {
			return "", err
		}
		if newestVer == nil || v.GreaterThan(newestVer) {
			newestVer = v
			newest = s
		}
	}
	return newest, nil
}
