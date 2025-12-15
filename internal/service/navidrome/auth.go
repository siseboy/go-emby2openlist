package navidrome

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func PingChecker() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(strings.ToLower(c.Request.URL.Path), "/rest/") {
			return
		}

		// Navidrome 的 Web 与 Subsonic API 支持多种认证方式（header/cookie/query）
		// 此处不主动拦截，保持透传，避免阻断正常登录流程
		// 如需严格校验，可在后续增加针对 query/header 的白名单检查
		_ = http.StatusOK
	}
}
