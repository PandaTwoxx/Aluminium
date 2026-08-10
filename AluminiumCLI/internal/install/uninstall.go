package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/PandaTwoxx/Aluminium/internal/graph"
)

// FindDependents returns installed package names that depend on any package in targets.
func FindDependents(targets map[string]bool, state *InstalledState, cfg *config.Config, api *client.APIClient) ([]string, error) {
	var dependents []string

	for pkgName, installed := range state.Packages {
		if targets[pkgName] {
			continue
		}

		token := cfg.Servers[installed.Server].Token
		pkg, err := api.GetPackage(installed.Server, pkgName, installed.Version, token)
		if err != nil || pkg == nil {
			continue
		}

		for _, depSpec := range pkg.Dependencies {
			depName, _ := graph.ParseSpec(depSpec)
			if targets[depName] {
				dependents = append(dependents, pkgName)
				break
			}
		}
	}

	return dependents, nil
}

// UninstallSinglePackage removes a package from disk and updates installed state.
func UninstallSinglePackage(name string, cfg *config.Config, api *client.APIClient, state *InstalledState) error {
	installed, ok := state.Packages[name]
	if !ok {
		return fmt.Errorf("package %s is not installed", name)
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}

	installDir := filepath.Join(configDir, "install", name)
	buildDir := filepath.Join(configDir, "build", name)

	token := cfg.Servers[installed.Server].Token
	pkg, err := api.GetPackage(installed.Server, name, installed.Version, token)
	if err == nil && pkg != nil && pkg.BuildSetup != nil && pkg.BuildSetup.UninstallScript != "" {
		workingDir := installDir
		if _, statErr := os.Stat(workingDir); os.IsNotExist(statErr) {
			workingDir = buildDir
		}
		if _, statErr := os.Stat(workingDir); statErr == nil {
			fmt.Printf("Running uninstall script for %s@%s...\n", name, installed.Version)
			if scriptErr := runScript(pkg.BuildSetup.UninstallScript, workingDir); scriptErr != nil {
				fmt.Printf("Warning: uninstall script failed: %v\n", scriptErr)
			}
		}
	}

	fmt.Printf("Removing %s@%s from %s...\n", name, installed.Version, installDir)
	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("failed to remove install directory: %w", err)
	}
	_ = os.RemoveAll(buildDir)

	delete(state.Packages, name)
	if err := SaveInstalledState(state); err != nil {
		return err
	}
	if err := RegenerateEnvFile(state); err != nil {
		fmt.Printf("Warning: could not update env file: %v\n", err)
	}

	fmt.Printf("Successfully uninstalled %s@%s.\n", name, installed.Version)
	return nil
}
