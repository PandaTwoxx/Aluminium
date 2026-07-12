package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/PandaTwoxx/Aluminium/internal/graph"
	"github.com/PandaTwoxx/Aluminium/internal/install"
	"github.com/spf13/cobra"
)

// ─── aluminium update ────────────────────────────────────────────────────────

var alumUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Aluminium CLI to the latest release and check all source servers",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		// 1. Check all source servers
		fmt.Println("Checking source server connectivity...")
		httpClient := &http.Client{Timeout: 8 * time.Second}
		allOK := true
		for _, server := range cfg.SearchServers {
			resp, err := httpClient.Get(server + "/")
			if err != nil {
				fmt.Printf("  ✗ %s — unreachable (%v)\n", server, err)
				allOK = false
				continue
			}
			resp.Body.Close()
			fmt.Printf("  ✓ %s — reachable (HTTP %d)\n", server, resp.StatusCode)
		}
		if !allOK {
			fmt.Println("\nWarning: one or more servers are unreachable. Continuing with update anyway.")
		} else {
			fmt.Println("All servers reachable.")
		}

		// 2. Re-run the install.sh from GitHub (the README's update command)
		installURL := "https://github.com/PandaTwoxx/Aluminium/raw/refs/heads/main/install.sh"
		fmt.Printf("\nDownloading and running install script from %s...\n", installURL)

		var shellCmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			fmt.Println("Error: self-update via install.sh is not supported on Windows. Please download the latest binary manually from GitHub Releases.")
			os.Exit(1)
		default:
			shellCmd = exec.Command("/bin/bash", "-c",
				fmt.Sprintf(`/bin/bash -c "$(curl -fsSL '%s')"`, installURL))
		}

		shellCmd.Stdout = os.Stdout
		shellCmd.Stderr = os.Stderr
		if err := shellCmd.Run(); err != nil {
			fmt.Printf("Update failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\nAluminium CLI updated successfully.")
	},
}

// ─── aluminium upgrade ───────────────────────────────────────────────────────

var upgradeAllFlag bool

var alumUpgradeCmd = &cobra.Command{
	Use:   "upgrade [package...]",
	Short: "Upgrade installed packages to the latest available version",
	Long: `Upgrade checks all configured servers for newer versions of your installed packages,
validates the updated dependency graph for cycles, then installs any available upgrades.

Without arguments, all installed packages are upgraded. With arguments, only the
specified packages (and their dependencies) are considered.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		state, err := install.LoadInstalledState()
		if err != nil {
			fmt.Printf("Error loading installed state: %v\n", err)
			os.Exit(1)
		}

		if len(state.Packages) == 0 {
			fmt.Println("No packages are installed. Nothing to upgrade.")
			return
		}

		// Determine which packages to consider
		targets := args
		if len(targets) == 0 {
			for name := range state.Packages {
				targets = append(targets, name)
			}
		}

		api := client.NewAPIClient()

		// Build a map of name -> latest version available on any server
		fmt.Println("Checking servers for newer versions...")
		type upgrade struct {
			oldVersion string
			newVersion string
			server     string
		}
		available := make(map[string]upgrade)

		for _, server := range cfg.SearchServers {
			token := cfg.Servers[server].Token
			pkgs, err := api.ListPackages(server, token)
			if err != nil {
				fmt.Printf("  Warning: could not reach %s: %v\n", server, err)
				continue
			}
			for _, p := range pkgs {
				installed, ok := state.Packages[p.Name]
				if !ok {
					continue // not installed, skip
				}
				// Only consider packages in the targets list
				inTargets := false
				for _, t := range targets {
					tName, _ := graph.ParseSpec(t)
					if tName == p.Name {
						inTargets = true
						break
					}
				}
				if !inTargets {
					continue
				}

				// Check if this version is newer than what we have/found so far
				cur, found := available[p.Name]
				compareBase := installed.Version
				if found {
					compareBase = cur.newVersion
				}
				if graph.CompareVersions(p.Version, compareBase) > 0 {
					available[p.Name] = upgrade{
						oldVersion: installed.Version,
						newVersion: p.Version,
						server:     server,
					}
				}
			}
		}

		if len(available) == 0 {
			fmt.Println("All installed packages are up to date.")
			return
		}

		// Print upgrade plan
		fmt.Printf("\n%d package(s) can be upgraded:\n", len(available))
		upgradeSpecs := make([]string, 0, len(available))
		for name, u := range available {
			fmt.Printf("  %-20s %s  →  %s\n", name, u.oldVersion, u.newVersion)
			upgradeSpecs = append(upgradeSpecs, fmt.Sprintf("%s@%s", name, u.newVersion))
		}

		// Resolve & validate the dependency graph for the upgraded versions
		fmt.Println("\nResolving upgraded dependency graph...")
		resolvedGraph, err := graph.ResolveGraph(upgradeSpecs, cfg, api)
		if err != nil {
			fmt.Printf("Resolution error: %v\n", err)
			os.Exit(1)
		}

		sortedOrder, err := graph.TopoSort(resolvedGraph, upgradeSpecs)
		if err != nil {
			fmt.Printf("Dependency graph error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Install order: %s\n\n", strings.Join(sortedOrder, " -> "))

		// Install upgrades in topo order
		for _, pkgName := range sortedOrder {
			node := resolvedGraph[pkgName]
			u, isUpgrade := available[pkgName]
			if isUpgrade {
				fmt.Printf("Upgrading %s (%s -> %s) from %s...\n", pkgName, u.oldVersion, u.newVersion, node.ServerURL)
			} else {
				// dependency that was already up to date — still ensure it's installed
				existing, ok := state.Packages[pkgName]
				if ok && existing.Version == node.Version {
					fmt.Printf("Dependency %s@%s already up to date. Skipping.\n", pkgName, node.Version)
					continue
				}
				fmt.Printf("Installing dependency %s@%s from %s...\n", pkgName, node.Version, node.ServerURL)
			}

			if err := install.InstallSinglePackage(node, api, cfg, state); err != nil {
				fmt.Printf("Failed to upgrade %s: %v\n", pkgName, err)
				os.Exit(1)
			}
		}

		fmt.Println("\nAll upgrades installed successfully.")
		install.EnsureShellSourced()
	},
}

func init() {
	alumUpgradeCmd.Flags().BoolVarP(&upgradeAllFlag, "all", "a", false, "Upgrade all installed packages (default when no args given)")
	rootCmd.AddCommand(alumUpdateCmd)
	rootCmd.AddCommand(alumUpgradeCmd)
}
