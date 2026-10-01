// Command app is the template's entry point. Keep this file thin: resolve
// configuration, construct dependencies explicitly, hand them to commands,
// delegate everything else to internal/cli.
package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/guilhermelinosp/hellnet-cli-template/internal/cli/cobra" // activate CLI engine (swap implementations by changing this import)

	"github.com/guilhermelinosp/hellnet-cli-template/internal/build"
	"github.com/guilhermelinosp/hellnet-cli-template/internal/cli"
	"github.com/guilhermelinosp/hellnet-cli-template/internal/config"
	"github.com/guilhermelinosp/hellnet-cli-template/internal/logging"
	"github.com/guilhermelinosp/hellnet-cli-template/internal/service/health"
)

// startedAt anchors uptime metrics reported by `health`.
var startedAt = time.Now()

func main() {
	os.Exit(run(context.Background()))
}

// run is separated from main so integration tests exercise identical wiring.
// Signature returns the process exit code; os.Exit stays out of tests.
func run(ctx context.Context) int {
	cfg := config.Load(envPrefix())

	logger := logging.New(logging.Options{
		Service: filepath.Base(os.Args[0]),
		Version: build.Version,
		Level:   cfg.LogLevel,
		Format:  cfg.LogFormat,
	})

	// Cancellation context wired to SIGINT/SIGTERM — every command receives
	// ctx and must honor its Done() to support graceful shutdown.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := cli.New(cli.Options{
		Name:    filepath.Base(os.Args[0]),
		Version: build.Version,
		Long:    "Production-ready CLI scaffold generated from hellnet-cli-template.\nReplace this description with what your tool does.",
	})
	app.Add(
		newHealthCommand(health.New(startedAt, build.Version, logger)),
	)
	return app.Run(ctx, os.Args[1:])
}

// envPrefix namespaces configuration variables: a binary named "my-cli"
// reads MY_CLI_LOG_LEVEL / MY_CLI_LOG_FORMAT.
func envPrefix() string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe"), "-", "_"))
}
