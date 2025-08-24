package controller

import (
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/service"
)

func Health(group *gin.RouterGroup) {
	authorizeGroup := group.Group("/health")
	{
		authorizeGroup.GET("/ping", service.Ping)
	}
}
