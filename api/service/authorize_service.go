package service

import (
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/dao"
	"glt-calendar-service/api/model"
	"glt-calendar-service/settings/env"
	"glt-calendar-service/settings/log"
	"glt-calendar-service/utils"
	"go.uber.org/zap"
	"net/http"
	"time"
)

var (
	cfg            = env.GetConfig()
	respHandler    = utils.NewResponseHandler()
	logger         = log.GetLogger()
	sessionManager = NewSessionManager(dao.NewSessionDao(), logger)
	tokenManager   = NewTokenManager()
)

func GoogleLogin(context *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			respHandler.FailContextMessage(context, gin.H{"error": "Internal server error"}, "Recovered from panic", nil)
		}
	}()

	tokenResponse, err := tokenManager.GetTokenResponse(context)
	if err != nil {
		respHandler.FailContextMessage(context, gin.H{"error": "Failed to get token"}, "", err)
		return
	}

	// 使用訪問令牌獲取用戶信息
	userInfo, err := GetGoogleUserInfo(tokenResponse.AccessToken)
	if err != nil {
		respHandler.FailContextMessage(context, gin.H{"error": "Failed to get user information"}, "", err)
		return
	}

	// 在保存數據中添加令牌創建時間
	sessionData := model.SessionData{
		TokenResponse: tokenResponse,
		UserInfo:      userInfo,
	}

	var sessionId string

	session, _ := sessionManager.GetContextOrSession(context)
	if session != nil && session.Data != nil && session.Data.TokenResponse != nil {
		err = sessionManager.UpdateSession(session)
		if err != nil {
			respHandler.FailContextMessage(context, gin.H{"error": "Failed to update session"}, "", err)
			return
		}
		sessionId = session.SessionID
	} else {
		sessionId, err = sessionManager.SaveSession(userInfo.ID, &sessionData, 24*time.Hour)
		if err != nil {
			respHandler.FailContextMessage(context, gin.H{"error": "Failed to save session"}, "", err)
			return
		}
	}

	sessionManager.SetCookie(context, &model.Cookie{
		Name:     "session_id",
		Value:    sessionId,
		MaxAge:   24 * 60 * 60,
		Path:     "/",
		Domain:   "",
		Secure:   false,
		HttpOnly: true,
	})

	respHandler.SuccessContextMessage(context, userInfo)
}

// GoogleSignOut SignOut handles user logout by removing the session
func GoogleSignOut(context *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			respHandler.FailContextMessage(context, gin.H{"error": "Internal server error"}, "Recovered from panic in SignOut", nil)
		}
	}()

	// Get session ID from a cookie
	sessionID, err := context.Cookie("session_id")
	if err != nil {
		// No session cookie found, considered already logged out
		respHandler.SuccessContextMessage(context, gin.H{"message": "Already signed out"})
		return
	}

	// Delete the session
	err = sessionManager.DeleteSession(sessionID)
	if err != nil {
		logger.Error("Failed to delete session", zap.String("sessionID", sessionID), zap.Error(err))
		respHandler.FailContextMessage(context, gin.H{"error": "Failed to sign out"}, "", err)
		return
	}

	// delete cookie
	sessionManager.SetCookie(context, &model.Cookie{
		Name:     "session_id",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   "",
		Secure:   false,
		HttpOnly: true,
	})

	respHandler.SuccessContextMessage(context, gin.H{"message": "Successfully signed out"})
}

func ValidateSession(context *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			respHandler.FailContextMessage(context, gin.H{"error": "Internal server error"}, "Recovered from panic in validateSession", nil)
		}
	}()

	// 1. Check if a cookie exists
	sessionID, err := context.Cookie("session_id")
	if err != nil {
		respHandler.FailContextCodeMessage(context, http.StatusUnauthorized, gin.H{"error": "Please login first"}, "No session cookie found", err)
		return
	}

	// 2. Get session from DynamoDB
	session, err := sessionManager.GetContextOrSession(context)
	if err != nil {
		respHandler.FailContextCodeMessage(context, http.StatusUnauthorized, gin.H{"error": "Invalid session"}, "Failed to get session", err)
		return
	}

	// 3. Check if the session is expired
	if session.IsSessionExpired() {
		logger.Info("Session expired",
			zap.String("sessionId", sessionID),
			zap.Time("expiryDate", session.ExpiryDate),
		)

		// Delete an expired session
		err := sessionManager.DeleteSession(sessionID)
		if err != nil {
			logger.Error("Failed to delete expired session", zap.Error(err))
			respHandler.FailContextMessage(context, gin.H{"error": "Internal server error"}, "Failed to delete expired session", err)
			return
		}

		// delete cookie
		sessionManager.SetCookie(context, &model.Cookie{
			Name:     "session_id",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			Domain:   "",
			Secure:   false,
			HttpOnly: true,
		})

		respHandler.FailContextCodeMessage(context, http.StatusUnauthorized, gin.H{"error": "Session expired, please login again"}, "", nil)
		return
	}

	// 4. update session expire
	if err := sessionManager.UpdateSession(session); err != nil {
		logger.Error("Failed to update session", zap.Error(err))
		respHandler.FailContextMessage(context, gin.H{"error": "Internal server error"}, "Failed to update session", err)
	}

	// 5. Store session information in context for later use
	context.Set("session", session)
}
