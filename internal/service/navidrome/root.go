package navidrome

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ProxyRoot(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/app")
}
