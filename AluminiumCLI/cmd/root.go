package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/PandaTwoxx/Aluminium"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aluminium",
	Short: "Aluminium is a CLI tool for downloading system packages.",
	Long:  `Aluminium is a powerful command-line interface (CLI) tool that helps you download and manage system packages efficiently.`,
	Run: func(cmd *cobra.Command, args []string) {
		if val, _ := cmd.Flags().GetBool("version"); val {
			version := strings.TrimSpace(aluminium.Version)
			color.Cyan("Aluminium version %s\n", version)
			return
		}
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Print version number")
	rootCmd.PersistentFlags().Bool("interactive", false, "Force interactive mode for this command")
	rootCmd.PersistentFlags().Bool("no-interactive", false, "Disable interactive mode for this command")
}
