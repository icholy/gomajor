package goenv

import (
	"net/url"
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/sync/singleflight"
)

var group singleflight.Group

// Get retrieves a Go environment variable using 'go env <name>'.
// Results are cached using singleflight to avoid duplicate calls, unless running in test mode.
func Get(key string) string {
	get := func() string {
		cmd := exec.Command("go", "env", key)
		output, err := cmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(output))
	}
	// Don't cache during tests to allow environment overrides
	if testing.Testing() {
		return get()
	}
	value, _, _ := group.Do(key, func() (any, error) {
		return get(), nil
	})
	return value.(string)
}

// GOPROXYURL returns the GOPROXY URLs as a slice of parsed URLs, filtering out non-URL values.
func GOPROXYURL() []*url.URL {
	value := Get("GOPROXY")
	if value == "" {
		return nil
	}
	var proxies []*url.URL
	for _, p := range strings.Split(value, ",") {
		p = strings.TrimSpace(p)
		if u, err := url.Parse(p); err == nil && u.Scheme != "" {
			proxies = append(proxies, u)
		}
	}
	return proxies
}
