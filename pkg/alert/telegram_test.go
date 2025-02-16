package alert

import (
	"tebexpressapi/pkg/config"
	"testing"
)

func TestTele(t *testing.T) {
	err := config.ReadConfigByFiles("toml", []string{"../../config_dev.toml"})
	if err != nil {
		t.Log(err)
	}

	bot := newTeleBot()
	err = bot.SendMessage("hihi")
	if err != nil {
		t.Log(err)
	}
}
