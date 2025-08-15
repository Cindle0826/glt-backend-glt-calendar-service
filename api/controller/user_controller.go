package controller

import (
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/service"
)

func User(group *gin.RouterGroup) {
	calendarGroup := group.Group("/user")
	{
		calendarGroup.POST("/userProfile", service.FetchCompleteUserProfile)
	}
}
