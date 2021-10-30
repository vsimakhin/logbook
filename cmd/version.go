package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version string
var buildTime string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:\t", version)
		fmt.Println("Build Time:\t", buildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

}
