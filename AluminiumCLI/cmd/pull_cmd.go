package cmd

import (
	"fmt"
	"os"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/container"
	"github.com/spf13/cobra"
)

var pullSourceFlag string

var pullCmd = &cobra.Command{
	Use:   "pull [image_or_container_name]",
	Short: "Pull container image from Docker Hub or Aluminium server registry",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := container.EnsureContainerTools(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		target := args[0]
		server, err := getServerURL(cmd)
		if err != nil {
			fmt.Printf("Error getting server URL: %v\n", err)
			os.Exit(1)
		}

		token, _ := getAuthToken(cmd, server)
		api := client.NewAPIClient()

		if err := container.PullContainer(target, pullSourceFlag, server, token, api); err != nil {
			fmt.Printf("Error pulling container: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	pullCmd.Flags().StringVar(&pullSourceFlag, "source", "dockerhub", "Source registry ('dockerhub' or 'aluminium')")
	pullCmd.Flags().StringVar(&serverFlag, "server", "", "Aluminium server URL (overrides default)")
	pullCmd.Flags().StringVar(&tokenFlag, "auth-token", "", "Token to use for request (overrides saved token)")

	rootCmd.AddCommand(pullCmd)
}
