package auth

import (
	"tebexpressapi/pkg/config"
	"testing"
)

func TestAuth(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

}
