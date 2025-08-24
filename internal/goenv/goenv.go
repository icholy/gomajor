package goenv

import (
	"net/url"
	"os/exec"
	"strings"

	"golang.org/x/sync/singleflight"
)

var group singleflight.Group

// Get retrieves a Go environment variable using 'go env <name>'.
// Results are cached using singleflight to avoid duplicate calls.
func Get(key string) string {
	value, _, _ := group.Do(key, func() (any, error) {
		cmd := exec.Command("go", "env", key)
		output, err := cmd.Output()
		if err != nil {
			return "", nil
		}
		return strings.TrimSpace(string(output)), nil
	})
	return value.(string)
}

// GOPROXYURL returns the GOPROXY URLs as a slice, filtering out non-URL values.
func GOPROXYURL() []string {
	value := Get("GOPROXY")
	if value == "" {
		return nil
	}
	var proxies []string
	for _, p := range strings.Split(value, ",") {
		p = strings.TrimSpace(p)
		if u, err := url.Parse(p); err == nil && u.Scheme != "" {
			proxies = append(proxies, p)
		}
	}
	return proxies
}
