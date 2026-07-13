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

var packageUploadCmd = &cobra.Command{
	Use:   "upload-binary",
	Short: "Upload a prebuilt binary archive (.tar.gz) for a package",
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

func init() {
	packageCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "Aluminium server URL (overrides default)")
	packageCmd.PersistentFlags().StringVar(&tokenFlag, "auth-token", "", "Token to use for request (overrides saved token)")

	// Register subcommands flags
	packageRegisterCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageRegisterCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	packageRegisterCmd.Flags().StringVar(&pkgBuildSystemFlag, "build-system", "none", "Build system ('cmake', 'make', 'meson', 'custom', 'none')")
	packageRegisterCmd.Flags().StringSliceVar(&pkgDependenciesFlag, "dependencies", []string{}, "Comma-separated list of dependencies")
	packageRegisterCmd.Flags().StringVar(&pkgCustomBuildFlag, "custom-build", "", "Shell script content to compile (for custom build system)")
	packageRegisterCmd.Flags().StringVar(&pkgCustomInstallFlag, "custom-install", "", "Shell script content to install (for custom build system)")
	packageRegisterCmd.Flags().StringVar(&pkgCustomUninstallFlag, "custom-uninstall", "", "Shell script content to uninstall (for custom build system)")
	packageRegisterCmd.Flags().StringVar(&pkgBuildFlagsFlag, "build-flags", "", "Build flags/arguments passed to setup (for cmake, make, meson)")
	packageRegisterCmd.Flags().StringVar(&pkgSourceDirFlag, "source-dir", "", "Relative path or git URL pointing to package sources")
	packageCmd.AddCommand(packageRegisterCmd)

	// Get details flags
	packageGetCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageGetCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	_ = packageGetCmd.MarkFlagRequired("name")
	_ = packageGetCmd.MarkFlagRequired("version")
	packageCmd.AddCommand(packageGetCmd)

	// List command
	packageCmd.AddCommand(packageListCmd)

	// Upload flags
	packageUploadCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageUploadCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	packageUploadCmd.Flags().StringVar(&pkgFileFlag, "file", "", "Path to prebuilt tarball (.tar.gz)")
	_ = packageUploadCmd.MarkFlagRequired("name")
	_ = packageUploadCmd.MarkFlagRequired("version")
	_ = packageUploadCmd.MarkFlagRequired("file")
	packageCmd.AddCommand(packageUploadCmd)

	// Download flags
	packageDownloadCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageDownloadCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	packageDownloadCmd.Flags().StringVar(&pkgOutputFlag, "output", "", "Destination file path (.tar.gz)")
	_ = packageDownloadCmd.MarkFlagRequired("name")
	_ = packageDownloadCmd.MarkFlagRequired("version")
	_ = packageDownloadCmd.MarkFlagRequired("output")
	packageCmd.AddCommand(packageDownloadCmd)

	// Delete binary flags
	packageDeleteBinaryCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageDeleteBinaryCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	_ = packageDeleteBinaryCmd.MarkFlagRequired("name")
	_ = packageDeleteBinaryCmd.MarkFlagRequired("version")
	packageCmd.AddCommand(packageDeleteBinaryCmd)

	// Delete package flags
	packageDeleteCmd.Flags().StringVar(&pkgNameFlag, "name", "", "Package name")
	packageDeleteCmd.Flags().StringVar(&pkgVersionFlag, "version", "", "Package version")
	_ = packageDeleteCmd.MarkFlagRequired("name")
	_ = packageDeleteCmd.MarkFlagRequired("version")
	packageCmd.AddCommand(packageDeleteCmd)

	rootCmd.AddCommand(packageCmd)
}
