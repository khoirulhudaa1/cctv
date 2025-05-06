package middleware

import (
	"net/http"

	// ini sesuai dengan struktur module kamu
	// "your_project/libraries"

	"github.com/gin-gonic/gin"
)

// Middleware untuk memastikan user sudah login
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := c.Cookie("user")
		if err != nil || user == "" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}
