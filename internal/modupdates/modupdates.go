package modupdates

import (
	"os"

	"github.com/icholy/gomajor/internal/modproxy"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

type Options struct {
	Pre     bool
	Cached  bool
	Major   bool
	Modules []module.Version
	OnErr   func(module.Version, error)
}

type Update struct {
	Latest string
	Module module.Version
}

func List(opt Options) chan Update {
	ch := make(chan Update)
	go func() {
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
				mod, err := modproxy.Latest(m.Path, opt.Cached)
				if err != nil {
					if opt.OnErr != nil {
						opt.OnErr(m, err)
					}
					return nil
				}
				v := mod.MaxVersion("", opt.Pre)
				if opt.Major && semver.Compare(semver.Major(v), semver.Major(m.Version)) <= 0 {
					return nil
				}
				if semver.Compare(v, m.Version) <= 0 {
					return nil
				}
				ch <- Update{Latest: v, Module: m}
				return nil
			})
		}
		group.Wait()
		close(ch)
	}()
	return ch
}
