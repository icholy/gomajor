package modproxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"

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
		max = MaxVersion(v, max)
	}
	return max
}

// IsNewerVersion returns true if newversion is greater than oldversion in terms of semver.
// If major is true, then newversion must be a major version ahead of oldversion to be considered newer.
func IsNewerVersion(oldversion, newversion string, major bool) bool {
	if major {
		return semver.Compare(semver.Major(oldversion), semver.Major(newversion)) < 0
	}
	return semver.Compare(oldversion, newversion) < 0
}

// MaxVersion returns the larger of two versions according to semantic version precedence.
// Incompatible versions are considered lower than non-incompatible ones.
// Invalid versions are considered lower than valid ones.
// If both versions are invalid, the empty string is returned.
func MaxVersion(v, w string) string {
	// sort by validity
	vValid := semver.IsValid(v)
	wValid := semver.IsValid(w)
	if !vValid && !wValid {
		return ""
	}
	if vValid != wValid {
		if vValid {
			return v
		}
		return w
	}
	// sort by compatibility
	vIncompatible := strings.HasSuffix(semver.Build(v), "+incompatible")
	wIncompatible := strings.HasSuffix(semver.Build(w), "+incompatible")
	if vIncompatible != wIncompatible {
		if wIncompatible {
			return v
		}
		return w
	}
	// sort by semver
	if semver.Compare(v, w) == 1 {
		return v
	}
	return w
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
	req.Header.Set("User-Agent", "GoMajor/1.0")
	if cached {
		req.Header.Set("Disable-Module-Fetch", "true")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode == http.StatusNotFound && bytes.HasPrefix(body, []byte("not found:")) {
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
// pre controls whether to return modules which only contain pre-release versions.
func Latest(modpath string, cached, pre bool) (*Module, error) {
	mods, err := List(modpath, cached)
	if err != nil {
		return nil, err
	}
	for i := len(mods); i > 0; i-- {
		mod := mods[i-1]
		if max := mod.MaxVersion("", pre); max != "" {
			return mod, nil
		}
	}
	return nil, fmt.Errorf("no module versions found")
}

// List finds all the major versions of a module
// cached sets the Disable-Module-Fetch: true header
func List(modpath string, cached bool) ([]*Module, error) {
	latest, ok, err := Query(modpath, cached)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("module not found: %s", modpath)
	}
	history := []*Module{latest}
	for i := 0; i < 100; i++ {
		nextpath, ok := latest.NextMajorPath()
		if !ok {
			return history, nil
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
			return history, nil
		}
		latest = next
		history = append(history, latest)
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

// Update reports a newer version of a module.
// The Err field will be set if an error occured.
type Update struct {
	Module module.Version
	Latest module.Version
	Err    error `json:",omitempty"`
}

// UpdateOptions specifies a set of modules to check for updates.
// The OnUpdate callback will be invoked with any updates found.
type UpdateOptions struct {
	Pre      bool
	Cached   bool
	Major    bool
	Modules  []module.Version
	OnUpdate func(Update)
}

// Updates finds updates for a set of specified modules.
func Updates(opt UpdateOptions) {
	ch := make(chan Update)
	go func() {
		defer close(ch)
		private := os.Getenv("GOPRIVATE")
		var group errgroup.Group
		if opt.Cached {
			group.SetLimit(3)
		} else {
			group.SetLimit(1)
		}
		for _, m := range opt.Modules {
			m := m
			if module.MatchPrefixPatterns(private, m.Path) {
				continue
			}
			group.Go(func() error {
				mod, err := Latest(m.Path, opt.Cached, opt.Pre)
				if err != nil {
					ch <- Update{Module: m, Err: err}
					return nil
				}
				v := mod.MaxVersion("", opt.Pre)
				if IsNewerVersion(m.Version, v, opt.Major) {
					ch <- Update{
						Module: m,
						Latest: module.Version{
							Path:    mod.WithMajorPath(v),
							Version: v,
						},
					}
				}
				return nil
			})
		}
		group.Wait()
	}()
	for u := range ch {
		if opt.OnUpdate != nil {
			opt.OnUpdate(u)
		}
	}
}
