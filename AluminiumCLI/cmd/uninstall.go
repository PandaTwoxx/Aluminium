package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/PandaTwoxx/Aluminium/internal/graph"
	"github.com/PandaTwoxx/Aluminium/internal/install"
	"github.com/spf13/cobra"
)

var forceUninstallFlag bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [package_spec...]",
	Short: "Uninstall one or more installed packages",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()

		state, err := install.LoadInstalledState()
		if err != nil {
			fmt.Printf("Error loading installed packages state: %v\n", err)
			os.Exit(1)
		}

		uninstallSet := make(map[string]bool)
		for _, spec := range args {
			name, version := graph.ParseSpec(spec)
			installed, ok := state.Packages[name]
			if !ok {
				fmt.Printf("Error: package %s is not installed.\n", name)
				os.Exit(1)
			}
			if version != "" && installed.Version != version {
				fmt.Printf("Error: package %s@%s is not installed (installed version: %s).\n", name, version, installed.Version)
				os.Exit(1)
			}
			uninstallSet[name] = true
		}

		dependents, err := install.FindDependents(uninstallSet, state, cfg, api)
		if err != nil {
			fmt.Printf("Error checking package dependencies: %v\n", err)
			os.Exit(1)
		}
		if len(dependents) > 0 && !forceUninstallFlag {
			fmt.Printf("Error: cannot uninstall because these installed packages depend on the requested package(s): %s\n", strings.Join(dependents, ", "))
			fmt.Println("Uninstall those packages first, or use --force.")
			os.Exit(1)
		}

		uninstallOrder := make([]string, 0, len(uninstallSet))
		for name := range uninstallSet {
			uninstallOrder = append(uninstallOrder, name)
		}

		if resolvedGraph, resolveErr := graph.ResolveGraph(args, cfg, api); resolveErr == nil {
			if order, sortErr := graph.ReverseTopoSort(resolvedGraph, args); sortErr == nil {
				filtered := make([]string, 0, len(uninstallSet))
				seen := make(map[string]bool)
				for _, name := range order {
					if !uninstallSet[name] || seen[name] {
						continue
					}
					filtered = append(filtered, name)
					seen[name] = true
				}
				if len(filtered) == len(uninstallSet) {
					uninstallOrder = filtered
				}
			}
		}

		fmt.Printf("Uninstall order: %s\n\n", strings.Join(uninstallOrder, " -> "))

		for _, pkgName := range uninstallOrder {
			if err := install.UninstallSinglePackage(pkgName, cfg, api, state); err != nil {
				fmt.Printf("Failed to uninstall package %s: %v\n", pkgName, err)
				os.Exit(1)
			}
		}

		fmt.Println("\nAll requested packages uninstalled successfully.")
	},
}

func init() {
	uninstallCmd.Flags().BoolVarP(&forceUninstallFlag, "force", "f", false, "Uninstall even if other installed packages depend on these packages")
	rootCmd.AddCommand(uninstallCmd)
}
