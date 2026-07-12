package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/spf13/cobra"
)

var (
	serverFlag   string
	usernameFlag string
	passwordFlag string
	emailFlag    string
	scopesFlag   []string
	tokenFlag    string
	scopeFlag    string
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user on the server",
	Run: func(cmd *cobra.Command, args []string) {
		server, err := getServerURL(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		err = api.CreateUser(server, usernameFlag, passwordFlag, emailFlag)
		if err != nil {
			fmt.Printf("Error creating user: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("User '%s' created successfully on %s\n", usernameFlag, server)
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in and generate a token",
	Run: func(cmd *cobra.Command, args []string) {
		server, err := getServerURL(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if len(scopesFlag) == 0 {
			scopesFlag = []string{"read", "write"}
		}

		api := client.NewAPIClient()
		token, err := api.GenerateToken(server, usernameFlag, passwordFlag, scopesFlag)
		if err != nil {
			fmt.Printf("Login failed: %v\n", err)
			os.Exit(1)
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		sCfg := cfg.Servers[server]
		sCfg.Token = token
		cfg.Servers[server] = sCfg

		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Logged in successfully. Token generated and saved for server %s\n", server)
	},
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage authentication tokens",
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active tokens for the current user",
	Run: func(cmd *cobra.Command, args []string) {
		server, err := getServerURL(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		token, err := getAuthToken(cmd, server)
		if err != nil || token == "" {
			fmt.Printf("Error: not authenticated or token not provided for %s\n", server)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		tokens, err := api.ListTokens(server, token)
		if err != nil {
			fmt.Printf("Error listing tokens: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Active tokens on %s:\n", server)
		fmt.Printf("%-12s %-30s %s\n", "Prefix", "Created At", "Scopes")
		fmt.Println(strings.Repeat("-", 60))
		for _, t := range tokens {
			fmt.Printf("%-12s %-30s %s\n", t.First8Chars, t.CreatedAt, strings.Join(t.Scopes, ", "))
		}
	},
}

var tokenRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke an authentication token",
	Run: func(cmd *cobra.Command, args []string) {
		server, err := getServerURL(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		err = api.RevokeToken(server, tokenFlag, passwordFlag)
		if err != nil {
			fmt.Printf("Error revoking token: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Token revoked successfully.")
	},
}

var tokenValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate an authentication token",
	Run: func(cmd *cobra.Command, args []string) {
		server, err := getServerURL(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		resp, err := api.ValidateToken(server, tokenFlag)
		if err != nil {
			fmt.Printf("Token validation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Token is valid.")
		fmt.Printf("User ID: %s\n", resp.UserID)
		fmt.Printf("Scopes:  %s\n", strings.Join(resp.Scopes, ", "))
	},
}

var scopeCmd = &cobra.Command{
	Use:   "scope",
	Short: "Manage and view scopes",
}

var scopeGrantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Grant a scope to a user (requires admin permission)",
	Run: func(cmd *cobra.Command, args []string) {
		server, err := getServerURL(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		token, err := getAuthToken(cmd, server)
		if err != nil || token == "" {
			fmt.Printf("Error: not authenticated or token not provided for %s\n", server)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		err = api.GrantScope(server, usernameFlag, scopeFlag, token)
		if err != nil {
			fmt.Printf("Error granting scope: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Scope '%s' granted to user '%s' successfully.\n", scopeFlag, usernameFlag)
	},
}

var scopeGetUserCmd = &cobra.Command{
	Use:   "get-user",
	Short: "Get scopes associated with the current user",
	Run: func(cmd *cobra.Command, args []string) {
		server, err := getServerURL(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		token, err := getAuthToken(cmd, server)
		if err != nil || token == "" {
			fmt.Printf("Error: not authenticated or token not provided for %s\n", server)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		scopes, err := api.GetUserScopes(server, token)
		if err != nil {
			fmt.Printf("Error getting user scopes: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("User scopes on %s: %s\n", server, strings.Join(scopes, ", "))
	},
}

var scopeGetTokenCmd = &cobra.Command{
	Use:   "get-token",
	Short: "Get scopes associated with the current token",
	Run: func(cmd *cobra.Command, args []string) {
		server, err := getServerURL(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		token, err := getAuthToken(cmd, server)
		if err != nil || token == "" {
			fmt.Printf("Error: not authenticated or token not provided for %s\n", server)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		scopes, err := api.GetTokenScopes(server, token)
		if err != nil {
			fmt.Printf("Error getting token scopes: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Token scopes on %s: %s\n", server, strings.Join(scopes, ", "))
	},
}

func getServerURL(cmd *cobra.Command) (string, error) {
	val, err := cmd.Flags().GetString("server")
	if err == nil && val != "" {
		return val, nil
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", err
	}
	return cfg.DefaultServer, nil
}

func getAuthToken(cmd *cobra.Command, server string) (string, error) {
	val, err := cmd.Flags().GetString("auth-token")
	if err == nil && val != "" {
		return val, nil
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", err
	}
	return cfg.Servers[server].Token, nil
}

func init() {
	// Root flags for server override
	userCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "Aluminium server URL (overrides default)")
	loginCmd.Flags().StringVar(&serverFlag, "server", "", "Aluminium server URL (overrides default)")
	tokenCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "Aluminium server URL (overrides default)")
	scopeCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "Aluminium server URL (overrides default)")

	tokenCmd.PersistentFlags().StringVar(&tokenFlag, "auth-token", "", "Token to use for request (overrides saved token)")
	scopeCmd.PersistentFlags().StringVar(&tokenFlag, "auth-token", "", "Token to use for request (overrides saved token)")

	// Create user flags
	userCreateCmd.Flags().StringVarP(&usernameFlag, "username", "u", "", "Username")
	userCreateCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "Password")
	userCreateCmd.Flags().StringVarP(&emailFlag, "email", "e", "", "Email address")
	_ = userCreateCmd.MarkFlagRequired("username")
	_ = userCreateCmd.MarkFlagRequired("password")
	_ = userCreateCmd.MarkFlagRequired("email")
	userCmd.AddCommand(userCreateCmd)

	// Login flags
	loginCmd.Flags().StringVarP(&usernameFlag, "username", "u", "", "Username")
	loginCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "Password")
	loginCmd.Flags().StringSliceVar(&scopesFlag, "scopes", []string{}, "Comma-separated list of scopes to request (e.g. read,write)")
	_ = loginCmd.MarkFlagRequired("username")
	_ = loginCmd.MarkFlagRequired("password")

	// Revoke flags
	tokenRevokeCmd.Flags().StringVar(&tokenFlag, "token", "", "Token to revoke")
	tokenRevokeCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "User password")
	_ = tokenRevokeCmd.MarkFlagRequired("token")
	_ = tokenRevokeCmd.MarkFlagRequired("password")

	// Validate flags
	tokenValidateCmd.Flags().StringVar(&tokenFlag, "token", "", "Token to validate")
	_ = tokenValidateCmd.MarkFlagRequired("token")

	tokenCmd.AddCommand(tokenListCmd)
	tokenCmd.AddCommand(tokenRevokeCmd)
	tokenCmd.AddCommand(tokenValidateCmd)

	// Scope grant flags
	scopeGrantCmd.Flags().StringVarP(&usernameFlag, "username", "u", "", "Username of the target user")
	scopeGrantCmd.Flags().StringVarP(&scopeFlag, "scope", "s", "", "Scope to grant ('read', 'write', 'admin', 'dev')")
	_ = scopeGrantCmd.MarkFlagRequired("username")
	_ = scopeGrantCmd.MarkFlagRequired("scope")

	scopeCmd.AddCommand(scopeGrantCmd)
	scopeCmd.AddCommand(scopeGetUserCmd)
	scopeCmd.AddCommand(scopeGetTokenCmd)

	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(tokenCmd)
	rootCmd.AddCommand(scopeCmd)
}
