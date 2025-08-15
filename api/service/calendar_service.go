package service

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/model"
	"glt-calendar-service/utils"
	"io"
	"net/http"
	"net/url"
	"time"
)

func GetCalendarEvents(context *gin.Context) {
	// TODO: 驗證月曆邏輯
	defer func() {
		if r := recover(); r != nil {
			respHandler.FailContextMessage(context, gin.H{"error": "Internal server error"}, "Recovered from panic in GetCalendar", nil)
		}
	}()

	// 獲取訪問令牌
	accessToken, err := tokenManager.GetAccessToken(context)
	if err != nil {
		respHandler.FailContextMessage(context, gin.H{"error": "Failed to get access token"}, "", err)
		return
	}

	// 獲取請求參數
	timeMin := context.DefaultQuery("timeMin", time.Now().Format(time.RFC3339))
	timeMax := context.DefaultQuery("timeMax", time.Now().AddDate(0, 1, 0).Format(time.RFC3339))
	maxResults := context.DefaultQuery("maxResults", "100")
	singleEvents := context.DefaultQuery("singleEvents", "true")
	orderBy := context.DefaultQuery("orderBy", "startTime")
	calendarId := context.DefaultQuery("calendarId", "primary")

	// 構建 Google Calendar API URL
	baseURL := "https://www.googleapis.com/calendar/v3/calendars"
	apiURL := fmt.Sprintf("%s/%s/events", baseURL, url.PathEscape(calendarId))

	// 添加查詢參數
	q := url.Values{}
	q.Add("timeMin", timeMin)
	q.Add("timeMax", timeMax)
	q.Add("maxResults", maxResults)
	q.Add("singleEvents", singleEvents)
	q.Add("orderBy", orderBy)

	fullURL := apiURL + "?" + q.Encode()

	// 創建請求
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		respHandler.FailContextMessage(context, gin.H{"error": "Failed to create request"}, "", err)
		return
	}

	// 添加認證頭
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// 發送請求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		respHandler.FailContextMessage(context, gin.H{"error": "Failed to fetch calendar data"}, "", err)
		return
	}
	defer utils.CloseResponseBody(resp, "GetCalendar")

	// 處理響應（沒有額外的令牌處理，因為我們已經提前檢查並刷新了令牌）
	if resp.StatusCode != http.StatusOK {
		// 處理錯誤
		body, _ := io.ReadAll(resp.Body)
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(body, &errorResponse); err != nil {
			respHandler.FailContextMessage(
				context,
				gin.H{"error": "Failed to parse error response"},
				fmt.Sprintf("Failed to parse error response => body: %s, err: %v", string(body), err),
				err,
			)
			return
		}

		respHandler.FailContextMessage(
			context,
			gin.H{"error": "Failed to fetch calendar data", "details": errorResponse},
			fmt.Sprintf("statusCode : %v ", resp.StatusCode),
			fmt.Errorf("response : %v ", errorResponse),
		)
		return
	}

	// 讀取響應
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		respHandler.FailContextMessage(context, gin.H{"error": "Failed to read response"}, "", fmt.Errorf("failed to read response body : %w", err))
		return
	}

	// 解析日曆數據
	var calendarData model.CalendarResponse
	if err := json.Unmarshal(body, &calendarData); err != nil {
		respHandler.FailContextMessage(context, gin.H{"error": "Failed to parse calendar data"}, "", fmt.Errorf("failed to parse calendar data : %w", err))
		return
	}

	// 返回日曆數據
	respHandler.SuccessContextMessage(context, gin.H{
		"events":   calendarData.Items,
		"timeZone": calendarData.TimeZone,
		"summary":  calendarData.Summary,
	})
}
