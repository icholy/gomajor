package packages

import (
	"testing"

	"golang.org/x/mod/module"
)

func TestIndex(t *testing.T) {
	var idx Index

	idx.Add(module.Version{
		Path:    "github.com/aws/aws-sdk-go-v2",
		Version: "v1.15.11",
	})

	idx.Add(module.Version{
		Path:    "github.com/aws/aws-sdk-go-v2/config",
		Version: "v1.15.11",
	})

	idx.Add(module.Version{
		Path:    "github.com/aws/aws-sdk-go-v2/service/codepipeline",
		Version: "v1.13.6",
	})

	tests := []struct {
		path string
		ok   bool
		mod  module.Version
	}{
		{
			path: "",
		},
		{
			path: "github.com/aws/aws-sdk-go",
		},
		{
			path: "github.com/aws/aws-sdk-go-v2",
			ok:   true,
			mod: module.Version{
				Path:    "github.com/aws/aws-sdk-go-v2",
				Version: "v1.15.11",
			},
		},
		{
			path: "github.com/aws/aws-sdk-go-v2/missing",
			ok:   true,
			mod: module.Version{
				Path:    "github.com/aws/aws-sdk-go-v2",
				Version: "v1.15.11",
			},
		},
		{
			path: "github.com/aws/aws-sdk-go-v2/config",
			ok:   true,
			mod: module.Version{
				Path:    "github.com/aws/aws-sdk-go-v2/config",
				Version: "v1.15.11",
			},
		},
		{
			path: "github.com/aws/aws-sdk-go-v2/service/codepipeline",
			ok:   true,
			mod: module.Version{
		Path:    "github.com/aws/aws-sdk-go-v2/service/codepipeline",
		Version: "v1.13.6",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			mod, ok := idx.Lookup(tt.path)
			if ok != tt.ok || mod != tt.mod {
				t.Fatalf(
					"expected lookup to be %v (%v), got %v (%v)",
					tt.mod, tt.ok,
					mod, ok,
				)
			}
		})
	}

}
