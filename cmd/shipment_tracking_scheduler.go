package cmd

import (
	shipmenttrackingschedulder "tebexpressapi/services/shipment-tracking-schedulder"

	"github.com/spf13/cobra"
)

// candleCmd represents the candle command
var shipmentTrackingSchedulerCmd = &cobra.Command{
	Use:   "shipment-tracking-scheduler",
	Short: "shipment-tracking-scheduler",
	Long:  `This subcommand for ananbay api`,
	Run: func(cmd *cobra.Command, args []string) {
		app := shipmenttrackingschedulder.NewApp(configFile)
		app.Run()
		defer app.Stop()
	},
}

func init() {
	rootCmd.AddCommand(shipmentTrackingSchedulerCmd)
	shipmentTrackingSchedulerCmd.PersistentFlags().StringVar(&configFile, "config-file", "./config.toml", "config file?")
}
