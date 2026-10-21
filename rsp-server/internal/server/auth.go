package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/auth"
	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func (s *Server) registerAuth(r *gin.Engine) {
	limit := s.limitAuth()
	r.POST("/auth/register", limit, s.authRegister)
	r.POST("/auth/login", limit, s.authLogin)
	r.POST("/auth/guest", limit, s.authGuest)
	r.POST("/auth/google", limit, s.authGoogle)
	r.POST("/auth/forgot", limit, s.authForgot)
	r.POST("/auth/reset", limit, s.authReset)
	r.POST("/auth/refresh", limit, s.authRefresh)
	if s.Auth.Dev() {
		r.POST("/dev/guest", limit, s.devGuest)
	}

	me := r.Group("/")
	me.Use(s.requirePlayer())
	me.GET("/auth/me", s.authMe)
	me.POST("/auth/logout", s.authLogout)
	me.POST("/auth/upgrade", s.authUpgrade)
}

func (s *Server) limitAuth() gin.HandlerFunc {
	l := newIPLimiter(s.AuthLimit, time.Minute)
	return func(c *gin.Context) {
		if !l.allow(c.ClientIP(), time.Now()) {
			writeAPIError(c, http.StatusTooManyRequests, "RATE_LIMIT", "too many requests", nil)
			return
		}
		c.Next()
	}
}

func writeSession(c *gin.Context, status int, sess auth.Session) {
	c.JSON(status, gin.H{
		"accessToken":  sess.AccessToken,
		"refreshToken": sess.RefreshToken,
		"user":         auth.PublicUser(sess.User),
	})
}

func (s *Server) authRegister(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeAPIError(c, http.StatusBadRequest, "VALIDATION", "invalid JSON", nil)
		return
	}
	sess, err := s.Auth.Register(c.Request.Context(), body.Username, body.Password, body.Email)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	writeSession(c, http.StatusCreated, sess)
}

func (s *Server) authLogin(c *gin.Context) {
	var body struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		RememberMe bool   `json:"rememberMe"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeAPIError(c, http.StatusBadRequest, "VALIDATION", "invalid JSON", nil)
		return
	}
	sess, err := s.Auth.Login(c.Request.Context(), body.Username, body.Password, body.RememberMe)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	writeSession(c, http.StatusOK, sess)
}

func (s *Server) authGuest(c *gin.Context) {
	var body guestBody
	_ = c.ShouldBindJSON(&body)
	sess, err := s.Auth.Guest(c.Request.Context(), body.DisplayName, body.Avatar, body.Color)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	writeSession(c, http.StatusOK, sess)
}

func (s *Server) authGoogle(c *gin.Context) {
	var body struct {
		IDToken string `json:"idToken"`
	}
	_ = c.ShouldBindJSON(&body)
	sess, err := s.Auth.Google(c.Request.Context(), body.IDToken)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	writeSession(c, http.StatusOK, sess)
}

func (s *Server) authForgot(c *gin.Context) {
	var body struct {
		Email string `json:"email"`
	}
	_ = c.ShouldBindJSON(&body)
	resetURL, err := s.Auth.Forgot(c.Request.Context(), body.Email)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	out := gin.H{"message": "If that email is on an account, we sent a reset link."}
	if s.Auth.Dev() && resetURL != "" {
		out["resetUrl"] = resetURL
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) authReset(c *gin.Context) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeAPIError(c, http.StatusBadRequest, "VALIDATION", "invalid JSON", nil)
		return
	}
	if err := s.Auth.Reset(c.Request.Context(), body.Token, body.Password); err != nil {
		writeErr(c, err, "")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password updated"})
}

func (s *Server) authRefresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = c.ShouldBindJSON(&body)
	sess, err := s.Auth.Refresh(c.Request.Context(), body.RefreshToken)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	writeSession(c, http.StatusOK, sess)
}

func (s *Server) authMe(c *gin.Context) {
	c.JSON(http.StatusOK, auth.PublicUser(playerFrom(c)))
}

func (s *Server) authLogout(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = c.ShouldBindJSON(&body)
	_ = s.Auth.Logout(c.Request.Context(), playerFrom(c).ID, body.RefreshToken)
	c.Status(http.StatusNoContent)
}

func (s *Server) authUpgrade(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeAPIError(c, http.StatusBadRequest, "VALIDATION", "invalid JSON", nil)
		return
	}
	sess, err := s.Auth.Upgrade(c.Request.Context(), playerFrom(c).ID, body.Username, body.Password, body.Email)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	writeSession(c, http.StatusOK, sess)
}

func (s *Server) devGuest(c *gin.Context) {
	if !s.Auth.Dev() {
		writeAPIError(c, http.StatusNotFound, "NOT_FOUND", "not found", nil)
		return
	}
	s.authGuest(c)
}

func (s *Server) playerFromRequest(c *gin.Context) (ident.Player, error) {
	raw := bearerOrQuery(c.Request)
	if raw != "" {
		uid, err := s.Auth.ParseAccess(raw)
		if err == nil {
			return s.Auth.Me(c.Request.Context(), uid)
		}
		if ident.LooksLikeJWT(raw) || !s.Auth.Dev() {
			return ident.Player{}, auth.ErrUnauthorized
		}
	}
	if !s.Auth.Dev() {
		return ident.Player{}, ident.ErrNoPlayer
	}
	p, err := ident.FromRequest(c.Request)
	if err != nil {
		return ident.Player{}, err
	}
	return s.Hub.Store().EnsureGuest(c.Request.Context(), p)
}

func bearerOrQuery(r *http.Request) string {
	if authz := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return strings.TrimSpace(authz[7:])
	}
	for _, p := range websocket.Subprotocols(r) {
		if strings.HasPrefix(p, "bearer.") {
			return strings.TrimPrefix(p, "bearer.")
		}
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}
