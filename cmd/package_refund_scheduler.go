package cmd

import (
	packagerefundscheduler "tebexpressapi/services/schedulers/package-refund-scheduler"

	"github.com/spf13/cobra"
)

// candleCmd represents the candle command
var packageRefundSchedulerCmd = &cobra.Command{
	Use:   "package-refund-scheduler",
	Short: "package-refund-scheduler",
	Long:  `This subcommand for ananbay api`,
	Run: func(cmd *cobra.Command, args []string) {
		app := packagerefundscheduler.NewApp(configFile)
		app.Run()
		defer app.Stop()
	},
}

func init() {
	rootCmd.AddCommand(packageRefundSchedulerCmd)
	packageRefundSchedulerCmd.PersistentFlags().StringVar(&configFile, "config-file", "./config_dev.toml", "config file?")
}
