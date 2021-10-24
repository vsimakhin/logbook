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

	checkParam(apiKey, "api_key")
	checkParam(spreadsheetId, "spreadsheet_id")
	checkParam(startRow, "start_row")

	logbook.RendersMap(apiKey, spreadsheetId, startRow, noRoutes, filterDate)
}

func init() {
	rootCmd.AddCommand(renderMapCmd)

	renderMapCmd.Flags().StringVarP(&filterDate, "filter-date", "d", "", "Set filter for the `DATE` logbook field for map rendering")
	renderMapCmd.Flags().BoolVar(&noRoutes, "no-routes", false, "Skip rendering routes on the map")
}
