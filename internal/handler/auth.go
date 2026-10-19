package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/find-work/tools-web-backend/internal/model"
	"github.com/find-work/tools-web-backend/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterUser(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "auth is not configured (MYSQL_DSN required)"})
		return
	}
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	user, err := h.auth.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "已存在") {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}

func (h *Handler) Login(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "auth is not configured (MYSQL_DSN required)"})
		return
	}
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	clientKey := c.ClientIP() + "|" + strings.TrimSpace(req.Username)
	sessionID, user, err := h.auth.Login(c.Request.Context(), req.Username, req.Password, clientKey)
	if err != nil {
		status := http.StatusUnauthorized
		if strings.Contains(err.Error(), "过多") {
			status = http.StatusTooManyRequests
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	h.setSessionCookie(c, sessionID)
	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	sid, _ := c.Cookie(service.SessionCookieName)
	_ = h.auth.Logout(c.Request.Context(), sid)
	h.clearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) Me(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "auth is not configured (MYSQL_DSN required)"})
		return
	}
	user := currentUser(c)
	if user == nil {
		sid, _ := c.Cookie(service.SessionCookieName)
		u, err := h.auth.UserFromSession(c.Request.Context(), sid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		user = u
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *Handler) setSessionCookie(c *gin.Context, sessionID string) {
	maxAge := int(h.auth.SessionTTL().Seconds())
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     service.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	// 同时清 Secure true/false，避免配置变更后旧 Cookie 清不掉
	for _, secure := range []bool{h.cookieSecure, !h.cookieSecure} {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     service.SessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func (h *Handler) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.auth == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "auth is not configured (MYSQL_DSN required)"})
			return
		}
		sid, err := c.Cookie(service.SessionCookieName)
		if err != nil || sid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}
		user, err := h.auth.UserFromSession(c.Request.Context(), sid)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if user == nil {
			h.clearSessionCookie(c)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}
		c.Set(contextUserKey, user)
		c.Next()
	}
}

const contextUserKey = "auth_user"

func currentUser(c *gin.Context) *model.User {
	v, ok := c.Get(contextUserKey)
	if !ok {
		return nil
	}
	user, _ := v.(*model.User)
	return user
}
