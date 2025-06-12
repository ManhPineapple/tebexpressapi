package cmd

import (
	ocrscheduler "tebexpressapi/services/ocr-tiktok-label-scheduler"

	"github.com/spf13/cobra"
)

var ocrTiktokSchedulerCmd = &cobra.Command{
	Use:   "ocr-tiktok-label-scheduler",
	Short: "OCR TikTok Label Scheduler",
	Long:  `Process TikTok labels using OCR to extract tracking numbers`,
	Run: func(cmd *cobra.Command, args []string) {
		app := ocrscheduler.NewApp(configFile)
		app.Run()
		defer app.Stop()
	},
}

func init() {
	rootCmd.AddCommand(ocrTiktokSchedulerCmd)
	ocrTiktokSchedulerCmd.PersistentFlags().StringVar(&configFile, "config-file", "./config_dev.toml", "config file")
}
