package cmd

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/vsimakhin/logbook/logbook"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export logbook records to pdf",
	Run:   exportRun,
}

func exportRun(cmd *cobra.Command, args []string) {

	checkParam(apiKey, "api_key")
	checkParam(spreadsheetId, "spreadsheet_id")
	checkParam(startRow, "start_row")
	checkParam(logbookOwner, "owner")
	checkParam(pageBrakes, "page_brakes")
	checkParam(reverseEntries, "reverse")

	reverse, _ := strconv.ParseBool(reverseEntries)

	logbook.Export(apiKey, spreadsheetId, startRow, logbookOwner, pageBrakes, reverse)
}

func init() {
	rootCmd.AddCommand(exportCmd)

}
