package goenv

import (
	"reflect"
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		setenv string
		want   string
	}{
		{
			name:   "custom value",
			key:    "GOPROXY",
			setenv: "custom.proxy.com",
			want:   "custom.proxy.com",
		},
		{
			name: "non-existent env var",
			key:  "NONEXISTENT_GO_ENV_VAR",
			want: "",
		},
		{
			name:   "trailing newline trimmed",
			key:    "GOPROXY",
			setenv: "proxy.example.com\n",
			want:   "proxy.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setenv != "" {
				t.Setenv(tt.key, tt.setenv)
			}
			result := Get(tt.key)
			if result != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, result)
			}
		})
	}
}

func TestGOPROXYURL(t *testing.T) {
	tests := []struct {
		name   string
		setenv string
		want   []string
	}{
		{
			name:   "default when unset",
			setenv: "",
			want:   []string{"https://proxy.golang.org"},
		},
		{
			name:   "single proxy",
			setenv: "https://proxy.golang.org",
			want:   []string{"https://proxy.golang.org"},
		},
		{
			name:   "multiple proxies",
			setenv: "https://proxy.golang.org,direct",
			want:   []string{"https://proxy.golang.org"},
		},
		{
			name:   "proxies with whitespace",
			setenv: " https://proxy.golang.org , direct , https://custom.proxy.com ",
			want:   []string{"https://proxy.golang.org", "https://custom.proxy.com"},
		},
		{
			name:   "proxies with empty entries",
			setenv: "https://proxy.golang.org,,direct, ,https://custom.proxy.com",
			want:   []string{"https://proxy.golang.org", "https://custom.proxy.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setenv != "" {
				t.Setenv("GOPROXY", tt.setenv)
			}
			result := GOPROXYURL()
			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("Expected %v, got %v", tt.want, result)
			}
		})
	}
}


