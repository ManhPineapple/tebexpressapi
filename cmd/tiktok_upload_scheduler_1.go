package cmd

import (
	tiktokuploadlabelscheduler "tebexpressapi/services/tiktok-upload-label-scheduler"

	"github.com/spf13/cobra"
)

// candleCmd represents the candle command
var tiktokUploadSchedulerCmd = &cobra.Command{
	Use:   "tiktok-upload-scheduler",
	Short: "tiktok-upload-scheduler",
	Long:  `This subcommand for ananbay api`,
	Run: func(cmd *cobra.Command, args []string) {
		app := tiktokuploadlabelscheduler.NewApp(configFile)
		app.Run()
		defer app.Stop()
	},
}

func init() {
	rootCmd.AddCommand(tiktokUploadSchedulerCmd)
	tiktokUploadSchedulerCmd.PersistentFlags().StringVar(&configFile, "config-file", "./config_dev.toml", "config file?")
}
