package cmd

import (
	"tebexpressapi/services/api"

	"github.com/spf13/cobra"
)

// candleCmd represents the candle command
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "api",
	Long:  `This subcommand for ananbay api`,
	Run: func(cmd *cobra.Command, args []string) {
		app := api.NewApi(configFile)
		app.Start()
		defer app.Stop()
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.PersistentFlags().StringVar(&configFile, "config-file", "./config_dev.toml", "config file?")
}
