package tempmod

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

type Mod struct {
	Dir string
}

func Create(name string) (Mod, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "tempmod")
	if err != nil {
		return Mod{}, err
	}
	m := Mod{Dir: dir}
	if name == "" {
		name = "temp"
	}
	if err := m.ExecGo("mod", "init", name); err != nil {
		return Mod{}, err
	}
	return m, nil
}

func (m Mod) ExecGo(args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr
	cmd.Dir = m.Dir
	return cmd.Run()
}

func (m Mod) Delete() error {
	return os.RemoveAll(m.Dir)
}

func (m Mod) UsePackage(pkgpath string) error {
	tmp, err := os.CreateTemp(m.Dir, "*-import.go")
	if err != nil {
		return err
	}
	defer tmp.Close()
	var src bytes.Buffer
	src.WriteString("package tmp\n")
	fmt.Fprintf(&src, "import _ %q\n", pkgpath)
	src.WriteString("func main() {}\n")
	if _, err := src.WriteTo(tmp); err != nil {
		return err
	}
	return tmp.Close()
}
