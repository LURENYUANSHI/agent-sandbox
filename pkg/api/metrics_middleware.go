package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/metrics"
)

// MetricsMiddleware records API request count and latency.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		elapsed := time.Since(start).Seconds()

		metrics.APIRequests.WithLabelValues(method, path, status).Inc()
		metrics.APILatency.WithLabelValues(method, path).Observe(elapsed)
	}
}
