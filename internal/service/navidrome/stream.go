package navidrome

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/service/openlist"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/https"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/logs"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/web/cache"
	"github.com/gin-gonic/gin"
)

func isPrivateHost(host string) bool {
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 10 {
			return true
		}
		if v4[0] == 172 && v4[1] >= 16 && v4[1] <= 31 {
			return true
		}
		if v4[0] == 192 && v4[1] == 168 {
			return true
		}
		return false
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	b0 := ip[0]
	return b0&0xfe == 0xfc
}

func HandleStream(c *gin.Context) {
	if c == nil {
		return
	}
	var mapped string
	var fileName string
	if !strings.HasPrefix(strings.ToLower(c.Request.URL.Path), "/rest/stream") {
		ProxyOrigin(c)
		return
	}

	id := c.Query("id")
	if id != "" {
		var rawPath string
		{
			resp, err := https.Request(http.MethodGet, config.C.Navidrome.Host+"/api/song/"+id).
				Header(c.Request.Header.Clone()).
				Do()
			if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				var obj map[string]any
				if json.NewDecoder(resp.Body).Decode(&obj) == nil {
					if p, ok := obj["path"].(string); ok && p != "" {
						rawPath = p
					}
				}
			}
		}
		if rawPath == "" {
			q := c.Request.URL.Query()
			q.Set("f", "json")
			q.Set("id", id)
			u := config.C.Navidrome.Host + "/rest/getSong?" + q.Encode()
			resp, err := https.Request(http.MethodGet, u).Header(c.Request.Header.Clone()).Do()
			if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				var obj map[string]any
				if json.NewDecoder(resp.Body).Decode(&obj) == nil {
					if ss, ok := obj["subsonic-response"].(map[string]any); ok {
						if s, ok := ss["song"].(map[string]any); ok {
							if p, ok := s["path"].(string); ok && p != "" {
								rawPath = p
							}
						}
					}
				}
			}
		}
		if rawPath != "" {
			mapped = rawPath
			if mp, ok := config.C.Path.MapEmby2Openlist(mapped); ok {
				mapped = mp
			}
			fileName = path.Base(strings.ReplaceAll(mapped, "\\", "/"))
			fi := openlist.FetchInfo{
				Path:         mapped,
				UseTranscode: false,
				Header:       c.Request.Header.Clone(),
			}
			res := openlist.FetchResource(fi)
			if res.Code == http.StatusOK && res.Data.Url != "" {
				forwardedScheme := c.GetHeader("X-Forwarded-Proto")
				if forwardedScheme == "" {
					if c.Request.TLS != nil {
						forwardedScheme = "https"
					} else {
						forwardedScheme = "http"
					}
				}
				forwardedHost := c.GetHeader("X-Forwarded-Host")
				if forwardedHost == "" {
					forwardedHost = c.Request.Host
				}
				forwardedPrefix := c.GetHeader("X-Forwarded-Prefix")
				if forwardedPrefix != "" && !strings.HasPrefix(forwardedPrefix, "/") {
					forwardedPrefix = "/" + forwardedPrefix
				}
				target, _ := url.Parse(res.Data.Url)
				resHost := target.Hostname()
				useExternal := isPrivateHost(resHost)
				if useExternal {
					base := forwardedScheme + "://" + forwardedHost
					if forwardedPrefix != "" {
						if strings.HasSuffix(base, "/") {
							base = strings.TrimRight(base, "/")
						}
						base = base + forwardedPrefix
					}
					if !strings.HasSuffix(base, "/") {
						base = base + "/"
					}
					u, _ := url.Parse(base + "rest/download")
					u.RawQuery = c.Request.URL.RawQuery
					c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*10))
					if fileName != "" && mapped != "" {
						logs.Success("请求成功, 文件: %s | 路径: %s", fileName, mapped)
					}
					logs.Success("请求成功, 重定向到: %s", u.String())
					c.Redirect(http.StatusTemporaryRedirect, u.String())
					return
				} else {
					c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*10))
					if fileName != "" && mapped != "" {
						logs.Success("请求成功, 文件: %s | 路径: %s", fileName, mapped)
					}
					logs.Success("请求成功, 重定向到: %s", res.Data.Url)
					c.Redirect(http.StatusTemporaryRedirect, res.Data.Url)
					return
				}
			}
		}
	}

	// 优先重定向到当前外部域名，避免将 Location 指向内网地址导致客户端无法访问
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	prefix := c.GetHeader("X-Forwarded-Prefix")
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	base := scheme + "://" + host
	if prefix != "" {
		if strings.HasSuffix(base, "/") {
			base = strings.TrimRight(base, "/")
		}
		base = base + prefix
	}
	if !strings.HasSuffix(base, "/") {
		base = base + "/"
	}

	u, _ := url.Parse(base + "rest/download")
	u.RawQuery = c.Request.URL.RawQuery
	c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*10))
	if fileName != "" && mapped != "" {
		logs.Success("请求成功, 文件: %s | 路径: %s", fileName, mapped)
		logs.Tip("emby2openlist 转换路径: %s |  %s", fileName, mapped)
	}
	logs.Success("请求成功, 重定向到: %s", u.String())
	c.Redirect(http.StatusTemporaryRedirect, u.String())
}
