package latest

import "testing"

func TestVersion(t *testing.T) {
	v, err := Version("github.com/google/go-cmp/cmp")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Version: %s", v)
}
