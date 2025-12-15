package web

import (
	"log"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/service/emby"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/service/navidrome"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/logs"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/web/cache"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/web/webport"

	"github.com/gin-gonic/gin"
)

// Listen 监听指定端口
func Listen() error {
	initRulePatterns()

	errChan := make(chan error, 2)
	go listenEmby(errChan)
	go listenNavidrome(errChan)

	err := <-errChan
	log.Fatal("http 服务异常: ", err)
	return nil
}

// initRouter 初始化路由引擎
func initRouterEmby(r *gin.Engine) {
	r.Use(referrerPolicySetter())
	r.Use(emby.ApiKeyChecker())
	r.Use(emby.DownloadStrategyChecker())
	if config.C.Cache.Enable {
		r.Use(cache.CacheableRouteMarker())
		r.Use(cache.RequestCacher())
	}
	initRoutes(r)
}

func initRouterNavidrome(r *gin.Engine) {
	r.Use(referrerPolicySetter())
	r.Use(navidrome.PingChecker())
	if config.C.Cache.Enable {
		r.Use(cache.CacheableRouteMarker())
		r.Use(cache.RequestCacher())
	}
	initRoutes(r)
}

// listenHTTP 在指定端口上监听 http 服务
//
// 出现错误时, 会写入 errChan 中
func listenEmby(errChan chan error) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(CustomLogger(webport.HTTP))
	r.Use(func(c *gin.Context) {
		c.Set(webport.GinKey, webport.HTTP)
	})
	initRouterEmby(r)
	logs.Info("在端口【%s】上启动 HTTP 服务", webport.HTTP)
	err := r.Run("0.0.0.0:" + webport.HTTP)
	errChan <- err
}

func listenNavidrome(errChan chan error) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(CustomLogger(webport.NAVI))
	r.Use(func(c *gin.Context) {
		c.Set(webport.GinKey, webport.NAVI)
	})
	initRouterNavidrome(r)
	logs.Info("在端口【%s】上启动 HTTP 服务", webport.NAVI)
	err := r.Run("0.0.0.0:" + webport.NAVI)
	errChan <- err
}

// listenHTTPS 在指定端口上监听 https 服务
//
// 出现错误时, 会写入 errChan 中
