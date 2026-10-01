// Package cobra implements the cli.Engine interface on top of spf13/cobra.
//
// This is the ONLY package in the repository allowed to import Cobra.
// Everything the application can observe flows back through the abstraction,
// so replacing this directory swaps engines without touching business code.
package cobra

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guilhermelinosp/hellnet-cli-template/internal/cli"
)

func init() {
	cli.RegisterEngine("cobra", func() cli.Engine { return &Engine{} })
}

// Engine is the stateless Cobra-backed implementation of cli.Engine.
type Engine struct{}

// Execute builds a fresh Cobra tree from app.Root() and dispatches once.
// Success paths (command ran, help rendered, --version shown, shell
// completion emitted) return nil; failures are classified by the run phase:
// anything failing before user code gets usage semantics (exit 2).
func (e *Engine) Execute(ctx context.Context, app *cli.App) error {
	src := app.Root()

	root, tracker := translateTree(src)
	if root == nil {
		return nil
	}

	// Rendering centralization: engines stay mute; cli.App.Run owns stderr,
	// while native help/version/completion follow the configured streams.
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetOut(app.Out())
	root.SetErr(app.ErrOut())

	if v := app.Version(); v != "" {
		// Native flag support short-circuits before touching RunE; the
		// template overrides the renderer so `--version` and the built-in
		// `version` command emit byte-identical, script-friendly lines.
		root.Version = v
		root.SetVersionTemplate(fmt.Sprintf("%s %s\n", root.Name(), v))
	}

	root.SetArgs(app.Args())
	return tracker.classify(root.ExecuteContext(ctx))
}

// runTracker records whether any user RunFunc started executing, enabling
// phase-aware error classification (pre-user failures = usage layer).
type runTracker struct{ reached bool }

func (t *runTracker) mark() { t.reached = true }

// classify converts engine-surfaced errors into abstraction taxonomy:
//   - failure before any user code → usage semantics (exit 2),
//   - failure returned by user code → passthrough (codes carried inside).
func (t *runTracker) classify(err error) error {
	switch {
	case err == nil || t.reached:
		return err
	default:
		return usageFailure{err}
	}
}

type usageFailure struct{ err error }

func (u usageFailure) Error() string { return u.err.Error() }
func (usageFailure) ExitCode() int   { return cli.ExitUsage }
func (u usageFailure) Unwrap() error { return u.err }

// translateTree converts an entire abstraction tree in one pass, sharing the
// tracker across nodes for classification, and pairing each generated node
// with its source instance pointer.
func translateTree(src *cli.Command) (*cobra.Command, *runTracker) {
	tracker := &runTracker{}
	return translateNode(src, tracker), tracker
}

func translateNode(c *cli.Command, tracker *runTracker) *cobra.Command {
	cc := &cobra.Command{
		Use:     c.Name,
		Aliases: append([]string(nil), c.Aliases...),
		Short:   c.Short,
		Long:    orDefault(c.Long, c.Short),
		Example: indentExample(c.Example),
		Hidden:  c.Hidden,
	}

	bindFlags(cc, c.Flags)

	if run := c.Run; run != nil {
		cc.RunE = func(cmd *cobra.Command, args []string) error {
			tracker.mark()
			return run(cmd.Context(), c, args)
		}
	} else if len(c.Commands) == 0 {
		// Leaf grouping placeholder keeps `myapp group` answering help.
		cc.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
	}

	if validate := c.Args; validate != nil {
		src := c
		cc.Args = func(_ *cobra.Command, args []string) error { return validate(src, args) }
	}

	for _, sub := range c.Commands {
		if node := translateNode(sub, tracker); node != nil {
			cc.AddCommand(node)
		}
	}
	return cc
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// indentExample renders examples with conventional two-space offset while
// preserving author-provided line breaks verbatim.
func indentExample(example string) string {
	if example == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(example, "\n"), "\n")
	for i, ln := range lines {
		if strings.TrimSpace(ln) != "" && !strings.HasPrefix(ln, "  ") {
			lines[i] = "  " + ln
		}
	}
	return strings.Join(lines, "\n")
}

// bindFlags mirrors declared flags onto the command through value bridges —
// Cobra owns syntax parsing, the template owns storage and meaning.
func bindFlags(cc *cobra.Command, flags []*cli.Flag) {
	fs := cc.Flags()
	for _, f := range flags {
		if f == nil || f.Name == "" {
			continue
		}
		f.ResetRuntime()
		pf := fs.VarPF(newFlagBridge(f), f.Name, f.Shorthand, f.Usage)
		pf.Hidden = f.Hidden
		pf.NoOptDefVal = boolNoOptDefault(f.Type)
	}
}
