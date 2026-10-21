package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) getStats(c *gin.Context) {
	st, err := s.Hub.Store().UserStats(c.Request.Context(), playerFrom(c).ID)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	c.JSON(http.StatusOK, st)
}

func (s *Server) listMatches(c *gin.Context) {
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	offset := 0
	if v := c.Query("cursor"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	page, err := s.Hub.Store().ListHistory(c.Request.Context(), playerFrom(c).ID, limit, offset)
	if err != nil {
		writeErr(c, err, "")
		return
	}
	c.JSON(http.StatusOK, page)
}

func (s *Server) getMatch(c *gin.Context) {
	d, err := s.Hub.Store().MatchDetail(c.Request.Context(), playerFrom(c).ID, c.Param("id"))
	if err != nil {
		writeErr(c, err, "")
		return
	}
	c.JSON(http.StatusOK, d)
}
