/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"log"
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

	err := logbook.CheckSystemFiles()
	if err != nil {
		log.Fatalf("System files check: %v", err)
	}

	logbook.Export(apiKey, spreadsheetId, startRow, logbookOwner, pageBrakes, reverse)
}

func init() {
	rootCmd.AddCommand(exportCmd)

}
