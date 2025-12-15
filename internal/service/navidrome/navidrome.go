package navidrome

import (
	"io"
	"net/http"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/https"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/logs"
	"github.com/gin-gonic/gin"
)

func HandleRoot(c *gin.Context) {
	if c == nil {
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, "/app/")
}

func HandleSSEEvents(c *gin.Context) {
	if c == nil {
		return
	}
	// 透传到 Navidrome 源站
	resp, err := https.ProxyRequest(c.Request, config.C.Navidrome.Host)
	if err != nil {
		logs.Error("SSE 代理失败: %v", err)
		c.String(http.StatusBadGateway, "SSE 代理失败")
		return
	}
	defer resp.Body.Close()

	// 设置禁止缓冲与保持长连接
	https.CloneHeader(c.Writer, resp.Header)
	c.Header("X-Accel-Buffering", "no")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Content-Length", "")
	c.Status(resp.StatusCode)

	flusher, ok := c.Writer.(http.Flusher)
	if ok {
		flusher.Flush()
	}
	buf := make([]byte, 1024)
	for {
		n, er := resp.Body.Read(buf)
		if n > 0 {
			_, _ = c.Writer.Write(buf[:n])
			if ok {
				flusher.Flush()
			}
		}
		if er != nil {
			if er != io.EOF {
				logs.Warn("SSE 流结束: %v", er)
			}
			break
		}
	}
}

func ProxyOrigin(c *gin.Context) {
	if c == nil {
		return
	}
	origin := config.C.Navidrome.Host
	c.Request.Header.Set("X-Forwarded-For", c.ClientIP())
	c.Request.Header.Set("X-Real-IP", c.ClientIP())
	proto := c.Request.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	c.Request.Header.Set("X-Forwarded-Proto", proto)
	c.Request.Header.Set("X-Forwarded-Host", c.Request.Host)
	c.Request.Header.Set("X-Forwarded-Port", func() string {
		h := c.Request.Host
		for i := len(h) - 1; i >= 0; i-- {
			if h[i] == ':' {
				return h[i+1:]
			}
		}
		if proto == "https" {
			return "443"
		}
		return "80"
	}())
	if err := https.ProxyPass(c.Request, c.Writer, origin); err != nil {
		logs.Error("代理异常: %v", err)
	}
}
