package cmd

import (
	"strconv"
	"strings"

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

	verifyConfig()

	verifyParameter(logbookOwner, "owner")
	verifyParameter(pageBrakes, "page_brakes")
	verifyParameter(reverseEntries, "reverse")

	reverse, _ := strconv.ParseBool(reverseEntries)

	logbookConfig := logbook.LogbookConfig{
		SourceType:    sourceType,
		FileName:      fileName,
		APIKey:        apiKey,
		SpreadsheetID: spreadsheetId,
		StartRow:      startRow,
		LogbookOwner:  logbookOwner,
		PageBrakes:    strings.Split(pageBrakes, ","),
		Reverse:       reverse,
	}

	logbook.Export(logbookConfig)
}

func init() {
	rootCmd.AddCommand(exportCmd)

}
