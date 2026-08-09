package cmd

import (
	"os"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/container"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var pullSourceFlag string

var containerPullSubCmd = &cobra.Command{
	Use:   "pull [image_or_container_name]",
	Short: "Pull container image from Docker Hub or Aluminium server registry",
	Args:  cobra.MinimumNArgs(1),
	Run:   runContainerPull,
}

var pullCmd = &cobra.Command{
	Use:   "pull [image_or_container_name]",
	Short: "Pull container image (alias for `aluminium container pull`)",
	Args:  cobra.MinimumNArgs(1),
	Run:   runContainerPull,
}

func runContainerPull(cmd *cobra.Command, args []string) {
	if err := container.EnsureContainerTools(); err != nil {
		color.Red("Error: %v\n", err)
		os.Exit(1)
	}

	target := args[0]
	server, err := getServerURL(cmd)
	if err != nil {
		color.Red("Error getting server URL: %v\n", err)
		os.Exit(1)
	}

	token, _ := getAuthToken(cmd, server)
	api := client.NewAPIClient()

	if err := container.PullContainer(target, pullSourceFlag, server, token, api); err != nil {
		color.Red("Error pulling container: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	containerPullSubCmd.Flags().StringVar(&pullSourceFlag, "source", "dockerhub", "Source registry ('dockerhub' or 'aluminium')")
	pullCmd.Flags().StringVar(&pullSourceFlag, "source", "dockerhub", "Source registry ('dockerhub' or 'aluminium')")

	containerCmd.AddCommand(containerPullSubCmd)
	rootCmd.AddCommand(pullCmd)
}
