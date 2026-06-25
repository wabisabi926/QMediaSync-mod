package controllers

import (
	"Q115-STRM/internal/models"

	"github.com/gin-gonic/gin"
)

const (
	authMethodAPIKey  = "api_key"
	authMethodSession = "session"
	contextUserKey    = "current_user"
	contextSessionKey = "current_session"
	contextAuthMethod = "auth_method"
)

func SetCurrentUser(c *gin.Context, user *models.User, authMethod string) {
	c.Set(contextUserKey, user)
	c.Set("username", user.Username)
	c.Set(contextAuthMethod, authMethod)
}

func SetCurrentSession(c *gin.Context, session *models.UserSession) {
	c.Set(contextSessionKey, session)
}

func CurrentUser(c *gin.Context) (*models.User, bool) {
	value, ok := c.Get(contextUserKey)
	if !ok {
		return nil, false
	}
	user, ok := value.(*models.User)
	return user, ok && user != nil
}

func CurrentSession(c *gin.Context) (*models.UserSession, bool) {
	value, ok := c.Get(contextSessionKey)
	if !ok {
		return nil, false
	}
	session, ok := value.(*models.UserSession)
	return session, ok && session != nil
}

func CurrentAuthMethod(c *gin.Context) string {
	value, ok := c.Get(contextAuthMethod)
	if !ok {
		return ""
	}
	method, _ := value.(string)
	return method
}
