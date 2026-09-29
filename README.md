# golang-cli-template

> Opinionated, production-ready **Go CLI template**. You get ~80% of the
> infrastructure pre-wired so you can focus on your business logic.

```go
// This is all your business command looks like.
// Cobra exists — but you never touch it.
Run: func(ctx context.Context, c *cli.Command, args []string) error {
    c.Printf("hello %s\n", c.String("name"))
    return nil
},
```

## Why this template

| Concern                | What you get out of the box                                              |
| ---------------------- | ------------------------------------------------------------------------ |
| CLI framework          | Own thin abstraction (`internal/cli`) with **Cobra as private engine**    |
| Version/build metadata | `version`, `commit`, `build date`, `go version` — injected by GoReleaser |
| Config                 | stdlib-first: defaults → env vars (`<APP>_LOG_LEVEL`…)                    |
| Logging                | `log/slog` structured, ready (text→stderr, json→stdout)                   |
| Errors                 | Centralized taxonomy, exit codes `0/1/2`, no stack traces for users       |
| Shutdown               | `SIGINT`/`SIGTERM` wired through `context.Context`                        |
| Tests                  | Unit + engine-adapter + true end-to-end process suite                     |
| CI/CD                  | GitHub Actions: lint, race tests, coverage, govulncheck, CodeQL           |
| Release                | GoReleaser → 5 platform binaries + checksums + changelog                  |
| Shell completion       | bash · zsh · fish · powershell, generated automatically                    |

Dependencies: **`spf13/cobra` only** (+ its `pflag`). No Viper, no Bubble Tea,
no zap, no DI frameworks. Every addition must justify itself.

---

## Quick start

1. **Use this template** on GitHub (or `git clone`).
2. Rename everything with one command:

```bash
make setup        # prompts app name + module path, rewrites module/dir/binary
```

3. Run it:

```bash
make test
make build
./bin/<your-app> --help
./bin/<your-app> health --json
```

That's the whole ceremony. Start writing commands.

### Manual renaming (if you prefer)

```bash
# 1. go.mod: rename the module; internal imports cascade from this string
sed -i '' 's|github.com/guilhermelinosp/golang-cli-template|github.com/YOU/YOUR-REPO|' go.mod
grep -rl --include='*.go' github.com/guilhermelinosp/golang-cli-template . \
  | xargs sed -i '' 's|github.com/guilhermelinosp/golang-cli-template|github.com/YOU/YOUR-REPO|'

# 2. rename the entrypoint dir and binary default
mv cmd/app cmd/yourapp
sed -i '' 's|^APP ?= .*|APP ?= yourapp|' Makefile
```

`make setup` automates exactly these steps plus a smoke test.

---

## Architecture

```
Application code            ← YOUR business logic lives here
      │                        (commands built ONLY with internal/cli types)
      ▼
internal/cli                ← abstraction layer: Command, Flag, App, errors
      │                        no engine types may leak upward (CI-enforced)
      ▼
internal/cli/cobra          ← THE ONLY package importing spf13/cobra
      │
      ▼
Cobra / pflag               ← swappable implementation detail
```

**The rule:** application code depends on the abstraction, never on Cobra.
The boundary is enforced in CI via `make verify` (a grep guard), so the
engine can be replaced later without touching a single business command.

```
internal/
├── cli/                  # abstraction layer (the API you program against)
│   ├── cli.go            #   App lifecycle: New/Add/Run → exit codes
│   ├── command.go        #   Command model + arg validators (NoArgs/ExactArgs…)
│   ├── flags.go          #   declarative flags + typed accessors
│   ├── errors.go         #   Errorf/Usagef/Exitf taxonomy + exit-code mapping
│   ├── output.go         #   stream discipline (results→stdout) + version cmd
│   └── clone.go          #   per-run snapshot isolation (test-friendly)
├── cli/cobra/            # engine adapter (only place that imports Cobra)
├── config/config.go      # stdlib-first config: defaults + env vars
├── logging/logging.go    # slog wrapper: levels, text/json, service identity
├── service/health/       # example DOMAIN SERVICE behind the health command
└── build/info.go         # ldflags-injected metadata
cmd/app/                  # main.go (wiring only) + example commands
scripts/setup.sh          # template bootstrap automation
```

Why not keep it simpler? The indirection costs ~200 lines and buys:
engine portability (req: future non-Cobra engines), hermetic testing
(snapshot isolation lets every test call `app.Run(...)` repeatedly without
process spawning), uniform UX contracts (exit codes, error rendering) that
don't drift when the engine changes.

---

## Creating a command

Define it anywhere sensible; wire it in `cmd/app/main.go`.

```go
package main

import (
	"context"

	"github.com/YOU/YOUR-REPO/internal/cli"
	"github.com/YOU/YOUR-REPO/internal/service/greet"
)

func newGreetCommand(svc *greet.Service) *cli.Command {
	return &cli.Command{
		Name:    "greet",
		Aliases: []string{"hi"},
		Short:   "Greet someone",
		Example: "$ app greet --name world",
		Flags: []*cli.Flag{
			cli.StringFlag("name", "n", "world", "who to greet"),
			cli.IntFlag("times", "t", 1, "repeat count"),
		},
		Args: cli.NoArgs,
		Run: func(ctx context.Context, c *cli.Command, args []string) error {
			for i := 0; i < c.Int("times"); i++ {
				svc.Say(c.String("name")) // services carry logic; commands stay thin
			}
			return nil
		},
	}
}
```

Registration:

```go
app.Add(newGreetCommand(greet.New(logger)))
```

Patterns to follow (already demonstrated in `cmd/app/health.go`):

* constructor injection of services — no locators, no globals;
* commands parse inputs and format outputs; **services own logic**;
* print results with `c.Printf`; report failures by **returning errors**
  (never printing them yourself).

## Flags

Declared inline, read typed inside Run:

```go
Flags: []*cli.Flag{
	cli.StringFlag("name", "n", "world", "who to greet"),
	cli.BoolFlag("json", "j", false, "single-line JSON"),
	cli.IntFlag("count", "", 3, "iterations"),
	cli.Float64Flag("ratio", "", 0.5, "sampling ratio"),
	cli.DurationFlag("timeout", "", 30*time.Second, "per-attempt deadline"),
},

// inside Run:
name := c.String("name")
if c.Changed("count") { /* user set it explicitly */ }
```

Unsupported values become usage errors automatically (exit 2).

## Configuration

Precedence: **flags > env > defaults** (files can slot into `config.Load`
later without changing call sites — deliberately deferred, YAGNI).

Variables are namespaced by binary name: an app named `my-cli` reads:

| Variable              | Values                  | Default |
| --------------------- | ----------------------- | ------- |
| `MY_CLI_LOG_LEVEL`    | debug info warn error   | info    |
| `MY_CLI_LOG_FORMAT`   | text json               | text    |

Add your own fields in `internal/config.Config` following the same pattern.

## Errors & exit codes

Return values decide everything — the framework owns rendering:

| Situation                     | Return                             | Exit |
| ----------------------------- | ---------------------------------- | ---- |
| success                       | `nil`                              | 0    |
| domain/business failure       | `cli.Errorf("quota %d exceeded")`  | 1    |
| custom semantics              | `cli.Exitf(64, "invalid cron")`    | n/a  |
| bad flag/arg usage            | automatic, or `cli.Usagef(...)`    | 2    |

Users never see stack traces. Wrapping is preserved internally:

```go
return cli.Errorf("read config: %w", err)
```

Output contract (stable across engines):

* stdout → results, help, `--version` (script-friendly single lines)
* stderr → errors, usage hints, human logs

## Testing

Three layers are already paid for:

1. **Unit** — packages next to their code (`go test ./...`).
2. **Engine-agnostic CLI tests** — drive `app.Run` against any `Engine`
   implementation, asserting exit-code mapping and snapshot isolation
   (`internal/cli/cli_test.go`).
3. **End-to-end process tests** — compile the real binary once and execute
   scenarios checking true exit codes and stream separation
   (`cmd/app/main_test.go`). Copy cases from there when adding commands.

Commands that use `c.Printf` are testable without process plumbing because
streams resolve per invocation (see `TestSnapshotIsolationBetweenRuns`).

## Building & releasing

```bash
make build     # bin/<app> with stamped version/commit/date
VERSION=1.2.3 make build   # override anything explicitly
make release   # local snapshot build of all platforms (dist/)
git tag v1.2.3 && git push origin v1.2.3   # real release → GitHub Actions
```

GoReleaser matrix: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64,
windows/amd64 — tar.gz archives (zip on Windows) plus `checksums.txt`
(SHA256) and an auto-generated changelog filtered from conventional commits.

Version contract — both surfaces byte-identical:

```console
$ <app> version          # or --version / -v
<app> 1.2.3 (commit abc1234, built 2026-08-26T12:00:00Z, go go1.27.0)
```

Metadata injection points live in `internal/build/info.go` (ldflags
documented there); nothing is hardcoded.

## Shell completion

Works everywhere, zero configuration — shipped by the engine adapter:

```bash
source <(<app> completion bash)
eval "$(<app> completion zsh)"      # or fish / powershell
```

If you define your own nested groups they appear automatically.

## Swapping/extending the CLI engine

One import line controls which engine activates:

```go
import _ "github.com/YOU/YOUR-REPO/internal/cli/cobra" // currently in cmd/app/main.go
```

To replace: create `internal/cli/yourengine` implementing `cli.Engine`,
register with `cli.RegisterEngine("yourname", factory)`, swap the blank
import. Application code does not change. Adding TUI sugar (e.g. Bubble Tea),
plugin loaders or telemetry means new commands/engine features — the core
stays untouched. **Keep Bubble Tea out of the base template**: CLIs must stay
pipe-and-script friendly first.

## CI / CD

| Workflow       | Trigger | Jobs (all org reusables from [ci-templates](https://github.com/guilhermelinosp/ci-templates)) |
| -------------- | ------- | ---------------------------------------------------------------------------------------------- |
| `pr-check.yml` | PR      | shellcheck, merge-check, gitleaks, labeler, `go-quality` (+boundary guard), govulncheck — org reusables · CodeQL inline per-repo |
| `pipeline.yml` | push to `main` | semver release → go test/build → GoReleaser cross-platform binaries onto the release (+boundary guard) → govulncheck |

Secret scanning is native to GitHub — enable *Push protection* under repo
Settings → Code security. Commits follow [Conventional Commits](https://www.conventionalcommits.org);
`lefthook install` wires the local guards (`fmt`, vet, tests, lint, secrets,
conventional message).

## Principles baked in

KISS · YAGNI · DRY · stdlib-first · explicit dependencies · small interfaces
· separation of concerns. Abstractions here exist because each one removes a
recurring pain (engine lock-in, flaky flag state across tests, drift between
version surfaces). If you cannot name the pain your addition removes, leave
it out.

## License

[Apache 2.0](LICENSE)

<!-- release pipeline verification -->

<!-- verify round 2 -->

<!-- verify round 3 -->

<!-- verify round 4 -->
