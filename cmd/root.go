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
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string
var apiKey string
var spreadsheetId string
var startRow string
var logbookOwner string
var pageBrakes string
var reverseEntries string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "logbook",
	Short: "Logbook CLI tool",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.logbook.json)")

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.CompletionOptions.DisableDescriptions = true
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".logbook" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".logbook")

		// Check if config file exists
		cfgFile = home + string(os.PathSeparator) + ".logbook.json"
		if _, err = os.Stat(cfgFile); os.IsNotExist(err) {
			log.Printf("Config file %s does not exist, creating...\n", cfgFile)
			if _, err := os.Create(cfgFile); err != nil { // perm 0666
				fmt.Fprintln(os.Stderr, err)
			}

			// default values
			viper.SetDefault("api_key", "")
			viper.SetDefault("spreadsheet_id", "")
			viper.SetDefault("start_row", 20)
			viper.SetDefault("owner", "Loogbook Owner")
			viper.SetDefault("page_brakes", "")
			viper.SetDefault("reverse", "true")

			err = viper.WriteConfig()
			if err != nil {
				log.Fatalf("Error writing config file: %v\n", err)
			}
		}

	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		apiKey = viper.GetString("api_key")
		spreadsheetId = viper.GetString("spreadsheet_id")
		startRow = viper.GetString("start_row")
		logbookOwner = viper.GetString("owner")
		pageBrakes = viper.GetString("page_brakes")
		reverseEntries = viper.GetString("reverse")
	}
}

func checkParam(param interface{}, name string) {
	if reflect.ValueOf(param).IsZero() {
		log.Fatalf("The '%s' value in the %s is not set", name, cfgFile)
	}
}
