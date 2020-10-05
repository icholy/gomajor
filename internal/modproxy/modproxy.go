package modproxy

import (
	"bufio"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// DisableModuleFetch sets the Disable-Module-Fetch header to true
var DisableModuleFetch bool

// Module contains the module path and versions
type Module struct {
	Path     string
	Versions []string
}

// Latest returns the latest version.
// If there are no versions, the empty string is returned.
// If pre is false, non-v0 pre-release versions will are excluded.
func (m *Module) Latest(pre bool) string {
	var max string
	for _, v := range m.Versions {
		if !semver.IsValid(v) {
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

// NextMajorPath returns the module path of the next major version
func (m *Module) NextMajorPath() (string, bool) {
	latest := m.Latest(true)
	if latest == "" {
		return "", false
	}
	prefix, _, ok := module.SplitPathVersion(m.Path)
	if !ok {
		return "", false
	}
	if semver.Major(latest) == "v0" {
		return "", false
	}
	major, err := strconv.Atoi(strings.TrimPrefix(semver.Major(latest), "v"))
	if err != nil {
		return "", false
	}
	major++
	return fmt.Sprintf("%s/v%d", prefix, major), true
}

// Query the module proxy for all versions of a module.
// If the module does not exist, the second return parameter will be false
// cached sets the Disable-Module-Fetch: true header
func Query(modpath string, cached bool) (*Module, bool, error) {
	escaped, err := module.EscapePath(modpath)
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
		if res.StatusCode == http.StatusGone {
			// version does not exist
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("proxy request failed: %s", res.Status)
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

// ForPackage tries to find the module path for the provided package path
func ForPackage(pkgpath string, cached bool) (*Module, error) {
	prefix := pkgpath
	for prefix != "" {
		mod, ok, err := Query(prefix, cached)
		if err != nil {
			return nil, err
		}
		if ok {
			return mod, nil
		}
		remaining, last := path.Split(prefix)
		if last == "" {
			break
		}
		prefix = strings.TrimSuffix(remaining, "/")
	}
	return nil, fmt.Errorf("failed to find module for package: %s", pkgpath)
}
