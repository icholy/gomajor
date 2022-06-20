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
// If pre is false, pre-release versions will are excluded.
func (m *Module) MaxVersion(prefix string, pre bool) string {
	var max string
	for _, v := range m.Versions {
		if !semver.IsValid(v) || !strings.HasPrefix(v, prefix) {
			continue
		}
		if !pre && semver.Prerelease(v) != "" {
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

// WithMajorPath returns the module path for the provided version
func (m *Module) WithMajorPath(version string) string {
	prefix := packages.ModPrefix(m.Path)
	return packages.JoinPath(prefix, version, "")
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
	return m.WithMajorPath(next), true
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
			// handle the case where a project switched to modules
			// without incrementing the major version
			version := latest.MaxVersion("", true)
			if semver.Build(version) == "+incompatible" {
				nextpath = latest.WithMajorPath(semver.Major(version))
				if nextpath != latest.Path {
					next, ok, err = Query(nextpath, cached)
					if err != nil {
						return nil, err
					}
				}
			}
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
					if major, ok := packages.ModMajor(modpath); ok {
						if v := mod.MaxVersion(major, false); v != "" {
							spec := packages.JoinPath(modprefix, "", pkgdir) + "@" + v
							return nil, fmt.Errorf("%s doesn't support import versioning; use %s", major, spec)
						}
						return nil, fmt.Errorf("failed to find module for package: %s", pkgpath)
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

type Spec struct {
	ModPrefix  string
	Version    string
	PackageDir string
	Query      string
}

func (s Spec) Module() module.Version {
	return module.Version{
		Path: packages.JoinPath(s.ModPrefix, s.Version, ""),
		Version: s.Version,
	}
}

// String formats the spec to a string that can be passed to 'go get'.
func (s Spec) String() string {
	spec := packages.JoinPath(s.ModPrefix, s.Version, s.PackageDir)
	if s.Query != "" {
		spec += "@" + s.Query
	}
	return spec
}

// Resolve a module version given a package@version spec string.
func Resolve(spec string, cached, pre bool) (*Spec, error) {
	// split the package spec into its components
	pkgpath, query := packages.SplitSpec(spec)
	mod, err := QueryPackage(pkgpath, cached)
	if err != nil {
		return nil, err
	}
	// figure out what version to get
	var version string
	switch query {
	case "":
		version = mod.MaxVersion("", pre)
	case "latest":
		latest, err := Latest(mod.Path, cached)
		if err != nil {
			return nil, err
		}
		version = latest.MaxVersion("", pre)
		query = version
	case "master", "default":
		latest, err := Latest(mod.Path, cached)
		if err != nil {
			return nil, err
		}
		version = latest.MaxVersion("", pre)
	default:
		if !semver.IsValid(query) {
			return nil, fmt.Errorf("invalid version: %s", query)
		}
		// best effort to detect +incompatible versions
		if v := mod.MaxVersion(query, pre); v != "" {
			version = v
		} else {
			version = query
		}
	}
	modprefix := packages.ModPrefix(mod.Path)
	_, pkgdir, _ := packages.SplitPath(modprefix, pkgpath)
	return &Spec{
		ModPrefix:  modprefix,
		PackageDir: pkgdir,
		Version:    version,
		Query:      query,
	}, nil
}
