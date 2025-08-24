package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/controller"
	"glt-calendar-service/middleware"
	"glt-calendar-service/settings/log"
)

var logger = log.GetLogger()

var routeRegistrations = []func(*gin.RouterGroup){
	controller.Authorize,
	controller.Calendar,
	controller.Health,
}

func RegisterRoutes(route *gin.Engine) {
	route.Use(middleware.InitBaseHandlers()...)

	group := route.Group("/api", middleware.APIHandler())
	for _, apiSetup := range routeRegistrations {
		apiSetup(group)
	}

	route.NoRoute(func(context *gin.Context) {
		logger.Error(fmt.Sprintf("No route found for method : %s, url :  %s", context.Request.Method, context.Request.URL.Path))
		context.JSON(404, gin.H{"error": "Not Found"})
	})
}
