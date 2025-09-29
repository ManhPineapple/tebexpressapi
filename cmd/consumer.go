package cmd

import (
	"tebexpressapi/services/consumer"

	"github.com/spf13/cobra"
)

var consumerCmd = &cobra.Command{
	Use:   "consumer",
	Short: "Start consumer service",
	Long:  `This subcommand runs the RabbitMQ consumers (OCR, Upload, etc).`,
	Run: func(cmd *cobra.Command, args []string) {
		consumer.StartConsumers(configFile)
	},
}

func init() {
	rootCmd.AddCommand(consumerCmd)
	consumerCmd.PersistentFlags().StringVar(&configFile, "config-file", "./config_dev.toml", "config file?")
}
