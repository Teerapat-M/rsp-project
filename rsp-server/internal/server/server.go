package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/auth"
	"github.com/Teerapat-M/rsp-project/internal/hub"
	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
	"github.com/gin-gonic/gin"
)

type Server struct {
	Hub  *hub.Hub
	Auth *auth.Service
	// CORSOrigins is the browser allowlist. Empty means development: reflect any Origin.
	CORSOrigins []string
	// AuthLimit is max unauthenticated auth POSTs per IP per minute. Zero disables the limiter (tests).
	AuthLimit int
	// Ping is the readiness check (Postgres). Nil treats /ready as always ok.
	Ping func(context.Context) error
}

func New(h *hub.Hub, a *auth.Service) *Server {
	return &Server{Hub: h, Auth: a}
}

func (s *Server) Router() *gin.Engine {
	r := gin.New()
	// Rate limits key on ClientIP; do not trust X-Forwarded-For until
	// TRUSTED_PROXIES is set (should-fix).
	_ = r.SetTrustedProxies(nil)
	r.Use(gin.Recovery(), gin.Logger(), s.cors())
	r.GET("/health", s.health)
	r.GET("/ready", s.ready)
	s.registerAuth(r)

	authed := r.Group("/")
	authed.Use(s.requirePlayer())
	authed.POST("/rooms", s.createRoom)
	authed.POST("/rooms/join", s.joinRoom)
	authed.GET("/rooms/:id", s.getRoom)
	authed.POST("/rooms/:id/leave", s.leaveRoom)
	authed.POST("/rooms/:id/start", s.startRoom)
	authed.GET("/stats", requireRegistered, s.getStats)
	authed.GET("/matches", requireRegistered, s.listMatches)
	authed.GET("/matches/:id", s.getMatch)

	r.GET("/ws", s.ws)
	return r
}

func (s *Server) cors() gin.HandlerFunc {
	allow := s.CORSOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if OriginAllowed(origin, allow) {
			if origin == "" {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Player-ID, X-Player-Name, X-Player-Avatar, X-Player-Color")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "phase": 6})
}

func (s *Server) ready(c *gin.Context) {
	if s.Ping != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := s.Ping(ctx); err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "UNAVAILABLE", "database not ready", nil)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type guestBody struct {
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
	Color       string `json:"color"`
}

func (s *Server) requirePlayer() gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := s.playerFromRequest(c)
		if err != nil {
			writeAPIError(c, http.StatusUnauthorized, "UNAUTHORIZED",
				"sign in required: Authorization: Bearer <jwt>", nil)
			return
		}
		c.Set("player", p)
		c.Next()
	}
}

func playerFrom(c *gin.Context) ident.Player {
	v, _ := c.Get("player")
	p, _ := v.(ident.Player)
	return p
}

type createBody struct {
	WinsToFinish    int `json:"winsToFinish"`
	TurnSeconds     int `json:"turnSeconds"`
	WinsToFinishAlt int `json:"wins_to_finish"`
	TurnSecondsAlt  int `json:"turn_seconds"`
}

func (b createBody) wins() int {
	if b.WinsToFinish != 0 {
		return b.WinsToFinish
	}
	return b.WinsToFinishAlt
}

func (b createBody) seconds() int {
	if b.TurnSeconds != 0 {
		return b.TurnSeconds
	}
	return b.TurnSecondsAlt
}

func (s *Server) createRoom(c *gin.Context) {
	var body createBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeAPIError(c, http.StatusBadRequest, "VALIDATION", "invalid JSON", nil)
		return
	}
	room, err := s.Hub.Create(c.Request.Context(), playerFrom(c), body.wins(), body.seconds())
	if err != nil {
		writeErr(c, err, "")
		return
	}
	c.JSON(http.StatusCreated, room)
}

type joinBody struct {
	Passcode string `json:"passcode"`
}

func (s *Server) joinRoom(c *gin.Context) {
	var body joinBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeAPIError(c, http.StatusBadRequest, "VALIDATION", "invalid JSON", nil)
		return
	}
	code := rooms.NormalizePasscode(body.Passcode)
	room, err := s.Hub.Join(c.Request.Context(), playerFrom(c), code)
	if err != nil {
		writeErr(c, err, code)
		return
	}
	c.JSON(http.StatusOK, room)
}

func (s *Server) getRoom(c *gin.Context) {
	room, err := s.Hub.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeErr(c, err, "")
		return
	}
	c.JSON(http.StatusOK, room)
}

func (s *Server) leaveRoom(c *gin.Context) {
	_, err := s.Hub.Leave(c.Request.Context(), c.Param("id"), playerFrom(c).ID)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) startRoom(c *gin.Context) {
	err := s.Hub.StartMatch(c.Param("id"), playerFrom(c).ID)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) ws(c *gin.Context) {
	p, err := s.playerFromRequest(c)
	if err != nil {
		writeAPIError(c, http.StatusUnauthorized, "UNAUTHORIZED",
			"sign in required: token query or Authorization: Bearer <jwt>", nil)
		return
	}
	roomID := c.Query("room")
	if roomID == "" {
		writeAPIError(c, http.StatusBadRequest, "VALIDATION", "room query param required", nil)
		return
	}
	s.Hub.ServeWS(c.Writer, c.Request, roomID, p)
}
