package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/container"
	"github.com/spf13/cobra"
)

var (
	containerNameFlag        string
	containerReplicasFlag    int
	containerSourceFlag      string
	containerInteractiveFlag bool
	containerRestartFlag     string
	containerShellPathFlag   string
	containerPackagesFlag    []string
)

var launchContainerCmd = &cobra.Command{
	Use:   "launch-container [file.ac | image_name]",
	Short: "Launch a container setup (.ac file) or image using nerdctl and colima",
	Long: `Launch container environments defined by text setup documents (.ac files) or directly from container images.
Supports scaling replicas, setting restart policies, interactive shells, and Aluminium package integration.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := container.EnsureContainerTools(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		target := args[0]
		var spec *container.ContainerSpec

		if strings.HasSuffix(target, ".ac") || isFileExist(target) {
			parsedSpec, err := container.ParseACFile(target)
			if err != nil {
				fmt.Printf("Error parsing container setup file '%s': %v\n", target, err)
				os.Exit(1)
			}
			spec = parsedSpec

			// Override with CLI flags if explicitly provided
			if cmd.Flags().Changed("name") {
				spec.Name = containerNameFlag
			}
			if cmd.Flags().Changed("replicas") {
				spec.Replicas = containerReplicasFlag
			}
			if cmd.Flags().Changed("source") {
				spec.Source = containerSourceFlag
			}
			if cmd.Flags().Changed("interactive") {
				spec.InteractiveShell = containerInteractiveFlag
			}
			if cmd.Flags().Changed("restart") {
				spec.RestartPolicy = containerRestartFlag
			}
			if cmd.Flags().Changed("shell-path") {
				spec.InteractiveShellPath = containerShellPathFlag
			}
			if len(containerPackagesFlag) > 0 {
				spec.AluminiumPackages = append(spec.AluminiumPackages, containerPackagesFlag...)
			}

			if spec.Name == "" {
				base := filepath.Base(target)
				spec.Name = strings.TrimSuffix(base, filepath.Ext(base))
			}

			tag, err := container.BuildContainerImage(spec, filepath.Dir(target))
			if err != nil {
				fmt.Printf("Error building container image: %v\n", err)
				os.Exit(1)
			}

			if err := container.LaunchContainer(spec, tag); err != nil {
				fmt.Printf("Error launching container: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Argument is image name or container tag (e.g. ubuntu:latest or test)
			spec = container.NewContainerSpec()
			spec.Image = target
			spec.Source = containerSourceFlag

			if containerNameFlag != "" {
				spec.Name = containerNameFlag
			} else {
				parts := strings.Split(target, ":")
				spec.Name = parts[0]
			}

			spec.Replicas = containerReplicasFlag
			spec.InteractiveShell = containerInteractiveFlag
			spec.RestartPolicy = containerRestartFlag
			spec.InteractiveShellPath = containerShellPathFlag
			spec.AluminiumPackages = containerPackagesFlag

			if spec.Source == "aluminium" {
				server, err := getServerURL(cmd)
				if err != nil {
					fmt.Printf("Error getting server URL: %v\n", err)
					os.Exit(1)
				}
				token, _ := getAuthToken(cmd, server)
				api := client.NewAPIClient()

				fmt.Printf("Pulling container '%s' from Aluminium server...\n", target)
				if err := container.PullContainer(target, "aluminium", server, token, api); err != nil {
					fmt.Printf("Error pulling container from Aluminium server: %v\n", err)
					os.Exit(1)
				}
			}

			if err := container.LaunchContainer(spec, target); err != nil {
				fmt.Printf("Error launching container: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

var publishContainerCmd = &cobra.Command{
	Use:   "publish-container [file.ac | container_name]",
	Short: "Publish container image to Aluminium server registry or Docker Hub",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := container.EnsureContainerTools(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		target := args[0]
		var spec *container.ContainerSpec
		workDir := ""

		if strings.HasSuffix(target, ".ac") || isFileExist(target) {
			parsedSpec, err := container.ParseACFile(target)
			if err != nil {
				fmt.Printf("Error parsing container setup file '%s': %v\n", target, err)
				os.Exit(1)
			}
			spec = parsedSpec
			workDir = filepath.Dir(target)

			if containerNameFlag != "" {
				spec.Name = containerNameFlag
			}
			if cmd.Flags().Changed("source") {
				spec.Source = containerSourceFlag
			}
		} else {
			spec = container.NewContainerSpec()
			spec.Name = target
			if containerNameFlag != "" {
				spec.Name = containerNameFlag
			}
			spec.Source = containerSourceFlag
		}

		server, err := getServerURL(cmd)
		if err != nil {
			fmt.Printf("Error getting server URL: %v\n", err)
			os.Exit(1)
		}
		token, err := getAuthToken(cmd, server)
		if err != nil || token == "" {
			if spec.Source != "dockerhub" {
				fmt.Printf("Error: not authenticated or token not provided for %s\n", server)
				os.Exit(1)
			}
		}

		api := client.NewAPIClient()
		if err := container.PublishContainer(spec, workDir, server, token, api); err != nil {
			fmt.Printf("Error publishing container: %v\n", err)
			os.Exit(1)
		}
	},
}

func isFileExist(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func init() {
	// launch-container flags
	launchContainerCmd.Flags().StringVar(&containerNameFlag, "name", "", "Container base name")
	launchContainerCmd.Flags().IntVar(&containerReplicasFlag, "replicas", 1, "Number of container replicas to launch")
	launchContainerCmd.Flags().StringVar(&containerSourceFlag, "source", "dockerhub", "Container image source registry ('dockerhub' or 'aluminium')")
	launchContainerCmd.Flags().BoolVarP(&containerInteractiveFlag, "interactive", "i", false, "Run container interactively")
	launchContainerCmd.Flags().BoolVarP(&containerInteractiveFlag, "tty", "t", false, "Allocate a pseudo-TTY")
	launchContainerCmd.Flags().StringVar(&containerRestartFlag, "restart", "no", "Restart policy ('no', 'always', 'unless-stopped', 'on-failure')")
	launchContainerCmd.Flags().StringVar(&containerShellPathFlag, "shell-path", "/bin/sh", "Path to interactive shell inside container")
	launchContainerCmd.Flags().StringSliceVar(&containerPackagesFlag, "packages", []string{}, "Aluminium packages to install into container")

	launchContainerCmd.Flags().StringVar(&serverFlag, "server", "", "Aluminium server URL (overrides default)")
	launchContainerCmd.Flags().StringVar(&tokenFlag, "auth-token", "", "Token to use for request (overrides saved token)")

	// publish-container flags
	publishContainerCmd.Flags().StringVar(&containerNameFlag, "name", "", "Container base name")
	publishContainerCmd.Flags().StringVar(&containerSourceFlag, "source", "aluminium", "Destination registry ('aluminium' or 'dockerhub')")
	publishContainerCmd.Flags().StringVar(&serverFlag, "server", "", "Aluminium server URL (overrides default)")
	publishContainerCmd.Flags().StringVar(&tokenFlag, "auth-token", "", "Token to use for request (overrides saved token)")

	rootCmd.AddCommand(launchContainerCmd)
	rootCmd.AddCommand(publishContainerCmd)
}
