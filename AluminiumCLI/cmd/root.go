package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aluminium",
	Short: "Aluminium is a CLI tool for downloading system packages.",
	Long:  `Aluminium is a powerful command-line interface (CLI) tool that helps you download and manage system packages efficiently.`,
	Run: func(cmd *cobra.Command, args []string) {
		if val, _ := cmd.Flags().GetBool("version"); val {
			fmt.Println("Aluminium version 2.0.0")
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
} //r
