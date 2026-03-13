package middleware

import (
	"time"

	"github.com/QuantumNous/new-api/metrics"
	"github.com/gin-gonic/gin"
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		tag, _ := c.Get(RouteTagKey)
		tagStr, _ := tag.(string)
		if tagStr == "" {
			tagStr = "web"
		}
		path := c.FullPath()
		if path == "" && c.Request != nil && c.Request.URL != nil {
			path = c.Request.URL.Path
		}
		metrics.ObserveHTTPRequest(tagStr, c.Request.Method, path, c.Writer.Status(), time.Since(start))
	}
}
