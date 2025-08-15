package utils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"glt-calendar-service/api/model"
	"time"
)

func GetCurrentTime() time.Time {
	return time.Now()
}

func GetSessionFromContext(context *gin.Context) (*model.Session, error) {
	session, exists := context.Get("session")
	if !exists {
		return nil, fmt.Errorf("failed to get session from context")
	}

	sessionPtr, ok := session.(*model.Session)
	if !ok {
		return nil, fmt.Errorf("failed to convert session to *model.Session")
	}

	return sessionPtr, nil
}
