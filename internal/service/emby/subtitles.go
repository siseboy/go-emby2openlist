package emby

import (
	"time"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/web/cache"
	"github.com/gin-gonic/gin"
)

// ProxySubtitles 字幕代理, 过期时间设置为 30 天
func ProxySubtitles(c *gin.Context) {
	if c == nil {
		return
	}

	c.Header(cache.HeaderKeyExpired, cache.Duration(time.Hour*24*30))
	ProxyOrigin(c)
}
