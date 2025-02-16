package packages

import (
	"encoding/json"
	"net/http"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/httputil"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func (h *PackageHandler) Track() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		id := cast.ToInt64(c.Param("id"))
		if id < 1 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:   constant.MessageValidateInput,
				Message: "Missing package id",
			})

			return
		}

		pkg, err := h.PackageManager.GetPackageByPackageID(id)
		if err == gorm.ErrRecordNotFound {
			h.Logger.Error("Get order detail: ", err)
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.MessageNotFound,
			})

			return
		}

		if err != nil {
			h.Logger.Error("Get order detail: ", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})

			return
		}

		if pkg.UserID != userId {
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageNotFound,
			})

			return
		}

		env := viper.GetString("tracking_env")

		if env == constant.EnvDevelopment {
			item := `[
			{
				"location": "NEWALLA,OK,US",
				"description": "Delivered",
				"status": "delivered",
				"ship_time": "2022-02-04T17:03:00Z"
			},
			{
				"location": "OK,NEWALLA,USA",
				"description": "Out for Delivery",
				"status": "in-transit",
				"ship_time": "2022-02-04T12:34:00Z"
			},
			{
				"location": "OK,NEWALLA,USA",
				"description": "Received by Local Unit",
				"status": "in-transit",
				"ship_time": "2022-02-04T12:23:00Z"
			},
			{
				"location": "OK,OKLAHOMA CITY,USA",
				"description": "Departed USPS Facility",
				"status": "in-transit",
				"ship_time": "2022-02-04T10:26:00Z"
			},
			{
				"location": "OK,OKLAHOMA CITY,USA",
				"description": "Departed USPS Facility",
				"status": "in-transit",
				"ship_time": "2022-02-04T10:15:00Z"
			},
			{
				"location": "OK,OKLAHOMA CITY,USA",
				"description": "Departed USPS Facility",
				"status": "in-transit",
				"ship_time": "2022-02-04T10:12:00Z"
			},
			{
				"location": "OK,OKLAHOMA CITY,USA",
				"description": "Scanned at:",
				"status": "in-transit",
				"ship_time": "2022-02-04T07:22:00Z"
			},
			{
				"location": "RICHMOND, TX, US",
				"description": "Accepted at International Facility, USPS pickup scheduled",
				"status": "in-transit",
				"ship_time": "2022-02-01T16:08:22Z"
			},
			{
				"location": "Stafford, TX, US",
				"description": "Arrived at Facility",
				"status": "in-transit",
				"ship_time": "2022-02-01T13:27:00Z"
			},
			{
				"location": "Houston, TX, US",
				"description": "Arrived at Facility",
				"status": "in-transit",
				"ship_time": "2022-02-01T11:42:00Z"
			},
			{
				"location": "Louisville, KY, US",
				"description": "Departed from Facility",
				"status": "in-transit",
				"ship_time": "2022-02-01T10:43:00Z"
			},
			{
				"location": "Anchorage, AK, US",
				"description": "Departed from Facility",
				"status": "in-transit",
				"ship_time": "2022-01-31T20:40:00Z"
			},
			{
				"location": "Incheon, KR",
				"description": "Departed from Facility",
				"status": "in-transit",
				"ship_time": "2022-01-31T14:50:00Z"
			},
			{
				"location": "Hanoi, VN",
				"description": "Departed from Facility",
				"status": "in-transit",
				"ship_time": "2022-01-29T16:00:00Z"
			},
			{
				"location": "Hanoi, VN",
				"description": "Arriving at international airport to go abroad",
				"status": "in-transit",
				"ship_time": "2022-01-28T08:11:06Z"
			},
			{
				"location": "Hanoi, VN",
				"description": "Departed from Processing Center",
				"status": "in-transit",
				"ship_time": "2022-01-28T07:52:26Z"
			},
			{
				"location": "Hanoi, VN",
				"description": "Accepted at Processing Center",
				"status": "processing",
				"ship_time": "2022-01-28T02:35:21Z"
			},
			{
				"location": "",
				"description": "Shipping label created, awaiting item",
				"status": "pre-transit",
				"ship_time": "2022-01-27T15:54:18Z"
			}
		]`

			items := TrackResponse{}

			json.Unmarshal([]byte(item), &items)

			c.JSON(http.StatusOK, items)
		}

		if env != constant.EnvDevelopment && env != constant.EnvProduction {
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{Error: constant.MessageServerInternalError})

			return
		}

		logs, err := h.PackageManager.GetDeliverLogsByPkgID(pkg.ID)
		if err != nil {
			h.Logger.Error("Get order logs: ", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{Error: constant.MessageServerInternalError})

			return
		}

		logs = dto.Transfomer(logs)

		items := TrackResponse{}
		for _, v := range logs {
			status := constant.MapTextStatusLog[v.Type]
			if v.Type == constant.PackageDeliverLogTypeReship {
				status = constant.MapTextStatusLog[constant.PackageStatusInTransit]
			}

			items = append(items, dto.TrackDeDeliverDTO{
				Location:    v.Location,
				Description: v.Description,
				Status:      status,
				ShipTime:    v.ShipTime,
			})
		}

		c.JSON(http.StatusOK, items)
	}
}
