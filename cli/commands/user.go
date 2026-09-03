package commands

import (
	"context"
	"fmt"

	"control-plane/cli/client"
	"control-plane/cli/config"
	"control-plane/cli/printer"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func newUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		Example: `  mygocloud user create --email dev@example.com --name "Developer"
  mygocloud user create -e dev@example.com -n "Developer" -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")
			name, _ := cmd.Flags().GetString("name")

			c, err := client.New(config.Get())
			if err != nil {
				return err
			}

			user, err := c.CreateUser(context.Background(), email, name)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.ErrOrStderr(), "✓ User created successfully")
			return printer.Print(user, getPrinterFormat(), cmd.OutOrStdout())
		},
	}
	createCmd.Flags().StringP("email", "e", "", "User email")
	createCmd.Flags().StringP("name", "n", "", "User name")
	_ = createCmd.MarkFlagRequired("email")
	_ = createCmd.MarkFlagRequired("name")

	getCmd := &cobra.Command{
		Use:   "get <user-id>",
		Short: "Get a user by ID",
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

			user, err := c.GetUser(context.Background(), id)
			if err != nil {
				return err
			}

			return printer.Print(user, getPrinterFormat(), cmd.OutOrStdout())
		},
	}

	cmd.AddCommand(createCmd)
	cmd.AddCommand(getCmd)
	return cmd
}
