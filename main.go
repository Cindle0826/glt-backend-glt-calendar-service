package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"

	"glt-calendar-service/api"
	"glt-calendar-service/api/database"
	"glt-calendar-service/settings/env"
	"glt-calendar-service/settings/log"
	"go.uber.org/zap"
)

var ginLambdaV2 *ginadapter.GinLambdaV2

func setupGin() *gin.Engine {
	// import environment
	config := env.GetConfig()

	// import log config
	logger := log.GetLogger()

	// init gin
	gin.SetMode(config.GinConfig.Mode)
	logger.Info("Gin mode", zap.String("mode", gin.Mode()))

	instance := gin.New()
	api.RegisterRoutes(instance)

	// init DynamoDB Client
	if err := database.InitDynamoDB(); err != nil {
		logger.Error("Failed to initialize DynamoDB", zap.Error(err))
		return nil
	}
	logger.Info("DynamoDB initialized successfully")

	return instance
}

func HandlerV2(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if ginLambdaV2 == nil {
		engine := setupGin()
		ginLambdaV2 = ginadapter.NewV2(engine)
	}
	return ginLambdaV2.ProxyWithContext(ctx, req)
}

func runningInLambda() bool {
	// AWS_LAMBDA_FUNCTION_NAME 在 Lambda 執行環境中會存在
	return os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != ""
}

func main() {
	config := env.GetConfig()

	if runningInLambda() {
		// 在 Lambda 環境一律啟動 Lambda handler（避免因 GIN_MODE 設錯而啟用本地 HTTP 伺服器）
		lambda.Start(HandlerV2)
		return
	}

	// 本地開發模式
	engine := setupGin()
	_ = engine.Run(":" + config.ServerConfig.Port)
}
