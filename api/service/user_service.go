package service

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/model"
	"go.uber.org/zap"
	"io"
	"net/http"
)

// GetGoogleUserInfo 通過 access token 獲取用戶資訊
func GetGoogleUserInfo(accessToken string) (*model.GoogleUserInfo, error) {
	// 使用 Google 的 userinfo 端點獲取用戶基本資訊
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request error: %v", err)
	}

	// 在請求頭中設置 access token
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// 發送請求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch request error: %v", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			if err == nil {
				logger.Error("Failed to close response body")
				return
			}
		}
	}()

	// 檢查響應狀態
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch user info error，statusCode: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析響應
	var userInfo model.GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("error parsing user information: %v", err)
	}

	return &userInfo, nil
}

// GetGoogleUserPhone 通過 access token 獲取用戶的電話號碼（需要額外的權限）
func GetGoogleUserPhone(accessToken string) ([]string, error) {
	// 使用 Google People API 獲取電話號碼
	phoneInfoURL := "https://people.googleapis.com/v1/people/me?personFields=phoneNumbers"
	req, err := http.NewRequest("GET", phoneInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request error: %v", err)
	}

	// request set access token
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// 發送請求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request error: %v", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			if err == nil {
				logger.Error("Failed to close response body")
				return
			}
		}
	}()

	// 檢查響應狀態
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch phone info error，statusCode: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析響應
	var phoneInfo model.GooglePhoneInfo
	if err := json.NewDecoder(resp.Body).Decode(&phoneInfo); err != nil {
		return nil, fmt.Errorf("error parsing phone info: %v", err)
	}

	// 提取電話號碼
	var phoneNumbers []string
	for _, phone := range phoneInfo.PhoneNumbers {
		phoneNumbers = append(phoneNumbers, phone.Value)
	}

	return phoneNumbers, nil
}

// GetCompleteUserProfile 獲取完整的用戶資料（基本資訊和電話號碼）
func GetCompleteUserProfile(session *model.Session) (map[string]interface{}, error) {

	accessToken := session.Data.TokenResponse.AccessToken

	// 獲取用戶基本資訊
	userInfo, err := GetGoogleUserInfo(accessToken)
	if err != nil {
		return nil, err
	}

	// 初始化結果
	result := map[string]interface{}{
		"id":             userInfo.ID,
		"email":          userInfo.Email,
		"verified_email": userInfo.VerifiedEmail,
		"name":           userInfo.Name,
		"given_name":     userInfo.GivenName,
		"family_name":    userInfo.FamilyName,
		"picture":        userInfo.Picture,
	}

	// 嘗試獲取電話號碼（可能會失敗，如果沒有適當的權限）
	phoneNumbers, err := GetGoogleUserPhone(accessToken)
	if err == nil && len(phoneNumbers) > 0 {
		result["phone_numbers"] = phoneNumbers
	} else {
		// 記錄錯誤但不阻止返回其他信息
		result["phone_numbers_error"] = "無法獲取電話號碼，可能缺少權限"
	}

	return result, nil
}

func FetchCompleteUserProfile(context *gin.Context) {
	session, err := sessionManager.GetContextOrSession(context)
	if err != nil {
		logger.Error("Failed to get session", zap.Error(err))
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	userinfo, err := GetCompleteUserProfile(session)
	if err != nil {
		logger.Error("Failed to get user info", zap.Error(err))
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
	}
	context.JSON(http.StatusOK, userinfo)
}
