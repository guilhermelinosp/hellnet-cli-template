package cli_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestEngineBoundary enforces the architecture rule: application code depends on internal/cli,
// never on the engine. Only internal/cli/cobra may import spf13/cobra or spf13/pflag, so the engine
// can be swapped without touching a single business command.
func TestEngineBoundary(t *testing.T) {
	root := filepath.Join("..", "..")
	allowed := filepath.Join("internal", "cli", "cobra")
	banned := []string{"github.com/spf13/cobra", "github.com/spf13/pflag"}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			if d.Name() == ".git" || rel == allowed {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			for _, b := range banned {
				if p == b || strings.HasPrefix(p, b+"/") {
					t.Errorf("%s imports %s: only %s may import the CLI engine", rel, p, allowed)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
