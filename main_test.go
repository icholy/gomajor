package main

import (
	"net/http/httptest"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/icholy/gomajor/internal/modproxy/testmodproxy"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"gomajor": main,
	})
}

func TestListCommand(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/testscript/list",
		Setup: func(env *testscript.Env) error {
			proxy, err := testmodproxy.Load("testdata/modules")
			if err != nil {
				return err
			}
			server := httptest.NewServer(proxy)
			env.Vars = append(env.Vars, "GOPROXY="+server.URL)
			env.Defer(func() { server.Close() })
			return nil
		},
	})
}

func TestHelpCommand(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/testscript/help",
	})
}

func TestPathCommand(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/testscript/path",
	})
}
