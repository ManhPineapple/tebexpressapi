package cmd

import (
	packagestatusscheduler "tebexpressapi/services/schedulers/package-status-scheduler"

	"github.com/spf13/cobra"
)

// candleCmd represents the candle command
var packageStatusSchedulerCmd = &cobra.Command{
	Use:   "package-status-scheduler",
	Short: "package-status-scheduler",
	Long:  `This subcommand for ananbay api`,
	Run: func(cmd *cobra.Command, args []string) {
		app := packagestatusscheduler.NewApp(configFile)
		app.Run()
		defer app.Stop()
	},
}

func init() {
	rootCmd.AddCommand(packageStatusSchedulerCmd)
	packageStatusSchedulerCmd.PersistentFlags().StringVar(&configFile, "config-file", "./config_dev.toml", "config file?")
}
