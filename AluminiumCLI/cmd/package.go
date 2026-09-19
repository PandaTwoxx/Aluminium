package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/PandaTwoxx/Aluminium/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	pkgNameFlag            string
	pkgVersionFlag         string
	pkgBuildSystemFlag     string
	pkgForgeFlag           bool
	pkgDependenciesFlag    []string
	pkgCustomBuildFlag     string
	pkgCustomInstallFlag   string
	pkgCustomUninstallFlag string
	pkgBuildFlagsFlag      string
	pkgSourceDirFlag       string
	pkgFileFlag            string
	pkgOutputFlag          string
)

var packageCmd = &cobra.Command{
	Use:   "package",
	Short: "Manage packages on the Aluminium server",
}

var packageRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a package on the server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		if prompt.IsInteractive(cmd, cfg) {
			input, err := prompt.PackageRegister(prompt.PackageRegisterInput{
				Name:                  pkgNameFlag,
				Version:               pkgVersionFlag,
				BuildSystem:           pkgBuildSystemFlag,
				Forge:                 pkgForgeFlag,
				SourceDir:             pkgSourceDirFlag,
				BuildFlags:            pkgBuildFlagsFlag,
				Dependencies:          pkgDependenciesFlag,
				CustomBuildScript:     pkgCustomBuildFlag,
				CustomInstallScript:   pkgCustomInstallFlag,
				CustomUninstallScript: pkgCustomUninstallFlag,
			})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			pkgNameFlag = input.Name
			pkgVersionFlag = input.Version
			pkgBuildSystemFlag = input.BuildSystem
			pkgForgeFlag = input.Forge
			pkgSourceDirFlag = input.SourceDir
			pkgBuildFlagsFlag = input.BuildFlags
			pkgDependenciesFlag = input.Dependencies
			pkgCustomBuildFlag = input.CustomBuildScript
			pkgCustomInstallFlag = input.CustomInstallScript
			pkgCustomUninstallFlag = input.CustomUninstallScript
		} else if pkgNameFlag == "" || pkgVersionFlag == "" {
			fmt.Println("Error: --name and --version are required (or enable interactive mode with `aluminium config set-interactive true`)")
			os.Exit(1)
		}

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

		payload := &client.RegisterPackagePayload{
			Name:                  pkgNameFlag,
			Version:               pkgVersionFlag,
			BuildSystem:           pkgBuildSystemFlag,
			Forge:                 pkgForgeFlag,
			Dependencies:          pkgDependenciesFlag,
			CustomBuildScript:     pkgCustomBuildFlag,
			CustomInstallScript:   pkgCustomInstallFlag,
			CustomUninstallScript: pkgCustomUninstallFlag,
			BuildFlags:            pkgBuildFlagsFlag,
			SourceDir:             pkgSourceDirFlag,
		}

		api := client.NewAPIClient()
		err = api.RegisterPackage(server, payload, token)
		if err != nil {
			fmt.Printf("Error registering package: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Package %s@%s registered successfully.\n", pkgNameFlag, pkgVersionFlag)
	},
}

var packageListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all packages registered on the server",
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
		pkgs, err := api.ListPackages(server, token)
		if err != nil {
			fmt.Printf("Error listing packages: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tBUILD SYSTEM\tDEPENDENCIES")
		fmt.Fprintln(w, "----\t-------\t------------\t------------")
		for _, p := range pkgs {
			deps := strings.Join(p.Dependencies, ", ")
			if deps == "" {
				deps = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.Version, p.BuildSystem, deps)
		}
		w.Flush()
	},
}

var packageGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a specific package",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensurePackageOperationInput(cmd, false, false); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
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
		pkg, err := api.GetPackage(server, pkgNameFlag, pkgVersionFlag, token)
		if err != nil {
			fmt.Printf("Error getting package details: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Package:     %s\n", pkg.Name)
		fmt.Printf("Version:     %s\n", pkg.Version)
		fmt.Printf("BuildSys:    %s\n", pkg.BuildSystem)
		fmt.Printf("Forge:       %t\n", pkg.Forge)
		fmt.Printf("Deps:        %s\n", strings.Join(pkg.Dependencies, ", "))
		if pkg.BuildSetup != nil {
			fmt.Println("Build Setup:")
			if pkg.BuildSetup.SourceCodeURL != "" {
				fmt.Printf("  Source URL: %s\n", pkg.BuildSetup.SourceCodeURL)
			}
			if pkg.BuildSetup.BuildScript != "" {
				fmt.Printf("  Build:      %s\n", pkg.BuildSetup.BuildScript)
			}
			if pkg.BuildSetup.InstallScript != "" {
				fmt.Printf("  Install:    %s\n", pkg.BuildSetup.InstallScript)
			}
			if pkg.BuildSetup.UninstallScript != "" {
				fmt.Printf("  Uninstall:  %s\n", pkg.BuildSetup.UninstallScript)
			}
		}
	},
}

var packageEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit an existing package",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensurePackageOperationInput(cmd, false, false); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
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
		current, err := api.GetPackage(server, pkgNameFlag, pkgVersionFlag, token)
		if err != nil {
			fmt.Printf("Error getting package details: %v\n", err)
			os.Exit(1)
		}

		sourceDir := ""
		if current.BuildSetup != nil {
			sourceDir = current.BuildSetup.SourceCodeURL
		}
		if prompt.IsInteractive(cmd, cfg) {
			input, err := prompt.PackageRegister(prompt.PackageRegisterInput{
				Name: current.Name, Version: current.Version, BuildSystem: current.BuildSystem,
				Forge: current.Forge, Dependencies: current.Dependencies,
				SourceDir: sourceDir,
			})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			pkgNameFlag, pkgVersionFlag = input.Name, input.Version
			pkgBuildSystemFlag, pkgForgeFlag = input.BuildSystem, input.Forge
			pkgDependenciesFlag, pkgSourceDirFlag = input.Dependencies, input.SourceDir
			pkgBuildFlagsFlag = input.BuildFlags
			pkgCustomBuildFlag, pkgCustomInstallFlag = input.CustomBuildScript, input.CustomInstallScript
			pkgCustomUninstallFlag = input.CustomUninstallScript
		}

		payload := &client.RegisterPackagePayload{
			Name: pkgNameFlag, Version: pkgVersionFlag, BuildSystem: pkgBuildSystemFlag,
			Forge: pkgForgeFlag, Dependencies: pkgDependenciesFlag, SourceDir: pkgSourceDirFlag,
			BuildFlags: pkgBuildFlagsFlag, CustomBuildScript: pkgCustomBuildFlag,
			CustomInstallScript: pkgCustomInstallFlag, CustomUninstallScript: pkgCustomUninstallFlag,
		}
		if err := api.UpdatePackage(server, payload, token); err != nil {
			fmt.Printf("Error updating package: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Package %s@%s updated successfully.\n", pkgNameFlag, pkgVersionFlag)
	},
}

var packageUploadCmd = &cobra.Command{
	Use:   "upload-binary",
	Short: "Upload a prebuilt binary archive (.tar.gz) for a package",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensurePackageOperationInput(cmd, true, false); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
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
		err = api.UploadPrebuilt(server, pkgNameFlag, pkgVersionFlag, pkgNameFlag, pkgFileFlag, token)
		if err != nil {
			fmt.Printf("Error uploading prebuilt binary: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Prebuilt binary uploaded successfully for %s@%s.\n", pkgNameFlag, pkgVersionFlag)
	},
}

var packageDownloadCmd = &cobra.Command{
	Use:   "download-binary",
	Short: "Download the prebuilt binary archive for a package",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensurePackageOperationInput(cmd, false, true); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
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
		stream, err := api.DownloadPrebuilt(server, pkgNameFlag, pkgVersionFlag, token)
		if err != nil {
			fmt.Printf("Error downloading prebuilt binary: %v\n", err)
			os.Exit(1)
		}
		defer stream.Close()

		out, err := os.Create(pkgOutputFlag)
		if err != nil {
			fmt.Printf("Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer out.Close()

		_, err = io.Copy(out, stream)
		if err != nil {
			fmt.Printf("Error saving binary: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Prebuilt binary downloaded to %s.\n", pkgOutputFlag)
	},
}

var packageDeleteBinaryCmd = &cobra.Command{
	Use:   "delete-binary",
	Short: "Delete a prebuilt binary archive from the server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensurePackageOperationInput(cmd, false, false); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
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
		err = api.DeletePrebuilt(server, pkgNameFlag, pkgVersionFlag, token)
		if err != nil {
			fmt.Printf("Error deleting prebuilt binary: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Prebuilt binary for %s@%s deleted successfully.\n", pkgNameFlag, pkgVersionFlag)
	},
}

var packageDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete package metadata and registration from the server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensurePackageOperationInput(cmd, false, false); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
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
		err = api.DeletePackage(server, pkgNameFlag, pkgVersionFlag, token)
		if err != nil {
			fmt.Printf("Error deleting package: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Package %s@%s deleted successfully.\n", pkgNameFlag, pkgVersionFlag)
	},
}

func ensurePackageOperationInput(cmd *cobra.Command, needFile, needOutput bool) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if prompt.IsInteractive(cmd, cfg) {
		input, err := prompt.PackageOperation(prompt.PackageOperationInput{
			Name: pkgNameFlag, Version: pkgVersionFlag, File: pkgFileFlag, Output: pkgOutputFlag,
		}, needFile, needOutput)
		if err != nil {
			return err
		}
		pkgNameFlag, pkgVersionFlag = input.Name, input.Version
		pkgFileFlag, pkgOutputFlag = input.File, input.Output
	}
	if pkgNameFlag == "" || pkgVersionFlag == "" {
		return fmt.Errorf("--name and --version are required (or enable interactive mode)")
	}
	if needFile && pkgFileFlag == "" {
		return fmt.Errorf("--file is required (or enable interactive mode)")
	}
	if needOutput && pkgOutputFlag == "" {
		return fmt.Errorf("--output is required (or enable interactive mode)")
	}
	return nil
}

func init() {
	packageCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "Aluminium server URL (overrides default)")
	packageCmd.PersistentFlags().StringVar(&tokenFlag, "auth-token", "", "Token to use for request (overrides saved token)")

	// Register subcommands flags
	packageRegisterCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageRegisterCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	packageRegisterCmd.Flags().StringVar(&pkgBuildSystemFlag, "build-system", "none", "Build system ('cmake', 'make', 'meson', 'custom', 'none')")
	packageRegisterCmd.Flags().BoolVar(&pkgForgeFlag, "forge", false, "Allow installation to a user-selected output directory (dev role or higher)")
	packageRegisterCmd.Flags().StringSliceVar(&pkgDependenciesFlag, "dependencies", []string{}, "Comma-separated list of dependencies")
	packageRegisterCmd.Flags().StringVar(&pkgCustomBuildFlag, "custom-build", "", "Shell script content to compile (for custom build system)")
	packageRegisterCmd.Flags().StringVar(&pkgCustomInstallFlag, "custom-install", "", "Shell script content to install (for custom build system)")
	packageRegisterCmd.Flags().StringVar(&pkgCustomUninstallFlag, "custom-uninstall", "", "Shell script content to uninstall (for custom build system)")
	packageRegisterCmd.Flags().StringVar(&pkgBuildFlagsFlag, "build-flags", "", "Build flags/arguments passed to setup (for cmake, make, meson)")
	packageRegisterCmd.Flags().StringVar(&pkgSourceDirFlag, "source-dir", "", "Relative path or cloneable URL pointing to package sources")
	packageCmd.AddCommand(packageRegisterCmd)

	// Get details flags
	packageGetCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageGetCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	packageCmd.AddCommand(packageGetCmd)

	// Edit flags
	packageEditCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageEditCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	packageEditCmd.Flags().StringVar(&pkgBuildSystemFlag, "build-system", "none", "Build system ('cmake', 'make', 'meson', 'custom', 'none')")
	packageEditCmd.Flags().BoolVar(&pkgForgeFlag, "forge", false, "Allow installation to a user-selected output directory (dev role or higher)")
	packageEditCmd.Flags().StringSliceVar(&pkgDependenciesFlag, "dependencies", []string{}, "Comma-separated list of dependencies")
	packageEditCmd.Flags().StringVar(&pkgCustomBuildFlag, "custom-build", "", "Shell script content to compile")
	packageEditCmd.Flags().StringVar(&pkgCustomInstallFlag, "custom-install", "", "Shell script content to install")
	packageEditCmd.Flags().StringVar(&pkgCustomUninstallFlag, "custom-uninstall", "", "Shell script content to uninstall")
	packageEditCmd.Flags().StringVar(&pkgBuildFlagsFlag, "build-flags", "", "Build flags/arguments")
	packageEditCmd.Flags().StringVar(&pkgSourceDirFlag, "source-dir", "", "Relative path or cloneable URL pointing to package sources")
	packageCmd.AddCommand(packageEditCmd)

	// List command
	packageCmd.AddCommand(packageListCmd)

	// Upload flags
	packageUploadCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageUploadCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	packageUploadCmd.Flags().StringVar(&pkgFileFlag, "file", "", "Path to prebuilt tarball (.tar.gz)")
	packageCmd.AddCommand(packageUploadCmd)

	// Download flags
	packageDownloadCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageDownloadCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	packageDownloadCmd.Flags().StringVar(&pkgOutputFlag, "output", "", "Destination file path (.tar.gz)")
	packageCmd.AddCommand(packageDownloadCmd)

	// Delete binary flags
	packageDeleteBinaryCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageDeleteBinaryCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	packageCmd.AddCommand(packageDeleteBinaryCmd)

	// Delete package flags
	packageDeleteCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageDeleteCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	packageCmd.AddCommand(packageDeleteCmd)

	rootCmd.AddCommand(packageCmd)
}
