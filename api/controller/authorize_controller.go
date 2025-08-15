package controller

import (
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/service"
)

func Authorize(group *gin.RouterGroup) {
	authorizeGroup := group.Group("/authorize")
	{
		authorizeGroup.GET("/validate", service.ValidateSession)
		authorizeGroup.POST("/googleLogin", service.GoogleLogin)
		authorizeGroup.POST("/googleSignOut", service.GoogleSignOut)
	}
}
