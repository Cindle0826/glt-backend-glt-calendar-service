package service

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"glt-calendar-service/api/dao"
	"glt-calendar-service/api/model"
	"glt-calendar-service/utils"
	"go.uber.org/zap"
	"time"
)

// SessionManager handles all session-related operations
type SessionManager struct {
	sessionDao dao.SessionDaoInterface
	logger     *zap.Logger
}

// NewSessionManager creates a new SessionManager instance
func NewSessionManager(sessionDao dao.SessionDaoInterface, logger *zap.Logger) *SessionManager {
	return &SessionManager{
		sessionDao: sessionDao,
		logger:     logger,
	}
}

// GetContextOrSession retrieves session from context or cookies
func (sm *SessionManager) GetContextOrSession(context *gin.Context) (*model.Session, error) {
	session, err := utils.GetSessionFromContext(context)
	if session != nil && err == nil {
		return session, nil
	}

	session, err = sm.GetSession(context)
	if session != nil && err == nil {
		return session, nil
	}
	return nil, err
}

// GetSession retrieves session from cookies
func (sm *SessionManager) GetSession(context *gin.Context) (*model.Session, error) {
	// From cookie get session_id
	sessionID, err := context.Cookie("session_id")
	if err != nil {
		return nil, err
	}

	// Get Session Id
	session, err := sm.sessionDao.GetSessionsBySessionID(sessionID)

	// if not found data
	if err != nil {
		return nil, err
	}

	// check is expired
	if session.IsSessionExpired() {
		// delete expired session
		err := sm.DeleteSession(sessionID)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// SaveSession creates and saves a new session
func (sm *SessionManager) SaveSession(userId string, data *model.SessionData, expiry time.Duration) (string, error) {
	sessionID := uuid.New().String()

	// session 效期設置
	expiryTime := time.Now().Add(expiry)

	// 計算 TTL 值（Unix 時間戳）
	ttl := expiryTime.Unix()

	// Create session Struct
	currentTime := utils.GetCurrentTime()
	saveSession := model.Session{
		SessionID:  sessionID,
		UserID:     userId,
		Data:       data,
		CreateDate: currentTime,
		UpdateDate: currentTime,
		ExpiryDate: expiryTime,
		TTL:        ttl,
	}

	err := sm.sessionDao.InsertSession(saveSession)
	if err != nil {
		return "", err
	}

	sm.logger.Info("Session created successfully", zap.String("sessionID", sessionID))

	return sessionID, nil
}

// UpdateSession updates an existing session
func (sm *SessionManager) UpdateSession(session *model.Session) error {
	session.UpdateDate = utils.GetCurrentTime()
	session.ExpiryDate = utils.GetCurrentTime().Add(24 * time.Hour)
	session.TTL = session.ExpiryDate.Unix()

	err := sm.sessionDao.UpdateSession(*session)
	if err != nil {
		return err
	}

	sm.logger.Info("Session updated successfully", zap.String("sessionID", session.SessionID))
	return nil
}

// DeleteSession deletes a session and its cookie
func (sm *SessionManager) DeleteSession(sessionID string) error {
	err := sm.sessionDao.DeleteSession(sessionID)
	if err != nil {
		return err
	}

	sm.logger.Info("Session deleted successfully", zap.String("sessionID", sessionID))

	return nil
}

// SetCookie sets a cookie in the response
func (sm *SessionManager) SetCookie(context *gin.Context, cookie *model.Cookie) {
	secure, httpOnly := cookie.Secure, cookie.HttpOnly

	if gin.Mode() == gin.ReleaseMode {
		secure, httpOnly = true, true
	}

	context.SetCookie(
		cookie.Name,
		cookie.Value,
		cookie.MaxAge,
		cookie.Path,
		cookie.Domain,
		secure,
		httpOnly,
	)
}
