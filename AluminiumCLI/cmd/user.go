package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/PandaTwoxx/Aluminium/internal/prompt"
	"github.com/fatih/color"
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
		cfg, _ := config.LoadConfig()
		if prompt.IsInteractive(cmd, cfg) {
			input, err := prompt.UserCreate(prompt.UserCreateInput{
				Username: usernameFlag,
				Password: passwordFlag,
				Email:    emailFlag,
			})
			if err != nil {
				color.Red("Error: %v\n", err)
				os.Exit(1)
			}
			usernameFlag = input.Username
			passwordFlag = input.Password
			emailFlag = input.Email
		} else {
			var missing []string
			if usernameFlag == "" { missing = append(missing, "username") }
			if passwordFlag == "" { missing = append(missing, "password") }
			if emailFlag == "" { missing = append(missing, "email") }
			if len(missing) > 0 {
				color.Red("Error: required flag(s) %q not set\n", missing)
				os.Exit(1)
			}
		}

		server, err := getServerURL(cmd)
		if err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		err = api.CreateUser(server, usernameFlag, passwordFlag, emailFlag)
		if err != nil {
			color.Red("Error creating user: %v\n", err)
			os.Exit(1)
		}
		color.Green("User '%s' created successfully on %s\n", usernameFlag, server)
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a user account from the server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.LoadConfig()
		if prompt.IsInteractive(cmd, cfg) {
			input, err := prompt.Login(prompt.LoginInput{
				Username: usernameFlag,
				Password: passwordFlag,
			})
			if err != nil {
				color.Red("Error: %v\n", err)
				os.Exit(1)
			}
			usernameFlag = input.Username
			passwordFlag = input.Password
		} else {
			var missing []string
			if usernameFlag == "" { missing = append(missing, "username") }
			if passwordFlag == "" { missing = append(missing, "password") }
			if len(missing) > 0 {
				color.Red("Error: required flag(s) %q not set\n", missing)
				os.Exit(1)
			}
		}

		server, err := getServerURL(cmd)
		if err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		err = api.DeleteUser(server, usernameFlag, passwordFlag)
		if err != nil {
			color.Red("Error deleting user: %v\n", err)
			os.Exit(1)
		}
		color.Green("User '%s' deleted successfully from %s\n", usernameFlag, server)
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in and generate a token",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.LoadConfig()
		if prompt.IsInteractive(cmd, cfg) {
			input, err := prompt.Login(prompt.LoginInput{
				Username: usernameFlag,
				Password: passwordFlag,
			})
			if err != nil {
				color.Red("Error: %v\n", err)
				os.Exit(1)
			}
			usernameFlag = input.Username
			passwordFlag = input.Password
		} else {
			var missing []string
			if usernameFlag == "" { missing = append(missing, "username") }
			if passwordFlag == "" { missing = append(missing, "password") }
			if len(missing) > 0 {
				color.Red("Error: required flag(s) %q not set\n", missing)
				os.Exit(1)
			}
		}

		server, err := getServerURL(cmd)
		if err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()

		var selectedScopes []string
		if prompt.IsInteractive(cmd, cfg) {
			// Generate temporary token to fetch user scopes
			tempToken, err := api.GenerateToken(server, usernameFlag, passwordFlag, nil)
			if err != nil {
				color.Red("Login failed: %v\n", err)
				os.Exit(1)
			}
			userScopes, err := api.GetTokenScopes(server, tempToken)
			if err != nil {
				_ = api.RevokeToken(server, tempToken, passwordFlag)
				color.Red("Failed to fetch user scopes: %v\n", err)
				os.Exit(1)
			}
			_ = api.RevokeToken(server, tempToken, passwordFlag)

			chosen, err := prompt.ChooseScopes(userScopes)
			if err != nil {
				color.Red("Error: %v\n", err)
				os.Exit(1)
			}
			selectedScopes = chosen
		} else {
			selectedScopes = scopesFlag
		}

		token, err := api.GenerateToken(server, usernameFlag, passwordFlag, selectedScopes)
		if err != nil {
			color.Red("Login failed: %v\n", err)
			os.Exit(1)
		}

		if cfg == nil {
			color.Red("Error loading config\n")
			os.Exit(1)
		}

		if cfg.Servers == nil {
			cfg.Servers = make(map[string]config.ServerConfig)
		}

		sCfg := cfg.Servers[server]
		sCfg.Token = token
		cfg.Servers[server] = sCfg

		if err := config.SaveConfig(cfg); err != nil {
			color.Red("Error saving config: %v\n", err)
			os.Exit(1)
		}

		color.Green("Logged in successfully. Token generated and saved for server %s\n", server)
	},
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new authentication token",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.LoadConfig()
		if prompt.IsInteractive(cmd, cfg) {
			input, err := prompt.Login(prompt.LoginInput{
				Username: usernameFlag,
				Password: passwordFlag,
			})
			if err != nil {
				color.Red("Error: %v\n", err)
				os.Exit(1)
			}
			usernameFlag = input.Username
			passwordFlag = input.Password
		} else {
			var missing []string
			if usernameFlag == "" { missing = append(missing, "username") }
			if passwordFlag == "" { missing = append(missing, "password") }
			if len(missing) > 0 {
				color.Red("Error: required flag(s) %q not set\n", missing)
				os.Exit(1)
			}
		}

		server, err := getServerURL(cmd)
		if err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()

		var selectedScopes []string
		if prompt.IsInteractive(cmd, cfg) {
			// Generate temporary token to fetch user scopes
			tempToken, err := api.GenerateToken(server, usernameFlag, passwordFlag, nil)
			if err != nil {
				color.Red("Login failed: %v\n", err)
				os.Exit(1)
			}
			userScopes, err := api.GetTokenScopes(server, tempToken)
			if err != nil {
				_ = api.RevokeToken(server, tempToken, passwordFlag)
				color.Red("Failed to fetch user scopes: %v\n", err)
				os.Exit(1)
			}
			_ = api.RevokeToken(server, tempToken, passwordFlag)

			chosen, err := prompt.ChooseScopes(userScopes)
			if err != nil {
				color.Red("Error: %v\n", err)
				os.Exit(1)
			}
			selectedScopes = chosen
		} else {
			selectedScopes = scopesFlag
		}

		token, err := api.GenerateToken(server, usernameFlag, passwordFlag, selectedScopes)
		if err != nil {
			color.Red("Failed to create token: %v\n", err)
			os.Exit(1)
		}

		color.Green("Token generated successfully: %s\n", token)
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
		cfg, _ := config.LoadConfig()
		if prompt.IsInteractive(cmd, cfg) {
			input, err := prompt.TokenRevoke(prompt.TokenRevokeInput{
				Token:    tokenFlag,
				Password: passwordFlag,
			})
			if err != nil {
				color.Red("Error: %v\n", err)
				os.Exit(1)
			}
			tokenFlag = input.Token
			passwordFlag = input.Password
		} else {
			var missing []string
			if tokenFlag == "" { missing = append(missing, "token") }
			if passwordFlag == "" { missing = append(missing, "password") }
			if len(missing) > 0 {
				color.Red("Error: required flag(s) %q not set\n", missing)
				os.Exit(1)
			}
		}

		server, err := getServerURL(cmd)
		if err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		err = api.RevokeToken(server, tokenFlag, passwordFlag)
		if err != nil {
			color.Red("Error revoking token: %v\n", err)
			os.Exit(1)
		}

		color.Green("Token revoked successfully.\n")
	},
}

var tokenValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate an authentication token",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.LoadConfig()
		if prompt.IsInteractive(cmd, cfg) {
			input, err := prompt.TokenValidate(prompt.TokenValidateInput{
				Token: tokenFlag,
			})
			if err != nil {
				color.Red("Error: %v\n", err)
				os.Exit(1)
			}
			tokenFlag = input.Token
		} else {
			var missing []string
			if tokenFlag == "" { missing = append(missing, "token") }
			if len(missing) > 0 {
				color.Red("Error: required flag(s) %q not set\n", missing)
				os.Exit(1)
			}
		}

		server, err := getServerURL(cmd)
		if err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		resp, err := api.ValidateToken(server, tokenFlag)
		if err != nil {
			color.Red("Token validation failed: %v\n", err)
			os.Exit(1)
		}

		color.Green("Token is valid.\n")
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
		cfg, _ := config.LoadConfig()
		if prompt.IsInteractive(cmd, cfg) {
			input, err := prompt.ScopeGrant(prompt.ScopeGrantInput{
				Username: usernameFlag,
				Scope:    scopeFlag,
			})
			if err != nil {
				color.Red("Error: %v\n", err)
				os.Exit(1)
			}
			usernameFlag = input.Username
			scopeFlag = input.Scope
		} else {
			var missing []string
			if usernameFlag == "" { missing = append(missing, "username") }
			if scopeFlag == "" { missing = append(missing, "scope") }
			if len(missing) > 0 {
				color.Red("Error: required flag(s) %q not set\n", missing)
				os.Exit(1)
			}
		}

		server, err := getServerURL(cmd)
		if err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}
		token, err := getAuthToken(cmd, server)
		if err != nil || token == "" {
			color.Red("Error: not authenticated or token not provided for %s\n", server)
			os.Exit(1)
		}

		api := client.NewAPIClient()
		err = api.GrantScope(server, usernameFlag, scopeFlag, token)
		if err != nil {
			color.Red("Error granting scope: %v\n", err)
			os.Exit(1)
		}
		color.Green("Scope '%s' granted to user '%s' successfully.\n", scopeFlag, usernameFlag)
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
	userCmd.AddCommand(userCreateCmd)

	// Delete user flags
	userDeleteCmd.Flags().StringVarP(&usernameFlag, "username", "u", "", "Username")
	userDeleteCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "Password")
	userCmd.AddCommand(userDeleteCmd)

	// Login flags
	loginCmd.Flags().StringVarP(&usernameFlag, "username", "u", "", "Username")
	loginCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "Password")
	loginCmd.Flags().StringSliceVar(&scopesFlag, "scopes", []string{}, "Comma-separated list of scopes to request (e.g. read,write)")

	// Revoke flags
	tokenRevokeCmd.Flags().StringVar(&tokenFlag, "token", "", "Token to revoke")
	tokenRevokeCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "User password")

	// Validate flags
	tokenValidateCmd.Flags().StringVar(&tokenFlag, "token", "", "Token to validate")

	tokenCmd.AddCommand(tokenListCmd)
	tokenCmd.AddCommand(tokenRevokeCmd)
	tokenCmd.AddCommand(tokenValidateCmd)
	tokenCmd.AddCommand(tokenCreateCmd)

	// Token Create flags
	tokenCreateCmd.Flags().StringVarP(&usernameFlag, "username", "u", "", "Username")
	tokenCreateCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "Password")
	tokenCreateCmd.Flags().StringSliceVar(&scopesFlag, "scopes", []string{}, "Comma-separated list of scopes to request (e.g. read,write)")

	// Scope grant flags
	scopeGrantCmd.Flags().StringVarP(&usernameFlag, "username", "u", "", "Username of the target user")
	scopeGrantCmd.Flags().StringVarP(&scopeFlag, "scope", "s", "", "Scope to grant ('read', 'write', 'admin', 'dev')")

	scopeCmd.AddCommand(scopeGrantCmd)
	scopeCmd.AddCommand(scopeGetUserCmd)
	scopeCmd.AddCommand(scopeGetTokenCmd)

	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(tokenCmd)
	rootCmd.AddCommand(scopeCmd)
}
