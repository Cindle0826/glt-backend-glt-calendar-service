package main

import (
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api"
	"glt-calendar-service/api/database"
	"glt-calendar-service/settings/env"
	"glt-calendar-service/settings/log"
	"go.uber.org/zap"
)

func inits() {
	// import environment
	config := env.GetConfig()

	// import log config
	logger := log.GetLogger()

	// init gin
	gin.SetMode(config.GinConfig.Mode)
	instance := gin.New()
	api.RegisterRoutes(instance)

	// init DynamoDB Client
	if err := database.InitDynamoDB(); err != nil {
		logger.Error("Failed to initialize DynamoDB", zap.Error(err))
		return
	}

	// setting port
	err := instance.Run(":" + config.ServerConfig.Port)
	if err != nil {
		return
	}
}

func main() {
	inits()
}
