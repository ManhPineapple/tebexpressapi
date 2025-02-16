package prices

import (
	"net/http"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/sqlmanager"

	"github.com/gin-gonic/gin"
)

type serviceResource struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type fetchResponse struct {
	Services []serviceResource `json:"services"`
}

type ErrorResponse struct {
	Error    string   `json:"error"`
	Messages []string `json:"messages,omitempty"`
}

func (h *PriceHandler) GetServices() gin.HandlerFunc {
	return func(c *gin.Context) {
		services, err := h.ServiceManager.GetServices(sqlmanager.ServiceQueryOption{
			Status:    constant.StatusActive,
			PartnerID: h.PartnerMaps[c.Request.Host],
		})
		if err != nil {
			h.Logger.Errorf("get services: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: constant.MessageServerInternalError})
			return
		}

		res := []serviceResource{}
		httputil.Transform(services, &res)
		c.JSON(http.StatusOK, fetchResponse{Services: res})
	}
}
