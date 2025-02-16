package alert

import "github.com/spf13/viper"

const (
	TeleType = "tele"
)

type Alert interface {
	SendMessage(msg string) error
}

func NewAlert() Alert {
	alertType := viper.GetString("alert.type")
	switch alertType {
	case TeleType:
		return newTeleBot()
	}

	return nil
}
