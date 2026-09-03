package commands

import (
	"context"
	"fmt"
	"time"

	"control-plane/cli/client"
	"control-plane/cli/config"
	"control-plane/cli/printer"
	"control-plane/internal/domain"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func newVisualizeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "viz",
		Short: "Visualize application status and infrastructure",
		Long: `Visualization commands for monitoring deployments and infrastructure.
These commands will be extended in future phases to include:
- Real-time container logs from Docker/Kubernetes
- Infrastructure topology graphs
- Resource usage metrics`,
	}

	cmd.AddCommand(newAppStatusCommand())
	cmd.AddCommand(newAppLogsCommand())
	cmd.AddCommand(newInfraPreviewCommand())

	return cmd
}

func newAppStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <app-id>",
		Short: "Watch deployment status in real-time",
		Long: `Polls the API every 2 seconds and displays the current deployment status.
Useful for watching pending → running → successful transitions.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid app-id UUID: %w", err)
			}

			watch, _ := cmd.Flags().GetBool("watch")
			interval, _ := cmd.Flags().GetDuration("interval")

			c, err := client.New(config.Get())
			if err != nil {
				return err
			}
			ctx := context.Background()

			if !watch {
				deps, err := c.ListDeployments(ctx, appID)
				if err != nil {
					return err
				}
				return printer.Print(deps, getPrinterFormat(), cmd.OutOrStdout())
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Watching deployments for %s (interval: %s)...\n", appID, interval)
			fmt.Fprintln(cmd.ErrOrStderr(), "Press Ctrl+C to stop.")

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				deps, err := c.ListDeployments(ctx, appID)
				if err != nil {
					return err
				}
				fmt.Fprint(cmd.OutOrStdout(), "\033[H\033[2J")
				fmt.Fprintf(cmd.OutOrStdout(), "Application: %s | Time: %s\n\n", appID, time.Now().Format("15:04:05"))

				if len(deps) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No deployments yet.")
				} else {
					printer.Print(deps, printer.FormatTable, cmd.OutOrStdout())
				}

				<-ticker.C
			}
		},
	}

	cmd.Flags().BoolP("watch", "w", false, "Continuously poll status")
	cmd.Flags().DurationP("interval", "i", 2*time.Second, "Poll interval")

	return cmd
}

func newAppLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <app-id>",
		Short: "Stream application container logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid app-id UUID: %w", err)
			}

			tail, _ := cmd.Flags().GetInt("tail")
			follow, _ := cmd.Flags().GetBool("follow")

			c, err := client.New(config.Get())
			if err != nil {
				return err
			}

			// Get latest deployment with container
			deps, err := c.ListDeployments(context.Background(), appID)
			if err != nil {
				return err
			}

			var target *domain.Deployment
			for i := range deps {
				if deps[i].ContainerID != "" {
					target = &deps[i]
					break // most recent with container
				}
			}
			if target == nil {
				return fmt.Errorf("no deployment with a running container found")
			}

			if follow {
				fmt.Fprintf(cmd.ErrOrStderr(), "Streaming logs for container %s...\n", target.ContainerID[:12])
				// Real streaming would need a WebSocket or SSE endpoint on API
				// For now, poll every 2s
				for {
					logs, err := c.GetLogs(context.Background(), target.ContainerID, tail)
					if err != nil {
						return err
					}
					cmd.OutOrStdout().Write([]byte(logs))
					time.Sleep(2 * time.Second)
				}
			}

			logs, err := c.GetLogs(context.Background(), target.ContainerID, tail)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), logs)
			return nil
		},
	}

	cmd.Flags().IntP("tail", "t", 100, "Number of log lines to show")
	cmd.Flags().BoolP("follow", "f", false, "Continuously stream logs")

	return cmd
}

func newInfraPreviewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "infra",
		Short: "Preview infrastructure topology [INFRASTRUCTURE PHASE]",
		Long: `Placeholder for visualizing infrastructure topology.
In the infrastructure phase, this will display:
- Running containers/pods
- Service mesh connections
- Load balancer status
- Resource allocation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "┌─────────────────────────────────────────┐")
			fmt.Fprintln(cmd.OutOrStdout(), "│     Infrastructure Topology Preview     │")
			fmt.Fprintln(cmd.OutOrStdout(), "├─────────────────────────────────────────┤")
			fmt.Fprintln(cmd.OutOrStdout(), "│                                         │")
			fmt.Fprintln(cmd.OutOrStdout(), "│    [CLI] ──► [API Server] ──► [DB]     │")
			fmt.Fprintln(cmd.OutOrStdout(), "│                  │                      │")
			fmt.Fprintln(cmd.OutOrStdout(), "│                  ▼                      │")
			fmt.Fprintln(cmd.OutOrStdout(), "│         [Docker/K8s Worker]            │")
			fmt.Fprintln(cmd.OutOrStdout(), "│                  │                      │")
			fmt.Fprintln(cmd.OutOrStdout(), "│         [Running Containers]           │")
			fmt.Fprintln(cmd.OutOrStdout(), "│                                         │")
			fmt.Fprintln(cmd.OutOrStdout(), "└─────────────────────────────────────────┘")
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "⚠  Infrastructure visualization not yet implemented.")
			fmt.Fprintln(cmd.OutOrStdout(), "   Phase 3 will add Docker/Kubernetes integration.")
			return nil
		},
	}
}
