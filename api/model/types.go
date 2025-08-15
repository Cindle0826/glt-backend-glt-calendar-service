package model

import (
	"glt-calendar-service/settings/log"
	"go.uber.org/zap"
	"time"
)

// GoogleTokenRequest ==================================== Google OAuth2 ====================================

const (
	// GoogleOAuth2TokenUrl Google OAuth2 Token URL
	GoogleOAuth2TokenUrl = "https://oauth2.googleapis.com/token"
	// GoogleOAuth2RefreshTokenUrl Google OAuth2 Refresh Token URL
	GoogleOAuth2RefreshTokenUrl = "https://oauth2.googleapis.com/token"
)

var logger = log.GetLogger()

type GoogleTokenRequest struct {
	Code        string `json:"code"`
	RedirectUri string `json:"redirectUri"`
}

type GoogleTokenResponse struct {
	AccessToken           string    `json:"access_token"`
	ExpiresIn             int       `json:"expires_in"`
	TokenType             string    `json:"token_type"`
	RefreshToken          string    `json:"refresh_token,omitempty"` // 在刷新令牌時可能不返回
	RefreshTokenExpiresIn int       `json:"refresh_token_expires_in"`
	Scope                 string    `json:"scope,omitempty"`    // 可選字段
	IdToken               string    `json:"id_token,omitempty"` // 可選字段
	CreatedAt             time.Time `json:"-"`                  // 創建時間，不從 JSON 序列化
}

// GooglePhoneInfo struct google phone info
type GooglePhoneInfo struct {
	ResourceName string `json:"resourceName"`
	PhoneNumbers []struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	} `json:"phoneNumbers"`
}

// GoogleUserInfo struct google user profile
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// Calendar ==================================== Google Calendar ====================================

type CalendarEvent struct {
	ID          string    `json:"id"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Start       EventTime `json:"start"`
	End         EventTime `json:"end"`
	Location    string    `json:"location"`
	ColorId     string    `json:"colorId,omitempty"`
	Creator     Person    `json:"creator"`
	Organizer   Person    `json:"organizer"`
	Status      string    `json:"status"`
}

type EventTime struct {
	DateTime string `json:"dateTime,omitempty"`
	Date     string `json:"date,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
}

type Person struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
}

type CalendarResponse struct {
	Kind     string          `json:"kind"`
	Etag     string          `json:"etag"`
	Summary  string          `json:"summary"`
	Updated  string          `json:"updated"`
	TimeZone string          `json:"timeZone"`
	Items    []CalendarEvent `json:"items"`
}

// Session ==================================== DynamoDB Sessions ====================================

type SessionData struct {
	TokenResponse *GoogleTokenResponse
	UserInfo      *GoogleUserInfo
}

type Session struct {
	SessionID  string       `json:"session_id" dynamodbav:"session_id"`
	UserID     string       `json:"user_id" dynamodbav:"user_id"`
	Data       *SessionData `json:"data" dynamodbav:"data"`
	CreateDate time.Time    `json:"create_date" dynamodbav:"create_date"`
	UpdateDate time.Time    `json:"update_date" dynamodbav:"update_date"`
	ExpiryDate time.Time    `json:"expiry_date" dynamodbav:"expiry_date"`
	TTL        int64        `json:"ttl" dynamodbav:"ttl"` // TTL Time To Leave
}

func (s *Session) IsSessionExpired() bool {
	// check session is nil?
	if s == nil {
		logger.Warn("Session is nil when checking session expiry")
		return true
	}

	now := time.Now()

	// check ExpiryDate if setting
	if s.ExpiryDate.IsZero() {
		logger.Warn("Session ExpiryDate is not set")
		return true
	}

	// check session is Expire
	return now.After(s.ExpiryDate)
}

func (s *Session) IsTokenExpired() bool {
	// check session and data exists
	if s == nil || s.Data == nil || s.Data.TokenResponse == nil {
		logger.Warn("Session, Data or TokenResponse is nil when checking token expiry")
		return true
	}

	// check accessToken exists
	accessToken := s.Data.TokenResponse.AccessToken
	if accessToken == "" {
		logger.Warn("Access token is empty")
		return true
	}

	now := time.Now()

	// buffer time
	bufferTime := 5 * time.Minute

	var tokenCreationTime time.Time
	if !s.Data.TokenResponse.CreatedAt.IsZero() {
		// use token creates Time
		tokenCreationTime = s.Data.TokenResponse.CreatedAt
	} else if !s.CreateDate.IsZero() {
		// use session to create time
		tokenCreationTime = s.CreateDate
	} else if !s.UpdateDate.IsZero() {
		// use session to update time
		tokenCreationTime = s.UpdateDate
	} else {
		logger.Warn("Cannot determine token creation time")
		return true
	}

	expiresIn := s.Data.TokenResponse.ExpiresIn
	if expiresIn <= 0 {
		logger.Warn("Invalid expiresIn value", zap.Int("expiresIn", expiresIn))
		return true
	}

	// calc token expire time
	tokenExpiryTime := tokenCreationTime.Add(time.Duration(expiresIn) * time.Second)

	// check token is expired or token about to expire
	isExpired := now.Add(bufferTime).After(tokenExpiryTime)

	if isExpired {
		logger.Info("OAuth token will expire soon",
			zap.Time("tokenExpiryTime", tokenExpiryTime),
			zap.Duration("timeUntilExpiry", tokenExpiryTime.Sub(now)))
	}

	return isExpired
}

// Cookie ==================================== Client Cookie ====================================

type Cookie struct {
	Name     string
	Value    string
	MaxAge   int
	Path     string
	Domain   string
	Secure   bool
	HttpOnly bool
}
