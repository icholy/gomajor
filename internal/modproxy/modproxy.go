package modproxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"

	"github.com/icholy/gomajor/internal/packages"
)

// Module contains the module path and versions
type Module struct {
	Path     string
	Versions []string
}

// MaxVersion returns the latest version.
// If there are no versions, the empty string is returned.
// Prefix can be used to filter the versions based on a prefix.
// If pre is false, non-v0 pre-release versions will are excluded.
func (m *Module) MaxVersion(prefix string, pre bool) string {
	var max string
	for _, v := range m.Versions {
		if !semver.IsValid(v) || !strings.HasPrefix(v, prefix) {
			continue
		}
		if !pre && semver.Major(v) != "v0" && semver.Prerelease(v) != "" {
			continue
		}
		if max == "" {
			max = v
		}
		if semver.Compare(v, max) == 1 {
			max = v
		}
	}
	return max
}

// NextMajor returns the next major version after the provided version
func NextMajor(version string) (string, error) {
	major, err := strconv.Atoi(strings.TrimPrefix(semver.Major(version), "v"))
	if err != nil {
		return "", err
	}
	major++
	return fmt.Sprintf("v%d", major), nil
}

// NextMajorPath returns the module path of the next major version
func (m *Module) NextMajorPath() (string, bool) {
	latest := m.MaxVersion("", true)
	if latest == "" {
		return "", false
	}
	if semver.Major(latest) == "v0" {
		return "", false
	}
	next, err := NextMajor(latest)
	if err != nil {
		return "", false
	}
	prefix := packages.ModPrefix(m.Path)
	return packages.JoinPath(prefix, next, ""), true
}

// Query the module proxy for all versions of a module.
// If the module does not exist, the second return parameter will be false
// cached sets the Disable-Module-Fetch: true header
func Query(modpath string, cached bool) (*Module, bool, error) {
	escaped, err := module.EscapePath(modpath)
	if err != nil {
		return nil, false, err
	}
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/list", escaped)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	if cached {
		req.Header.Set("Disable-Module-Fetch", "true")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(res.Body)
		if res.StatusCode == http.StatusGone && bytes.HasPrefix(body, []byte("not found:")) {
			return nil, false, nil
		}
		msg := string(body)
		if msg == "" {
			msg = res.Status
		}
		return nil, false, fmt.Errorf("proxy: %s", msg)
	}
	var mod Module
	mod.Path = modpath
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		mod.Versions = append(mod.Versions, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, false, err
	}
	return &mod, true, nil
}

// Latest finds the latest major version of a module
// cached sets the Disable-Module-Fetch: true header
func Latest(modpath string, cached bool) (*Module, error) {
	latest, ok, err := Query(modpath, cached)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("module not found: %s", modpath)
	}
	for i := 0; i < 100; i++ {
		nextpath, ok := latest.NextMajorPath()
		if !ok {
			return latest, nil
		}
		next, ok, err := Query(nextpath, cached)
		if err != nil {
			return nil, err
		}
		if !ok {
			return latest, nil
		}
		latest = next
	}
	return nil, fmt.Errorf("request limit exceeded")
}

// QueryPackage tries to find the module path for the provided package path
// it does so by repeatedly chopping off the last path element and trying to
// use it as a path.
func QueryPackage(pkgpath string, cached bool) (*Module, error) {
	prefix := pkgpath
	for prefix != "" {
		if module.CheckPath(prefix) == nil {
			mod, ok, err := Query(prefix, cached)
			if err != nil {
				return nil, err
			}
			if ok {
				modprefix := packages.ModPrefix(mod.Path)
				if modpath, pkgdir, ok := packages.SplitPath(modprefix, pkgpath); ok && modpath != mod.Path {
					if version, ok := packages.ModMajor(modpath); ok {
						if v := mod.MaxVersion(version, false); v != "" {
							spec := packages.JoinPath(modprefix, "", pkgdir) + "@" + v
							return nil, fmt.Errorf("module not found: %s; found %s", modpath, spec)
						}
						return nil, fmt.Errorf("module not found: %s", pkgpath)
					}
				}
				return mod, nil
			}
		}
		remaining, last := path.Split(prefix)
		if last == "" {
			break
		}
		prefix = strings.TrimSuffix(remaining, "/")
	}
	return nil, fmt.Errorf("failed to find module for package: %s", pkgpath)
}
