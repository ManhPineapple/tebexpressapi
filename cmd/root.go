package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

var configFile string
var rootCmd = &cobra.Command{
	Use:   "market",
	Short: "CLI of market service",
	Long:  `Market service`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
