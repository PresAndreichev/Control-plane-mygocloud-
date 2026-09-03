package commands

import (
	"fmt"

	"control-plane/cli/config"

	"github.com/spf13/cobra"
)

func newLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Configure the API endpoint",
		Long:  "Sets the control-plane API endpoint and saves it to the config file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			endpoint, _ := cmd.Flags().GetString("endpoint")
			cfg := &config.Config{
				Endpoint: endpoint,
			}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "✓ Configured API endpoint: %s\n", endpoint)
			fmt.Fprintf(cmd.ErrOrStderr(), "  Config saved to: %s\n", config.CfgFile())
			return nil
		},
	}

	cmd.Flags().StringP("endpoint", "e", "http://localhost:8080", "Control-plane API endpoint")
	_ = cmd.MarkFlagRequired("endpoint")

	return cmd
}
