package service

import "github.com/gin-gonic/gin"

func Ping(context *gin.Context) {
	logger.Info("ping success")
	context.JSON(200, gin.H{"message": "pong"})
}
