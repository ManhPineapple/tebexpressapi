package httputil

import (
	"tebexpressapi/pkg/httputil/auth"

	"github.com/gin-gonic/gin"
)

type Route struct {
	Name        string
	Method      string
	BasePath    string
	Pattern     string
	Handler     gin.HandlerFunc
	Middlewares []gin.HandlerFunc
	Timeout     int64
	AuthInfo    *auth.AuthInfo
}

// Routes -- Defines the type Routes which is just an array (slice) of Route structs.
type Routes []Route
