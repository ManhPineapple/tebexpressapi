package config

import (
	"bytes"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"
)

// ReadConfigByFiles read config file by file path
func ReadConfigByFiles(configType string, files []string) error {
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return err
		}

		value, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		viper.SetConfigType(configType)
		err = viper.MergeConfig(bytes.NewBuffer(value))
		if err != nil {
			return err
		}
	}

	return nil
}
