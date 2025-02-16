package order

import (
	"tebexpressapi/pkg/config"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"testing"

	"gorm.io/gorm"
)

func connect(t *testing.T) *gorm.DB {
	db, err := storage.NewMysqlConnection(storage.DefaultMysqlFromConfig(nil))
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestValidState(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})
	db := connect(t)
	stateManager := sqlmanager.NewStateManager(db)

	input := []struct {
		country string
		state   string
		code    string
		message string
	}{
		{
			country: "US",
			state:   "CA",
			code:    "CA",
			message: "",
		},
		{
			country: "US",
			state:   "VH",
			code:    "",
			message: "The state code is invalid",
		},
		{
			country: "US",
			state:   "AK",
			code:    "AK",
			message: "The state code no support shipping",
		},
		{
			country: "AU",
			state:   "VIC",
			code:    "VIC",
			message: "",
		},
		{
			country: "AU",
			state:   "Victoria",
			code:    "VIC",
			message: "",
		},
		{
			country: "AU",
			state:   "VI",
			code:    "",
			message: "The state code is invalid",
		},
	}

	validator := MakeValidator(stateManager).SetLang("EN")
	for _, v := range input {
		code, msg, err := validator.ValidState(v.country, v.state, "")
		if err != nil {
			t.Error(err)
		}

		if v.message != msg {
			t.Errorf("message: desire %v, result %v", v.message, msg)
		}

		if v.code != code {
			t.Errorf("code: desire %v, result %v", v.code, code)
		}
	}
}

func TestValid(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})
	db := connect(t)
	stateManager := sqlmanager.NewStateManager(db)

	input := []struct {
		form    *PackageResource
		isValid bool
	}{
		{
			form: &PackageResource{
				OrderNumber: "vh69",
				Code:        "",
				Recipient:   "Joe Biden Joe Biden Joe Biden Joe Biden Joe Biden Joe Biden",
				Phone:       "0912345678",
				Address1:    "1425 Harrison St Joe Biden Joe Biden Joe Biden Joe Biden Joe Biden",
				Address2:    "",
				City:        "MIILWAUKEE Joe Biden Joe Biden Joe Biden Joe Biden Joe Biden Joe Biden",
				State:       "VIC",
				Zipcode:     "94612",
				Country:     "AU",
				Detail:      "TNT-SHIRT",
				Weight:      500,
				Width:       106,
				Length:      106,
				Height:      106,
				Service:     "now",
				Base64Label: "",
			},
			isValid: false,
		},
		{
			form: &PackageResource{
				OrderNumber: "vh69",
				Code:        "",
				Recipient:   "Joe Biden",
				Phone:       "0912345678",
				Address1:    "1425 Harrison St",
				Address2:    "",
				City:        "MIILWAUKEE",
				State:       "VIC",
				Zipcode:     "9461",
				Country:     "AU",
				Detail:      "TNT-SHIRT",
				Weight:      500,
				Width:       8,
				Length:      15,
				Height:      2,
				Service:     "now",
				Base64Label: "",
			},
			isValid: true,
		},
		{
			form: &PackageResource{
				OrderNumber: "vh69",
				Code:        "",
				Recipient:   "Joe Biden",
				Phone:       "0912345678",
				Address1:    "1425 Harrison St",
				Address2:    "",
				City:        "MIILWAUKEE",
				State:       "CA",
				Zipcode:     "94612",
				Country:     "US",
				Detail:      "TNT-SHIRT",
				Weight:      500,
				Width:       8,
				Length:      15,
				Height:      2,
				Service:     "now",
				Base64Label: "",
			},
			isValid: true,
		},
		{
			form: &PackageResource{
				OrderNumber: "vh69",
				Code:        "",
				Recipient:   "Joe Biden",
				Phone:       "0912345678",
				Address1:    "1425 Harrison St",
				Address2:    "",
				City:        "",
				State:       "CA",
				Zipcode:     "94612",
				Country:     "US",
				Detail:      "TNT-SHIRT",
				Weight:      500,
				Width:       8,
				Length:      15,
				Height:      2,
				Service:     "now",
				Base64Label: "",
			},
			isValid: false,
		},
		{
			form: &PackageResource{
				OrderNumber: "vh69",
				Code:        "",
				Recipient:   "Joe Biden",
				Phone:       "0912345678",
				Address1:    "1425 Harrison St",
				Address2:    "",
				City:        "MIILWAUKEE",
				State:       "CA",
				Zipcode:     "94612",
				Country:     "US",
				Detail:      "TNT-SHIRT",
				Weight:      5000,
				Width:       89,
				Length:      15,
				Height:      2,
				Service:     "now",
				Base64Label: "",
			},
			isValid: false,
		},
	}

	validator := MakeValidator(stateManager).SetLang("EN")
	for _, v := range input {
		validator.Reset()
		validator.SetLang("EN")
		validator.Validate(v.form)
		// validator.ParseProducts(productManager, v.form, 2527, true)

		if err := validator.Error(); err != nil {
			t.Error(err)
		}

		if validator.IsValid() != v.isValid {
			if messages := validator.Errors(); len(messages) > 0 {
				t.Errorf("validate: desire %v, result %v", v.isValid, validator.IsValid())
			}
		}

		t.Logf("messages: %v", validator.Errors())
	}
}
