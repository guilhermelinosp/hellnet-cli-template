package main

import (
	"context"
	"encoding/json"

	"github.com/guilhermelinosp/hellnet-cli-template/internal/cli"
	"github.com/guilhermelinosp/hellnet-cli-template/internal/service/health"
)

// newHealthCommand is the canonical example of adding a command: declare the
// struct, inject the service through the constructor, read flags inside Run.
// Copy this shape for your own business commands.
func newHealthCommand(svc *health.Service) *cli.Command {
	return &cli.Command{
		Name:  "health",
		Short: "Check process health and report diagnostics",
		Long: "Runs the liveness checks implemented by the health service.\n" +
			"Useful for orchestrators, smoke tests and CI pipelines.",
		Example: "# human-readable\n" +
			"  $ app health\n\n" +
			"# machine-readable (single line)\n" +
			"  $ app health --json",
		Flags: []*cli.Flag{
			cli.BoolFlag("json", "j", false, "emit single-line JSON instead of text"),
		},
		Args: cli.NoArgs,
		Run: func(ctx context.Context, c *cli.Command, _ []string) error {
			report := svc.Check(ctx)

			if c.Bool("json") {
				data, err := json.Marshal(report)
				if err != nil {
					return cli.Errorf("marshal health report: %w", err)
				}
				c.Printf("%s\n", data)
				return nil
			}

			c.Printf("status:   %s\n", report.Status)
			c.Printf("uptime:   %s\n", report.UptimeText)
			c.Printf("version:  %s\n", report.Version)
			return nil
		},
	}
}
