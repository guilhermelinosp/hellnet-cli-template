package cobra

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/guilhermelinosp/hellnet-cli-template/internal/cli"
)

// harness builds a real Cobra-backed app bound to memory streams.
type harness struct {
	app *cli.App
	out *strings.Builder
	err *strings.Builder
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	out, errOut := &strings.Builder{}, &strings.Builder{}
	app := cli.New(cli.Options{Name: "app", Version: "1.2.3"})
	app.SetOutput(out, errOut)
	t.Cleanup(func() {})
	return &harness{app: app, out: out, err: errOut}
}

func run(h *harness, args ...string) int {
	return h.app.Run(context.Background(), args)
}

func TestCobraTranslatesAndParsesFlags(t *testing.T) {
	h := newHarness(t)
	var gotName string
	var gotCount int
	var gotTimeout time.Duration
	var gotJSON bool

	h.app.Add(&cli.Command{
		Name:  "deploy",
		Short: "Deploy something",
		Flags: []*cli.Flag{
			cli.StringFlag("name", "n", "dev", "target name"),
			cli.IntFlag("count", "c", 1, "replica count"),
			cli.DurationFlag("timeout", "", 30*time.Second, "deadline"),
			cli.BoolFlag("json", "", false, "json output"),
		},
		Run: func(_ context.Context, c *cli.Command, _ []string) error {
			gotName = c.String("name")
			gotCount = c.Int("count")
			gotTimeout = c.Duration("timeout")
			gotJSON = c.Bool("json")
			return nil
		},
	})

	if code := run(h, "deploy", "-n", "prod", "--count=3", "--timeout=1m30s", "--json"); code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, h.err.String())
	}
	if gotName != "prod" || gotCount != 3 || gotTimeout != 90*time.Second || !gotJSON {
		t.Fatalf("parsed values mismatch: %s %d %v %v", gotName, gotCount, gotTimeout, gotJSON)
	}
}

func TestCobraInvalidFlagValueExitsUsage(t *testing.T) {
	h := newHarness(t)
	h.app.Add(&cli.Command{
		Name: "deploy",
		Run:  func(context.Context, *cli.Command, []string) error { return nil },
	})
	if code := run(h, "deploy", "--count=NaN"); code != cli.ExitUsage {
		t.Fatalf("exit=%d want %d (%q)", code, cli.ExitUsage, h.err.String())
	}
	if !strings.Contains(h.err.String(), "error:") {
		t.Fatalf("missing error line: %q", h.err.String())
	}
}

func TestCobraVersionFlagMatchesContract(t *testing.T) {
	h := newHarness(t)
	if code := run(h, "--version"); code != 0 {
		t.Fatalf("--version exit=%d (%q)", code, h.err.String())
	}
	if got := h.out.String(); got != "app 1.2.3\n" {
		t.Fatalf("--version output %q does not match single-line contract", got)
	}
	if strings.Contains(h.out.String(), "version ") { // cobra's default wording must be gone
		t.Fatalf("default cobra template leaked: %q", h.out.String())
	}
}

func TestCobraUnknownCommandAndSuggestions(t *testing.T) {
	h := newHarness(t)
	h.app.Add(&cli.Command{Name: "healthcheck", Run: func(context.Context, *cli.Command, []string) error { return nil }})

	if code := run(h, "healthchek"); code != cli.ExitUsage {
		t.Fatalf("typo command should exit usage; got %d", code)
	}
	errLine := h.err.String()
	if !strings.Contains(errLine, `"healthchek"`) {
		t.Fatalf("expected unknown-command diagnostic, got %q", errLine)
	}
	if !strings.Contains(errLine, "healthcheck") { // did-you-mean suggestion included
		t.Fatalf("expected suggestion in %q", errLine)
	}
}

func TestCobraAliasesResolve(t *testing.T) {
	h := newHarness(t)
	executed := false
	h.app.Add(&cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Run: func(context.Context, *cli.Command, []string) error {
			executed = true
			return nil
		},
	})
	if code := run(h, "ls"); code != 0 || !executed {
		t.Fatalf("alias execution failed: code=%d executed=%v", code, executed)
	}
}

func TestCobraNestedSubcommandsAndContext(t *testing.T) {
	type keyT struct{}
	h := newHarness(t)

	var sawCtxKey bool
	var sawArgs []string
	h.app.Add(&cli.Command{
		Name: "db",
		Commands: []*cli.Command{
			{
				Name: "migrate",
				Args: cli.ExactArgs(1),
				Run: func(ctx context.Context, _ *cli.Command, args []string) error {
					_, sawCtxKey = ctx.Value(keyT{}).(string)
					sawArgs = args
					return nil
				},
			},
		},
	})

	ctx := context.WithValue(context.Background(), keyT{}, "injected")
	code := h.app.Run(ctx, []string{"db", "migrate", "up"})
	if code != 0 || !sawCtxKey || len(sawArgs) != 1 || sawArgs[0] != "up" {
		t.Fatalf("nested dispatch/context failed: code=%d ctx=%v args=%v", code, sawCtxKey, sawArgs)
	}
}

func TestCobraArgsValidation(t *testing.T) {
	h := newHarness(t)
	h.app.Add(&cli.Command{Name: "greet", Args: cli.ExactArgs(1), Run: func(context.Context, *cli.Command, []string) error { return nil }})
	if code := run(h, "greet"); code != cli.ExitUsage {
		t.Fatalf("ExactArgs violation exit=%d", code)
	}
}

func TestCobraCompletionShipsByDefault(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		h := newHarness(t)
		if code := run(h, "completion", shell); code != 0 || len(h.out.String()) < 100 {
			t.Fatalf("%s completion missing/broken: code=%d bytes=%d", shell, code, len(h.out.String()))
		}
	}
}
