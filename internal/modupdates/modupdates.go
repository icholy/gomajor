package modupdates

import (
	"os"

	"github.com/icholy/gomajor/internal/modproxy"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

type Options struct {
	Pre      bool
	Cached   bool
	Major    bool
	Modules  []module.Version
	OnErr    func(module.Version, error)
	OnUpdate func(module.Version, string)
}

type update struct {
	Latest string
	Module module.Version
	Err    error
}

func Do(opt Options) {
	ch := make(chan update)
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
					ch <- update{Err: err, Module: m}
					return nil
				}
				v := mod.MaxVersion("", opt.Pre)
				if opt.Major && semver.Compare(semver.Major(v), semver.Major(m.Version)) <= 0 {
					return nil
				}
				if semver.Compare(v, m.Version) <= 0 {
					return nil
				}
				ch <- update{Latest: v, Module: m}
				return nil
			})
		}
		group.Wait()
		close(ch)
	}()
	for u := range ch {
		if u.Err != nil {
			if opt.OnErr != nil {
				opt.OnErr(u.Module, u.Err)
			}
		} else {
			if opt.OnUpdate != nil {
				opt.OnUpdate(u.Module, u.Latest)
			}
		}
	}
}
