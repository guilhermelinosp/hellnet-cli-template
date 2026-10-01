package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/guilhermelinosp/hellnet-cli-template/internal/build"
)

// Printf writes a formatted business result to the invocation's stdout.
//
// Output discipline for every Run implementation:
//
//   - results   → c.Printf / c.Println   (stdout)
//   - failures  → return an error        (framework renders on stderr)
//
// Printing errors manually bypasses exit-code mapping and breaks script
// integrations — always return instead. Writes are best-effort by design:
// nothing actionable exists when a result stream misbehaves.
func (c *Command) Printf(format string, a ...any) {
	_, _ = fmt.Fprintf(c.Stdout(), format, a...)
}

// Println writes a result line to the invocation's stdout.
func (c *Command) Println(a ...any) {
	_, _ = fmt.Fprintln(c.Stdout(), a...)
}

// Stdout exposes the bound stream; exported for commands composing helpers
// that take an io.Writer (e.g. encoding/json encoder pipelines).
func (c *Command) Stdout() io.Writer {
	if c.out == nil {
		return os.Stdout
	}
	return c.out
}

// wireOutput attaches invocation-scoped streams recursively after cloning;
// unexported on purpose — only App snapshots may rebind streams.
func (c *Command) wireOutput(out io.Writer) {
	c.out = out
	for _, sub := range c.Commands {
		sub.wireOutput(out)
	}
}

// newVersionCommand ships inside every application generated from this
// template so `<app> version` always agrees with injected metadata. It reads
// build information stamped by GoReleaser/ldflags (see internal/build).
func newVersionCommand(name string) *Command {
	return &Command{
		Name:  "version",
		Short: "Show version and build information",
		Long: "Prints a single-line, script-friendly summary containing\n" +
			"the semantic version, commit SHA, build date and toolchain.",
		Example: fmt.Sprintf("# pipe into another tool\n  $ %s version | cut -d' ' -f2", name),
		Run: func(_ context.Context, c *Command, _ []string) error {
			c.Printf("%s %s\n", name, build.Get())
			return nil
		},
	}
}
