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

var forceInstallFlag bool

var installCmd = &cobra.Command{
	Use:   "install [package_spec...]",
	Short: "Install one or more packages, resolving dependencies and ordering installation",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		api := client.NewAPIClient()

		installedState, err := install.LoadInstalledState()
		if err != nil {
			fmt.Printf("Error loading installed packages state: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Resolving package dependency graph...")
		resolvedGraph, err := graph.ResolveGraph(args, cfg, api)
		if err != nil {
			fmt.Printf("Resolution error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Performing topological sort to compute install order...")
		sortedOrder, err := graph.TopoSort(resolvedGraph, args)
		if err != nil {
			fmt.Printf("Dependency graph error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Resolved install order: %s\n\n", strings.Join(sortedOrder, " -> "))

		for _, pkgName := range sortedOrder {
			node := resolvedGraph[pkgName]
			
			// Check if already installed
			existing, installed := installedState.Packages[pkgName]
			if installed && existing.Version == node.Version && !forceInstallFlag {
				fmt.Printf("Package %s@%s is already installed. Skipping.\n", pkgName, node.Version)
				continue
			}

			fmt.Printf("Installing %s@%s from %s...\n", node.Name, node.Version, node.ServerURL)
			err := install.InstallSinglePackage(node, api, cfg, installedState)
			if err != nil {
				fmt.Printf("Failed to install package %s: %v\n", pkgName, err)
				os.Exit(1)
			}
		}

		fmt.Println("\nAll requested packages installed successfully.")
	},
}

func init() {
	installCmd.Flags().BoolVarP(&forceInstallFlag, "force", "f", false, "Force reinstall even if package is already installed")
	rootCmd.AddCommand(installCmd)
}
