package httputil

import (
	"net/http"

	"github.com/spf13/cast"
)

func GetRequestPaginateForCustomer(r *http.Request) (offset, limit int) {
	page := cast.ToInt(r.URL.Query().Get("page"))
	limit = cast.ToInt(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 20
	}

	// Maximum records is 200
	if limit > 200 {
		limit = 200
	}

	return page*limit - limit, limit
}

func GetRequestPaginate(r *http.Request) (offset, limit int) {
	page := cast.ToInt(r.URL.Query().Get("page"))
	limit = cast.ToInt(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 50
	}

	// Maximum records is 250
	if limit > 250 {
		limit = 250
	}

	return page*limit - limit, limit
}
