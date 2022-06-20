package tempmod

import (
	"os"
	"os/exec"
)

type Mod struct {
	Dir string
}

func Create(name string) (Mod, error) {
	if name == "" {
		name = "temp"
	}
	dir, err := os.MkdirTemp(os.TempDir(), "tempmod")
	if err != nil {
		return Mod{}, err
	}
	m := Mod{Dir: dir}
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
