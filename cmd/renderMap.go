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

	err := logbook.CheckSystemFiles()
	if err != nil {
		log.Fatalf("System files check: %v", err)
	}

	logbook.RendersMap(apiKey, spreadsheetId, startRow, noRoutes, filterDate)
}

func init() {
	rootCmd.AddCommand(renderMapCmd)

	renderMapCmd.Flags().StringVarP(&filterDate, "filter-date", "d", "", "Set filter for the `DATE` logbook field for map rendering")
	renderMapCmd.Flags().BoolVar(&noRoutes, "no-routes", false, "Skip rendering routes on the map")
}
