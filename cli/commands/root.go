package commands

import (
	"control-plane/cli/config"
	"control-plane/cli/printer"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	outputFormat string
	cfgFilePath  string
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "mygocloud",
		Short: "MyGoCloud CLI — Control plane for application deployments",
		Long: `MyGoCloud CLI is the command-line interface for managing applications,
deployments, and infrastructure through the control-plane API.

Examples:
  mygocloud login --endpoint http://localhost:8080
  mygocloud user create --email dev@example.com --name "Developer"
  mygocloud app create --name api-gateway --owner-id <uuid>
  mygocloud app deploy <app-id> --version 1.0.0
  mygocloud app status <app-id> --watch`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Init(cfgFilePath); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format: table, json, yaml")
	root.PersistentFlags().StringVar(&cfgFilePath, "config", "", "Config file path (default ~/.mygocloud/config.yaml)")

	root.AddCommand(newLoginCommand())
	root.AddCommand(newUserCommand())
	root.AddCommand(newAppCommand())
	root.AddCommand(newVisualizeCommand())

	return root
}

func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func getPrinterFormat() printer.Format {
	switch outputFormat {
	case "json":
		return printer.FormatJSON
	case "yaml":
		return printer.FormatYAML
	default:
		return printer.FormatTable
	}
}
