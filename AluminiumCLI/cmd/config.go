package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local Aluminium CLI configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		bytes, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			fmt.Printf("Error marshalling config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(bytes))
	},
}

var configSetDefaultCmd = &cobra.Command{
	Use:   "set-default-server [url]",
	Short: "Set the default source server URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		cfg.DefaultServer = args[0]
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Default server set to: %s\n", args[0])
	},
}

var configAddServerCmd = &cobra.Command{
	Use:   "add-server [url]",
	Short: "Add a server to the package search servers list",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		url := args[0]
		for _, s := range cfg.SearchServers {
			if s == url {
				fmt.Printf("Server %s is already in search path.\n", url)
				return
			}
		}
		cfg.SearchServers = append(cfg.SearchServers, url)
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added server %s to search servers.\n", url)
	},
}

var configRemoveServerCmd = &cobra.Command{
	Use:   "remove-server [url]",
	Short: "Remove a server from the package search servers list",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		url := args[0]
		foundIdx := -1
		for i, s := range cfg.SearchServers {
			if s == url {
				foundIdx = i
				break
			}
		}
		if foundIdx == -1 {
			fmt.Printf("Server %s not found in search path.\n", url)
			return
		}
		cfg.SearchServers = append(cfg.SearchServers[:foundIdx], cfg.SearchServers[foundIdx+1:]...)
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed server %s from search servers.\n", url)
	},
}

var configSetTokenCmd = &cobra.Command{
	Use:   "set-token [url] [token]",
	Short: "Set authentication token for a server URL",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		url := args[0]
		token := args[1]

		sCfg := cfg.Servers[url]
		sCfg.Token = token
		cfg.Servers[url] = sCfg

		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Token saved for server: %s\n", url)
	},
}

var configSetInteractiveCmd = &cobra.Command{
	Use:   "set-interactive [true|false]",
	Short: "Enable or disable interactive mode for the CLI",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		switch args[0] {
		case "true", "1", "yes", "on":
			cfg.Interactive = true
		case "false", "0", "no", "off":
			cfg.Interactive = false
		default:
			fmt.Printf("Invalid value %q: use true or false\n", args[0])
			os.Exit(1)
		}
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Interactive mode set to: %v\n", cfg.Interactive)
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetDefaultCmd)
	configCmd.AddCommand(configAddServerCmd)
	configCmd.AddCommand(configRemoveServerCmd)
	configCmd.AddCommand(configSetTokenCmd)
	configCmd.AddCommand(configSetInteractiveCmd)
	rootCmd.AddCommand(configCmd)
}
