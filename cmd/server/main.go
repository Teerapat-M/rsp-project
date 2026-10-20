package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

// Phase 1: prove the module builds and listens. Rooms, WS, and auth come later.
func main() {
	addr := getenv("HTTP_ADDR", ":8080")

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "phase": 1})
	})

	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
