package libraries

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetCookieUser(c *gin.Context, username string) {
	c.SetCookie("user", username, 3600, "/", "", false, true)
}

func GetCookieUser(c *gin.Context) string {
	user, _ := c.Cookie("user")
	return user
}

func ClearCookieUser(c *gin.Context) {
	c.SetCookie("user", "", -1, "/", "", false, true)
}

func RequireLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := c.Cookie("user"); err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}
