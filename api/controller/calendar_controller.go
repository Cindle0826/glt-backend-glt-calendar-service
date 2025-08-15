package controller

import (
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/service"
	"glt-calendar-service/middleware"
)

func Calendar(group *gin.RouterGroup) {
	calendarGroup := group.Group("/calendar", middleware.ValidateSessionHandler())
	{
		calendarGroup.GET("/events", service.GetCalendarEvents)
	}
}
