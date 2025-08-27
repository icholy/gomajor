package modproxy

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"

	"github.com/icholy/gomajor/internal/goenv"
	"github.com/icholy/gomajor/internal/packages"
)

// Request sends requests to the module proxies in order and returns
// the first 200 response.
func Request(path string, cached bool) (*http.Response, error) {
	proxies := goenv.GOPROXYURL()
	if len(proxies) == 0 {
		return nil, errors.New("no GOPROXY urls available")
	}
	var last *http.Response
	for _, u := range proxies {
		res, err := doProxyRequest(u, path, cached)
		if err != nil {
			return nil, err
		}
		if res.StatusCode == http.StatusOK {
			return res, nil
		}
		last = res
	}
	return last, nil
}

func doProxyRequest(u *url.URL, subpath string, cached bool) (*http.Response, error) {
	switch u.Scheme {
	case "http", "https":
		return httpRequest(u, subpath, cached)
	case "file":
		return fileRequest(u, subpath)
	default:
		return nil, errors.New("unsupported protocol " + u.Scheme)
	}
}

func httpRequest(u *url.URL, subpath string, cached bool) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, u.JoinPath(subpath).String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GoMajor/1.0")
	if cached {
		req.Header.Set("Disable-Module-Fetch", "true")
	}
	return http.DefaultClient.Do(req)
}

func fileRequest(u *url.URL, subpath string) (*http.Response, error) {
	root := u.Path
	if filepath.VolumeName(root) != "" {
		root, _ = strings.CutPrefix(root, "/")
	}
	f, err := os.Open(filepath.Join(
		filepath.FromSlash(root),
		filepath.FromSlash(subpath),
	))
	if err != nil {
		if os.IsNotExist(err) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     http.StatusText(http.StatusNotFound),
				Body:       http.NoBody,
			}, nil
		}
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	return &http.Response{
		StatusCode:    http.StatusOK,
		Status:        http.StatusText(http.StatusOK),
		Body:          f,
		ContentLength: fi.Size(),
	}, nil
}

// Module contains the module path and versions
type Module struct {
	Path     string
	Versions []string
}

// MaxVersion returns the latest version of the module in the list.
// If pre is false, pre-release versions will are excluded.
// Retracted versions are excluded.
func MaxVersion(mods []*Module, pre bool, r Retractions) (*Module, string) {
	for i := len(mods); i > 0; i-- {
		mod := mods[i-1].Retract(r)
		if max := mod.MaxVersion("", pre); max != "" {
			return mod, max
		}
	}
	return nil, ""
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
		if CompareVersion(max, v) < 0 {
			max = v
		}
	}
	return max
}

// Retract returns a copy of m with the retracted versions removed.
func (m *Module) Retract(r Retractions) *Module {
	versions := slices.Clone(m.Versions)
	return &Module{
		Path:     m.Path,
		Versions: slices.DeleteFunc(versions, r.Includes),
	}
}

// IsNewerVersion returns true if newversion is greater than oldversion in terms of semver.
// If major is true, then newversion must be a major version ahead of oldversion to be considered newer.
func IsNewerVersion(oldversion, newversion string, major bool) bool {
	if major {
		return semver.Compare(semver.Major(oldversion), semver.Major(newversion)) < 0
	}
	return semver.Compare(oldversion, newversion) < 0
}

// CompareVersion returns -1 if v < w, 1 if v > w, and 0 if v == w
// Incompatible versions are considered lower than non-incompatible ones.
// Invalid versions are considered lower than valid ones.
// If both versions are invalid, the empty string is returned.
func CompareVersion(v, w string) int {
	// sort by validity
	vValid := semver.IsValid(v)
	wValid := semver.IsValid(w)
	if !vValid && !wValid {
		return 0
	}
	if vValid != wValid {
		if vValid {
			return 1
		}
		return -1
	}
	// sort by compatibility
	vIncompatible := strings.HasSuffix(semver.Build(v), "+incompatible")
	wIncompatible := strings.HasSuffix(semver.Build(w), "+incompatible")
	if vIncompatible != wIncompatible {
		if wIncompatible {
			return 1
		}
		return -1
	}
	// sort by semver
	return semver.Compare(v, w)
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
	res, err := Request(path.Join(escaped, "@v", "list"), cached)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode == http.StatusNotFound {
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

// ErrNoVersions is returned when the proxy has no version for a module
var ErrNoVersions = errors.New("no module versions found")

// Latest finds the latest major version of a module
// cached sets the Disable-Module-Fetch: true header
// pre controls whether to return modules which only contain pre-release versions.
func Latest(modpath string, cached, pre bool) (*Module, error) {
	mods, err := List(modpath, cached)
	if err != nil {
		return nil, err
	}
	// find the retractions
	var r Retractions
	if mod, _ := MaxVersion(mods, false, nil); mod != nil {
		var err error
		r, err = FetchRetractions(mod)
		if err != nil {
			return nil, err
		}
	}
	mod, _ := MaxVersion(mods, pre, r)
	if mod == nil {
		return nil, ErrNoVersions
	}
	return mod, nil
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

// FetchRetractions fetches the retractions for this module.
func FetchRetractions(mod *Module) (Retractions, error) {
	max := mod.MaxVersion("", false)
	if max == "" {
		return nil, nil
	}
	escaped, err := module.EscapePath(mod.Path)
	if err != nil {
		return nil, err
	}
	res, err := Request(path.Join(escaped, "@v", max+".mod"), false)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		msg := string(body)
		if msg == "" {
			msg = res.Status
		}
		return nil, fmt.Errorf("proxy: %s", msg)
	}
	file, err := modfile.ParseLax(mod.Path, body, nil)
	if err != nil {
		return nil, err
	}
	var retractions Retractions
	for _, r := range file.Retract {
		retractions = append(retractions, VersionRange{Low: r.Low, High: r.High})
	}
	return retractions, nil
}

// VersionRange is an inclusive version range.
type VersionRange struct {
	Low, High string
}

// Includes reports whether v is in the inclusive range
func (r VersionRange) Includes(v string) bool {
	return CompareVersion(v, r.Low) >= 0 && CompareVersion(v, r.High) <= 0
}

// Retractions is a list of retracted versions.
type Retractions []VersionRange

// Includes reports whether v is retracted
func (rr Retractions) Includes(v string) bool {
	for _, r := range rr {
		if r.Includes(v) {
			return true
		}
	}
	return false
}

// Update reports a newer version of a module.
// The Err field will be set if an error occured.
type Update struct {
	Module module.Version
	Latest module.Version
	Err    error
}

// MarshalJSON implements json.Marshaler
func (u Update) MarshalJSON() ([]byte, error) {
	var err string
	if u.Err != nil {
		err = u.Err.Error()
	}
	return json.Marshal(struct {
		Module module.Version
		Latest module.Version
		Err    string `json:",omitempty"`
	}{
		Module: u.Module,
		Latest: u.Latest,
		Err:    err,
	})
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
		private := goenv.Get("GOPRIVATE")
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
				if err == ErrNoVersions {
					return nil
				}
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
