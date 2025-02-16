package packages

import (
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/providers/ibblue"

	"github.com/gin-gonic/gin"
)

func (h *PackageHandler) ValidateAddress() gin.HandlerFunc {
	return func(c *gin.Context) {
		form := &CheckAddressForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{constant.MessageParseRequestBody},
			})
			return
		}

		if form.City == "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"City is required"},
			})

			return
		}

		if form.ZipCode == "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"ZipCode is required"},
			})

			return
		}

		if form.CountryCode == "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"Countrycode is required"},
			})

			return
		}

		if form.StateCode == "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"Statecode is required"},
			})

			return
		}

		if form.Address1 == "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"Address1 is required"},
			})

			return
		}

		if strings.ToUpper(form.CountryCode) != "US" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"Country code is invalid"},
			})

			return
		}

		myib := ibblue.NewIBBlue(nil)
		if myib == nil {
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})

			return
		}

		var body = ibblue.AddressRequest{
			Company:    form.Company,
			Line1:      form.Address1,
			Line2:      form.Address2,
			Line3:      "",
			City:       form.City,
			State:      form.StateCode,
			PostalCode: form.ZipCode,
			Country:    form.CountryCode,
		}

		res, message, err := myib.ValidateAddress(body)

		if err != nil {
			h.Logger.Errorf("error send request validate address: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})

			return
		}

		if message != "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{message},
			})

			return
		}

		type Response struct {
			Line1         string `json:"line1"`
			Line2         string `json:"line2"`
			Line3         string `json:"line3,omitempty"`
			LastLine      string `json:"last_line"`
			City          string `json:"city"`
			State         string `json:"state_province"`
			Zipcode       string `json:"zip5"`
			ZipcodeAddon  string `json:"zip4"`
			AddressExists string `json:"address_exists"`
		}

		data := Response{
			Line1:         res.Line1,
			Line2:         res.Line2,
			Line3:         res.Line3,
			LastLine:      res.LastLine,
			City:          res.City,
			State:         res.State,
			Zipcode:       res.Zipcode,
			ZipcodeAddon:  res.ZipcodeAddon,
			AddressExists: res.AddressExists,
		}

		c.JSON(http.StatusOK, data)
	}
}
