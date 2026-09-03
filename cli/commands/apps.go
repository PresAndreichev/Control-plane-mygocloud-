package commands

import (
	"context"
	"fmt"

	"control-plane/cli/client"
	"control-plane/cli/config"
	"control-plane/cli/printer"
	"control-plane/internal/domain"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func newAppCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "app",
		Aliases: []string{"application", "a"},
		Short:   "Manage applications and deployments",
	}

	cmd.AddCommand(newAppCreateCommand())
	cmd.AddCommand(newAppListCommand())
	cmd.AddCommand(newAppGetCommand())
	cmd.AddCommand(newAppDeployCommand())
	cmd.AddCommand(newAppDeploymentsCommand())
	cmd.AddCommand(newAppRollbackCommand())

	return cmd
}

func newAppCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new application",
		Example: `  mygocloud app create --name api-gateway --description "Edge proxy" --owner-id <uuid>
  mygocloud app create -n api-gateway -d "Edge proxy" -o <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			desc, _ := cmd.Flags().GetString("description")
			image, _ := cmd.Flags().GetString("image")
			ownerIDStr, _ := cmd.Flags().GetString("owner-id")

			ownerID, err := uuid.Parse(ownerIDStr)
			if err != nil {
				return fmt.Errorf("invalid owner-id UUID: %w", err)
			}

			c, err := client.New(config.Get())
			if err != nil {
				return err
			}

			app, err := c.CreateApplication(context.Background(), name, desc, image, ownerID)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.ErrOrStderr(), "✓ Application created successfully")
			return printer.Print(app, getPrinterFormat(), cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringP("name", "n", "", "Application name")
	cmd.Flags().StringP("description", "d", "", "Application description")
	cmd.Flags().StringP("owner-id", "o", "", "Owner user ID (UUID)")
	cmd.Flags().StringP("image", "i", "", "Container image to deploy")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("owner-id")

	return cmd
}

func newAppListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all applications",
		RunE: func(cmd *cobra.Command, args []string) error {
			ownerIDStr, _ := cmd.Flags().GetString("owner-id")

			c, err := client.New(config.Get())
			if err != nil {
				return err
			}

			var apps []domain.Application
			if ownerIDStr != "" {
				ownerID, err := uuid.Parse(ownerIDStr)
				if err != nil {
					return fmt.Errorf("invalid owner-id UUID: %w", err)
				}
				apps, err = c.ListApplicationsByOwner(context.Background(), ownerID)
			} else {
				apps, err = c.ListApplications(context.Background())
			}
			if err != nil {
				return err
			}

			return printer.Print(apps, getPrinterFormat(), cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringP("owner-id", "o", "", "Filter by owner user ID")
	return cmd
}

func newAppGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <app-id>",
		Short: "Get an application by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid UUID: %w", err)
			}

			c, err := client.New(config.Get())
			if err != nil {
				return err
			}

			app, err := c.GetApplication(context.Background(), id)
			if err != nil {
				return err
			}

			return printer.Print(app, getPrinterFormat(), cmd.OutOrStdout())
		},
	}
}

func newAppDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy <app-id>",
		Short: "Deploy a version to an application",
		Example: `  mygocloud app deploy <app-id> --version 1.0.0
  mygocloud app deploy <app-id> -v 1.0.0`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid app-id UUID: %w", err)
			}

			version, _ := cmd.Flags().GetString("version")

			c, err := client.New(config.Get())
			if err != nil {
				return err
			}

			dep, err := c.Deploy(context.Background(), appID, version)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.ErrOrStderr(), "✓ Deployment queued")
			return printer.Print(dep, getPrinterFormat(), cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringP("version", "v", "", "Version to deploy")
	_ = cmd.MarkFlagRequired("version")

	return cmd
}

func newAppDeploymentsCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "deployments <app-id>",
		Aliases: []string{"deps", "history"},
		Short:   "List deployment history for an application",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid app-id UUID: %w", err)
			}

			c, err := client.New(config.Get())
			if err != nil {
				return err
			}

			deps, err := c.ListDeployments(context.Background(), appID)
			if err != nil {
				return err
			}

			return printer.Print(deps, getPrinterFormat(), cmd.OutOrStdout())
		},
	}
}

func newAppRollbackCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback <app-id>",
		Short: "Rollback to the last successful deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid app-id UUID: %w", err)
			}

			c, err := client.New(config.Get())
			if err != nil {
				return err
			}

			dep, err := c.Rollback(context.Background(), appID)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.ErrOrStderr(), "✓ Rollback queued")
			return printer.Print(dep, getPrinterFormat(), cmd.OutOrStdout())
		},
	}
}
