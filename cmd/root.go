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
var startRow int
var logbookOwner string
var pageBrakes string
var reverseEntries string
var sourceType string
var fileName string

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
			viper.SetDefault("type", "")
			viper.SetDefault("file_name", "logbook.xlsx")
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
		startRow = viper.GetInt("start_row")
		logbookOwner = viper.GetString("owner")
		pageBrakes = viper.GetString("page_brakes")
		reverseEntries = viper.GetString("reverse")
		sourceType = viper.GetString("type")
		fileName = viper.GetString("file_name")
	}
}

// verifyParameter checks individual paramater if it has some value
//
// param interface{} - parameter to verify
// name string - paramater name in the configuration file
func verifyParameter(param interface{}, name string) {
	if reflect.ValueOf(param).IsZero() {
		log.Fatalf("The '%s' value in the %s is not set", name, cfgFile)
	}
}

// verifyConfig checks the set of parameters which should be placed together
// in the configuration file
func verifyConfig() {

	verifyParameter(sourceType, "type")

	if sourceType == "xlsx" {
		verifyParameter(fileName, "file_name")

	} else if sourceType == "google" {
		verifyParameter(apiKey, "api_key")
		verifyParameter(spreadsheetId, "spreadsheet_id")

	} else {
		log.Fatalf("unknown type for the source in the %s config file", cfgFile)

	}

	verifyParameter(startRow, "start_row")

}
