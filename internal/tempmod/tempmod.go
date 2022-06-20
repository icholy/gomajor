package tempmod

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
)

type Mod struct {
	Dir     string
	Imports []string
}

func (m *Mod) Delete() error {
	return os.RemoveAll(m.Dir)
}

func Init(imports []string) (*Mod, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("go", "mod", "init", "tmp")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var src bytes.Buffer
	src.WriteString("package tmp\n")
	for _, path := range imports {
		src.WriteString("import _ \"" + path + "\"\n")
	}
	src.WriteString("func main() {}\n")
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, src.Bytes(), os.ModePerm); err != nil {
		return nil, err
	}
	cmd = exec.Command("go", "get")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &Mod{
		Dir:     dir,
		Imports: imports,
	}, nil
}
