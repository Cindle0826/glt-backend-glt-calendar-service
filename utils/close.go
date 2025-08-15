package utils

import (
	"io"
	"net/http"

	"glt-calendar-service/settings/log"
	"go.uber.org/zap"
)

var logger = log.GetLogger()

// CloseResponseBody 安全地關閉 HTTP 響應體
// 可以作為 defer 函數使用，會處理所有可能的錯誤情況並記錄日誌
// 用法: defer utils.CloseResponseBody(resp, "操作描述")
func CloseResponseBody(resp *http.Response, operation string) {
	if resp == nil {
		return
	}

	if resp.Body == nil {
		return
	}

	if err := resp.Body.Close(); err != nil {
		logger.Error("Failed to close response body",
			zap.Error(err),
			zap.String("operation", operation),
			zap.Int("statusCode", resp.StatusCode),
			zap.String("url", resp.Request.URL.String()),
		)
	}
}

// CloseReader 安全地關閉任何實現了 io.Closer 接口的對象
// 更通用的工具函數，適用於 HTTP 響應體之外的其他資源
// 用法: defer utils.CloseReader(file, "關閉文件")
func CloseReader(closer io.Closer, operation string) {
	if closer == nil {
		return
	}

	if err := closer.Close(); err != nil {
		logger.Error("Failed to close resource",
			zap.Error(err),
			zap.String("operation", operation),
		)
	}
}
