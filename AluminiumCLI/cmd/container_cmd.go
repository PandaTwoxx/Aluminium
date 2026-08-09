package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/PandaTwoxx/Aluminium/internal/container"
	"github.com/PandaTwoxx/Aluminium/internal/prompt"
	"github.com/fatih/color"
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
	containerShowAllFlag     bool
	containerForceFlag       bool
)

var containerCmd = &cobra.Command{
	Use:   "container",
	Short: "Manage Docker-style containers using nerdctl and colima",
	Long: `Manage scalable, recreatable container environments using nerdctl and colima.
Create, publish, pull, list, start, stop, restart, and delete containers using setup documents (.ac files) or standard container images.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var containerLaunchSubCmd = &cobra.Command{
	Use:   "launch [file.ac | image_name]",
	Short: "Launch a container setup (.ac file) or image using nerdctl and colima",
	Long: `Launch container environments defined by text setup documents (.ac files) or directly from container images.
Supports scaling replicas, setting restart policies, interactive shells, and Aluminium package integration.`,
	Run: runContainerLaunch,
}

var launchContainerCmd = &cobra.Command{
	Use:   "launch-container [file.ac | image_name]",
	Short: "Launch a container setup (.ac file) or image (alias for `aluminium container launch`)",
	Long:  `Launch container environments defined by text setup documents (.ac files) or directly from container images.`,
	Run:   runContainerLaunch,
}

func runContainerLaunch(cmd *cobra.Command, args []string) {
	cfg, _ := config.LoadConfig()

	if err := container.EnsureContainerTools(); err != nil {
		color.Red("Error: %v\n", err)
		os.Exit(1)
	}

	var target string
	if len(args) > 0 {
		target = args[0]
	}

	if target == "" && prompt.IsInteractive(cmd, cfg) {
		input, err := prompt.ContainerLaunchPrompt(prompt.ContainerLaunchInput{
			Target:        target,
			Source:        containerSourceFlag,
			Name:          containerNameFlag,
			Replicas:      containerReplicasFlag,
			RestartPolicy: containerRestartFlag,
			Interactive:   containerInteractiveFlag,
		})
		if err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}
		target = input.Target
		containerSourceFlag = input.Source
		containerNameFlag = input.Name
		containerReplicasFlag = input.Replicas
		containerRestartFlag = input.RestartPolicy
		containerInteractiveFlag = input.Interactive
	} else if target == "" {
		color.Red("Error: target setup file (.ac) or image name is required.\n")
		os.Exit(1)
	}

	var spec *container.ContainerSpec

	if strings.HasSuffix(target, ".ac") || isFileExist(target) {
		parsedSpec, err := container.ParseACFile(target)
		if err != nil {
			color.Red("Error parsing container setup file '%s': %v\n", target, err)
			os.Exit(1)
		}
		spec = parsedSpec

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
			color.Red("Error building container image: %v\n", err)
			os.Exit(1)
		}

		if err := container.LaunchContainer(spec, tag); err != nil {
			color.Red("Error launching container: %v\n", err)
			os.Exit(1)
		}
	} else {
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
				color.Red("Error getting server URL: %v\n", err)
				os.Exit(1)
			}
			token, _ := getAuthToken(cmd, server)
			api := client.NewAPIClient()

			color.Cyan("Pulling container '%s' from Aluminium server...\n", target)
			if err := container.PullContainer(target, "aluminium", server, token, api); err != nil {
				color.Red("Error pulling container from Aluminium server: %v\n", err)
				os.Exit(1)
			}
		}

		if err := container.LaunchContainer(spec, target); err != nil {
			color.Red("Error launching container: %v\n", err)
			os.Exit(1)
		}
	}
}

var containerPublishSubCmd = &cobra.Command{
	Use:   "publish [file.ac | container_name]",
	Short: "Publish container image to Aluminium server registry or Docker Hub",
	Run:   runContainerPublish,
}

var publishContainerCmd = &cobra.Command{
	Use:   "publish-container [file.ac | container_name]",
	Short: "Publish container image (alias for `aluminium container publish`)",
	Run:   runContainerPublish,
}

func runContainerPublish(cmd *cobra.Command, args []string) {
	cfg, _ := config.LoadConfig()

	if err := container.EnsureContainerTools(); err != nil {
		color.Red("Error: %v\n", err)
		os.Exit(1)
	}

	var target string
	if len(args) > 0 {
		target = args[0]
	}

	if target == "" && prompt.IsInteractive(cmd, cfg) {
		containers, _ := container.ListContainers(true)
		opts := make([]prompt.ContainerOption, 0, len(containers))
		for _, c := range containers {
			opts = append(opts, prompt.ContainerOption{ID: c.ID, Name: c.Names, Status: c.Status})
		}
		selected, err := prompt.ContainerSelectPrompt(opts, "Select Container Image to Publish")
		if err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}
		target = selected
	} else if target == "" {
		color.Red("Error: target setup file (.ac) or container name is required.\n")
		os.Exit(1)
	}

	var spec *container.ContainerSpec
	workDir := ""

	if strings.HasSuffix(target, ".ac") || isFileExist(target) {
		parsedSpec, err := container.ParseACFile(target)
		if err != nil {
			color.Red("Error parsing container setup file '%s': %v\n", target, err)
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
		color.Red("Error getting server URL: %v\n", err)
		os.Exit(1)
	}
	token, err := getAuthToken(cmd, server)
	if err != nil || token == "" {
		if spec.Source != "dockerhub" {
			color.Red("Error: not authenticated or token not provided for %s\n", server)
			os.Exit(1)
		}
	}

	api := client.NewAPIClient()
	if err := container.PublishContainer(spec, workDir, server, token, api); err != nil {
		color.Red("Error publishing container: %v\n", err)
		os.Exit(1)
	}
}

var containerPsCmd = &cobra.Command{
	Use:     "ps",
	Aliases: []string{"ls"},
	Short:   "List containers",
	Run: func(cmd *cobra.Command, args []string) {
		containers, err := container.ListContainers(containerShowAllFlag)
		if err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}
		container.PrintContainersTable(containers)
	},
}

var containerStartCmd = &cobra.Command{
	Use:   "start [container_name_or_id]",
	Short: "Start one or more stopped containers",
	Run: func(cmd *cobra.Command, args []string) {
		target := getOrPromptContainerTarget(cmd, args, true, "Start Container")
		if err := container.StartContainer(target); err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var containerStopCmd = &cobra.Command{
	Use:   "stop [container_name_or_id]",
	Short: "Stop one or more running containers",
	Run: func(cmd *cobra.Command, args []string) {
		target := getOrPromptContainerTarget(cmd, args, false, "Stop Container")
		if err := container.StopContainer(target); err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var containerRestartCmd = &cobra.Command{
	Use:   "restart [container_name_or_id]",
	Short: "Restart one or more containers",
	Run: func(cmd *cobra.Command, args []string) {
		target := getOrPromptContainerTarget(cmd, args, true, "Restart Container")
		if err := container.RestartContainer(target); err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var containerDeleteCmd = &cobra.Command{
	Use:     "delete [container_name_or_id]",
	Aliases: []string{"rm"},
	Short:   "Remove one or more containers",
	Run: func(cmd *cobra.Command, args []string) {
		target := getOrPromptContainerTarget(cmd, args, true, "Delete Container")
		if err := container.DeleteContainer(target, containerForceFlag); err != nil {
			color.Red("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func isFileExist(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func getOrPromptContainerTarget(cmd *cobra.Command, args []string, showAll bool, title string) string {
	cfg, _ := config.LoadConfig()
	if len(args) > 0 {
		return args[0]
	}
	if prompt.IsInteractive(cmd, cfg) {
		containers, _ := container.ListContainers(showAll)
		opts := make([]prompt.ContainerOption, 0, len(containers))
		for _, c := range containers {
			opts = append(opts, prompt.ContainerOption{ID: c.ID, Name: c.Names, Status: c.Status})
		}
		selected, err := prompt.ContainerSelectPrompt(opts, title)
		if err == nil && selected != "" {
			return selected
		}
	}
	color.Red("Error: container name or ID is required.\n")
	os.Exit(1)
	return ""
}

func init() {
	// Flags for container subcommands
	containerLaunchSubCmd.Flags().StringVar(&containerNameFlag, "name", "", "Container base name")
	containerLaunchSubCmd.Flags().IntVar(&containerReplicasFlag, "replicas", 1, "Number of container replicas to launch")
	containerLaunchSubCmd.Flags().StringVar(&containerSourceFlag, "source", "dockerhub", "Container image source registry ('dockerhub' or 'aluminium')")
	containerLaunchSubCmd.Flags().BoolVarP(&containerInteractiveFlag, "interactive", "i", false, "Run container interactively")
	containerLaunchSubCmd.Flags().BoolVarP(&containerInteractiveFlag, "tty", "t", false, "Allocate a pseudo-TTY")
	containerLaunchSubCmd.Flags().StringVar(&containerRestartFlag, "restart", "no", "Restart policy ('no', 'always', 'unless-stopped', 'on-failure')")
	containerLaunchSubCmd.Flags().StringVar(&containerShellPathFlag, "shell-path", "/bin/sh", "Path to interactive shell inside container")
	containerLaunchSubCmd.Flags().StringSliceVar(&containerPackagesFlag, "packages", []string{}, "Aluminium packages to install into container")

	containerPublishSubCmd.Flags().StringVar(&containerNameFlag, "name", "", "Container base name")
	containerPublishSubCmd.Flags().StringVar(&containerSourceFlag, "source", "aluminium", "Destination registry ('aluminium' or 'dockerhub')")

	containerPsCmd.Flags().BoolVarP(&containerShowAllFlag, "all", "a", false, "Show all containers (default shows running only)")
	containerDeleteCmd.Flags().BoolVarP(&containerForceFlag, "force", "f", false, "Force removal of a running container")

	// Parent container subcommands
	containerCmd.AddCommand(containerLaunchSubCmd)
	containerCmd.AddCommand(containerPublishSubCmd)
	containerCmd.AddCommand(containerPsCmd)
	containerCmd.AddCommand(containerStartCmd)
	containerCmd.AddCommand(containerStopCmd)
	containerCmd.AddCommand(containerRestartCmd)
	containerCmd.AddCommand(containerDeleteCmd)

	// Top-level aliases for launch-container and publish-container
	launchContainerCmd.Flags().StringVar(&containerNameFlag, "name", "", "Container base name")
	launchContainerCmd.Flags().IntVar(&containerReplicasFlag, "replicas", 1, "Number of container replicas to launch")
	launchContainerCmd.Flags().StringVar(&containerSourceFlag, "source", "dockerhub", "Container image source registry ('dockerhub' or 'aluminium')")
	launchContainerCmd.Flags().BoolVarP(&containerInteractiveFlag, "interactive", "i", false, "Run container interactively")
	launchContainerCmd.Flags().BoolVarP(&containerInteractiveFlag, "tty", "t", false, "Allocate a pseudo-TTY")
	launchContainerCmd.Flags().StringVar(&containerRestartFlag, "restart", "no", "Restart policy ('no', 'always', 'unless-stopped', 'on-failure')")
	launchContainerCmd.Flags().StringVar(&containerShellPathFlag, "shell-path", "/bin/sh", "Path to interactive shell inside container")
	launchContainerCmd.Flags().StringSliceVar(&containerPackagesFlag, "packages", []string{}, "Aluminium packages to install into container")

	publishContainerCmd.Flags().StringVar(&containerNameFlag, "name", "", "Container base name")
	publishContainerCmd.Flags().StringVar(&containerSourceFlag, "source", "aluminium", "Destination registry ('aluminium' or 'dockerhub')")

	containerCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "Aluminium server URL (overrides default)")
	containerCmd.PersistentFlags().StringVar(&tokenFlag, "auth-token", "", "Token to use for request (overrides saved token)")

	rootCmd.AddCommand(containerCmd)
	rootCmd.AddCommand(launchContainerCmd)
	rootCmd.AddCommand(publishContainerCmd)
}
