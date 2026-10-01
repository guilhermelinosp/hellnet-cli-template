// Package cli provides a small, engine-agnostic abstraction for building
// command-line applications.
//
// Application code interacts exclusively with the types in this package
// (App, Command, Flag and the error helpers). The concrete execution engine
// lives behind the [Engine] interface, implemented in the internal/cli/cobra
// subpackage and activated through a side-effect import (same mechanism as
// image codecs in the standard library):
//
//	import (
//	    _ "github.com/guilhermelinosp/hellnet-cli-template/internal/cli/cobra"
//	)
//
// Swapping engines later means changing exactly one underscore import — no
// application code moves.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Engine executes parsed CLI definitions against an [App]. Implementations
// live under internal/cli/<name> and register themselves with
// [RegisterEngine]; they are chosen once per binary, never per call site.
//
// Contract:
//   - Success (a command ran, help/version was rendered, completion emitted)
//     returns nil — output went through app.Out().
//   - Failures return an error that either implements [ExitCoder] (user or
//     validation code) or wraps one tagged as usage error ([ExitUsage]).
//   - Engines never touch os.Exit, never print to os.Stdout/os.Stderr
//     directly and never expose their own concrete types upward.
type Engine interface {
	Execute(ctx context.Context, app *App) error
}

// EngineFactory constructs a stateless engine instance.
type EngineFactory func() Engine

// engineName is the engine activated by [New]. Changing implementations is
// a one-line edit here — apps stay untouched.
const engineName = "cobra"

var registry = map[string]EngineFactory{}

// RegisterEngine makes a factory available to [New]. Called from engine
// packages' init functions, mirroring database/image driver registration.
func RegisterEngine(name string, f EngineFactory) {
	if name == "" || f == nil {
		panic("cli: invalid engine registration")
	}
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("cli: engine %q registered twice", name))
	}
	registry[name] = f
}

func defaultEngine() Engine {
	f, ok := registry[engineName]
	if !ok {
		panic(fmt.Sprintf("cli: no engine registered as %q — add the corresponding blank import", engineName))
	}
	return f()
}

// Options configures the top-level application surface.
type Options struct {
	// Name is the binary name shown across help/version output.
	Name string
	// Version is injected at build time (see internal/build).
	Version string
	// Long is the root help body describing what the tool does.
	Long string
	// Example renders an examples block in the root help.
	Example string
}

// App is a configured CLI application: root metadata, command tree, streams
// and the selected engine. Create once in main; reuse safely across test
// invocations because every Run works on a fresh snapshot (see clone).
type App struct {
	opts   Options
	root   *Command
	engine Engine
	args   []string // invocation argv (engine-visible)

	out    io.Writer
	errOut io.Writer
}

// New creates an application backed by the registered default engine.
func New(opts Options) *App {
	return newWith(opts, defaultEngine())
}

// newWith injects engine and streams — the seam used by tests to prove that
// application behavior does not depend on any particular engine.
func newWith(opts Options, eng Engine) *App {
	if opts.Name == "" {
		opts.Name = "app"
	}
	a := &App{opts: opts, engine: eng}
	a.resetRoot()
	a.out, a.errOut = os.Stdout, os.Stderr
	return a
}

func (a *App) resetRoot() {
	a.root = &Command{
		Name:    a.opts.Name,
		Short:   a.opts.Long,
		Long:    a.opts.Long,
		Example: a.opts.Example,
	}
	a.root.Commands = append(a.root.Commands, newVersionCommand(a.opts.Name))
}

// Add registers top-level commands. Nil entries and entries without a name
// are ignored so wiring code never needs defensive checks.
func (a *App) Add(cmds ...*Command) {
	for _, c := range cmds {
		if c == nil || c.Name == "" {
			continue
		}
		a.root.Commands = append(a.root.Commands, c)
	}
}

// SetOutput redirects stdout/stderr, enabling hermetic tests without process
// plumbing. Output discipline stays identical in production.
func (a *App) SetOutput(out, errOut io.Writer) {
	a.out = out
	a.errOut = errOut
}

// Accessors used by engines and tests ----------------------------------------

// Name returns the configured binary name.
func (a *App) Name() string { return a.opts.Name }

// Version returns the injected version string.
func (a *App) Version() string { return a.opts.Version }

// Root exposes the current invocation's command tree (freshly cloned per
// Run — mutating it affects this invocation only).
func (a *App) Root() *Command { return a.root }

// Args returns the raw invocation arguments for parsing.
func (a *App) Args() []string { return a.args }

// Out returns the standard output stream.
func (a *App) Out() io.Writer { return a.out }

// ErrOut returns the error stream.
func (a *App) ErrOut() io.Writer { return a.errOut }

// Run executes the full pipeline — snapshot, parse, validate, user Run — and
// maps outcome to a process exit code. It NEVER terminates the process;
// main owns that decision:
//
//	os.Exit(app.Run(ctx, os.Args[1:]))
//
// Output contract (identical on every engine):
//
//	stdout — results, help, version        (single-line, script-friendly)
//	stderr — errors and usage hints        (no stack traces, ever)
func (a *App) Run(ctx context.Context, args []string) int {
	if ctx == nil {
		ctx = context.Background()
	}
	run := a.runSnapshot()
	run.args = append([]string(nil), args...) // defensive copy; engines may retain

	err := run.engine.Execute(ctx, run)
	if err == nil {
		return ExitSuccess
	}

	code := ExitCodeOf(err)
	_, _ = fmt.Fprintf(run.errOut, "%s: error: %v\n", run.opts.Name, err)
	if code == ExitUsage {
		printAvailableCommands(run.errOut, run.root)
	}
	return code
}

// printAvailableCommands complements misuse errors with a terse recovery
// hint listing legal top-level commands, hidden ones excluded.
func printAvailableCommands(w io.Writer, root *Command) {
	names := make([]string, 0, len(root.Commands))
	for _, c := range root.Commands {
		if !c.Hidden {
			names = append(names, c.Name)
		}
	}
	_, _ = fmt.Fprintf(w, "run '%s --help' for usage and available commands (%s).\n",
		root.Name, joinNames(names))
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
