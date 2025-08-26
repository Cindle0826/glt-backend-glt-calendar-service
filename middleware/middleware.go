package middleware

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/service"
	"glt-calendar-service/settings/env"
	"glt-calendar-service/settings/log"
	"go.uber.org/zap"
	"strings"
	"time"
)

var (
	cfg    = env.GetConfig()
	logger = log.GetLogger()
)

func InitBaseHandlers() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		CorsHandler(),
	}
}

func APIHandler() gin.HandlerFunc {
	return func(context *gin.Context) {
		logger.Info(fmt.Sprintf("Request API for method : %s, url :  %s", context.Request.Method, context.Request.URL.Path))
		context.Next()
	}
}

// CorsHandler 實際測試後發現，cors 機制在 lambda 環境會失效，需再 API Gateway 手動新增
func CorsHandler() gin.HandlerFunc {
	f := zap.String("domain", strings.Join(cfg.HttpAllows.Origins, ", "))
	logger.Info("Allow Origins", f)
	return cors.New(cors.Config{
		AllowOrigins: cfg.HttpAllows.Origins, // 允許的前端域名
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"Accept",
			"X-Requested-With",
			"Cookie",
		},
		ExposeHeaders:    []string{"Content-Length", "Set-Cookie"},
		AllowCredentials: true,           // 是否允許 Cookie
		MaxAge:           12 * time.Hour, // 預檢請求的緩存時間
	})
}

func ValidateSessionHandler() gin.HandlerFunc {
	return func(context *gin.Context) {
		service.ValidateSession(context)
		// 修正 S1023: 移除冗餘的 return 語句
	}
}
