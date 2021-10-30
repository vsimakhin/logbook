package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vsimakhin/logbook/logbook"
)

var filterDate string
var noRoutes bool

// renderMapCmd represents the renderMap command
var renderMapCmd = &cobra.Command{
	Use:   "render-map",
	Short: "Renders map with visited airports",
	Run:   renderMapRun,
}

func renderMapRun(cmd *cobra.Command, args []string) {

	verifyConfig()

	logbookConfig := logbook.LogbookConfig{
		SourceType:     sourceType,
		FileName:       fileName,
		APIKey:         apiKey,
		SpreadsheetID:  spreadsheetId,
		StartRow:       startRow,
		FilterDate:     filterDate,
		FilterNoRoutes: noRoutes,
	}

	logbook.RendersMap(logbookConfig)
}

func init() {
	rootCmd.AddCommand(renderMapCmd)

	renderMapCmd.Flags().StringVarP(&filterDate, "filter-date", "d", "", "Set filter for the `DATE` logbook field for map rendering")
	renderMapCmd.Flags().BoolVar(&noRoutes, "no-routes", false, "Skip rendering routes on the map")
}
