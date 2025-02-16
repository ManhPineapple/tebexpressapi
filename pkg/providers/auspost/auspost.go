package auspost

import "github.com/spf13/viper"

type Auspost struct {
	Client               *Client
	Address              Address
	ProductExpressID     string
	ProductParcelPostID  string
	ProductExpressName   string
	ProductExpressLayout string
	ISEnvDevelopment     bool
}

func NewAuspost(opts *Options) *Auspost {
	if opts == nil {
		opts = &Options{
			Username:      viper.GetString("provider.auspost.username"),
			Password:      viper.GetString("provider.auspost.password"),
			AccountNumber: viper.GetString("provider.auspost.account_number"),
			BaseURL:       viper.GetString("provider.auspost.base_url"),
			Address: Address{
				Name:     viper.GetString("provider.auspost.name"),
				Suburb:   viper.GetString("provider.auspost.suburb"),
				Lines:    []string{viper.GetString("provider.auspost.line1")},
				State:    viper.GetString("provider.auspost.state_province"),
				Postcode: viper.GetString("provider.auspost.postcode"),
				Country:  viper.GetString("provider.auspost.country_code"),
				Phone:    viper.GetString("provider.auspost.phone"),
				Email:    viper.GetString("provider.auspost.email"),
			},
			ProductExpressID:     viper.GetString("provider.auspost.product_express_id"),
			ProductParcelPostID:  viper.GetString("provider.auspost.product_parcel_post_id"),
			ProductExpressName:   viper.GetString("provider.auspost.product_express_name"),
			ProductExpressLayout: viper.GetString("provider.auspost.product_express_layout"),
			ISDevelopment:        viper.GetBool("provider.auspost.development"),
		}
	}

	return &Auspost{
		Client:               NewClient(opts.BaseURL, opts.Username, opts.Password, opts.AccountNumber),
		Address:              opts.Address,
		ProductExpressID:     opts.ProductExpressID,
		ProductParcelPostID:  opts.ProductParcelPostID,
		ProductExpressName:   opts.ProductExpressName,
		ProductExpressLayout: opts.ProductExpressLayout,
		ISEnvDevelopment:     opts.ISDevelopment,
	}
}
