# hellnet-cli-template

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

[![pipeline](https://github.com/guilhermelinosp/hellnet-cli-template/actions/workflows/pipeline.yml/badge.svg)](https://github.com/guilhermelinosp/hellnet-cli-template/actions/workflows/pipeline.yml)
[![pr-check](https://github.com/guilhermelinosp/hellnet-cli-template/actions/workflows/pr-check.yml/badge.svg)](https://github.com/guilhermelinosp/hellnet-cli-template/actions/workflows/pr-check.yml)
[![CodeQL](https://github.com/guilhermelinosp/hellnet-cli-template/actions/workflows/codeql.yml/badge.svg)](https://github.com/guilhermelinosp/hellnet-cli-template/actions/workflows/codeql.yml)

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

## Initialize from this template

After **Use this template**, clone the new repository and run:

```bash
scripts/setup.sh -n <app-name> -m github.com/<you>/<repo>   # renames the module, imports and cmd/app
scripts/setup-repo.sh                                       # repo settings, "main" ruleset and CI variable
```

Then install the [Octo STS](https://github.com/apps/octo-sts) GitHub App on the repository. No secret is needed: the CI exchanges its OIDC token for a short-lived token (policies in `.github/chainguard/`).

## Quick start

1. **Use this template** on GitHub (or `git clone`).
2. Rename everything with one command:

```bash
scripts/setup.sh -n <app-name> -m github.com/<you>/<repo>   # rewrites module, imports, cmd/ and binary, then smoke-tests
```

3. Run it:

```bash
go test ./...
go build -o bin/<your-app> ./cmd/<your-app>
./bin/<your-app> --help
./bin/<your-app> health --json
```

That's the whole ceremony. Start writing commands.

`scripts/setup.sh` runs `scripts/init-from-template.sh` (module, imports, `cmd/app`, binary, GoReleaser and Containerfile names), then `go mod tidy`, a build and the tests.

## Configuration

Precedence: **flags > env > defaults** (files can slot into `config.Load`
later without changing call sites — deliberately deferred, YAGNI).

Variables are namespaced by binary name: an app named `my-cli` reads:

| Variable              | Values                  | Default |
| --------------------- | ----------------------- | ------- |
| `MY_CLI_LOG_LEVEL`    | debug info warn error   | info    |
| `MY_CLI_LOG_FORMAT`   | text json               | text    |

Add your own fields in `internal/config.Config` following the same pattern.

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
The boundary is enforced by `TestEngineBoundary` (`internal/cli/boundary_test.go`), which runs with
`go test ./...` and therefore in CI, so the engine can be replaced later without touching a single
business command.

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
scripts/setup.sh          # template bootstrap automation (rename module/app)
scripts/setup-repo.sh     # GitHub repo settings, ruleset and CI variable
```

Why not keep it simpler? The indirection costs ~200 lines and buys:
engine portability (req: future non-Cobra engines), hermetic testing
(snapshot isolation lets every test call `app.Run(...)` repeatedly without
process spawning), uniform UX contracts (exit codes, error rendering) that
don't drift when the engine changes.

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
# bin/<app> with stamped version/commit/date (stored in internal/build)
M=github.com/<you>/<repo>
go build -ldflags "-X $M/internal/build.Version=1.2.3 -X $M/internal/build.Commit=$(git rev-parse --short HEAD) -X $M/internal/build.Date=$(date -u +%FT%TZ)" -o bin/<app> ./cmd/<app>

goreleaser release --snapshot --clean      # local snapshot build of all platforms (dist/)
```

Releases are automatic: every code change merged to `main` creates the next semver tag and GitHub
Release (derived from Conventional Commits) and builds the container image (the `pipeline` workflow).
The workflows do **not** run GoReleaser, so releases carry no binaries; `.goreleaser.yaml` is for local
cross-platform builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64: tar.gz archives,
zip on Windows, plus `checksums.txt`).

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

## Principles baked in

KISS · YAGNI · DRY · stdlib-first · explicit dependencies · small interfaces
· separation of concerns. Abstractions here exist because each one removes a
recurring pain (engine lock-in, flaky flag state across tests, drift between
version surfaces). If you cannot name the pain your addition removes, leave
it out.

## Development

```bash
go test -race ./...
go vet ./...
golangci-lint run ./...
```

Install the git hooks once with `lefthook install`: they run formatting, vet, tests (with and without `-race`), build, `go mod tidy`, lint, `govulncheck` and a secrets scan. Commits follow [Conventional Commits](https://www.conventionalcommits.org/).

## CI/CD

| Workflow | Trigger | What it does |
|---|---|---|
| `pr-check` | pull request | shellcheck, merge strategy and Conventional Commits (`merge-check`), Gitleaks, labels and the Go quality gate (module integrity, vet, race tests with coverage, lint, build, dependency review). `pr-gate` aggregates them and is the required check |
| `pipeline` | push to `main` (ignores `.github/**`) or manual | semver guard (blocks an automatic major), immutable tag + GitHub Release, container image |
| `codeql` | nightly or manual | static analysis (CodeQL) |
| `security` | nightly or manual | Gitleaks and Trivy scans |
| `auto-pr` | push to `feat/**` or `fix/**` | opens the pull request automatically |
| `dependabot-actions-auto-merge` | Dependabot pull requests | auto-merges GitHub Actions bumps |

The workflows call reusable workflows from [templates](https://github.com/guilhermelinosp/templates) at `@latest`. Releases need the `HELLNET_ACTIONS_PRIVATE_KEY` secret and the `HELLNET_ACTIONS_CLIENT_ID` variable (set them with `scripts/setup-repo.sh`).

## Contributing and license

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md). Licensed under [Apache 2.0](LICENSE).
