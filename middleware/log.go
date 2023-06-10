package middleware

import (
	"fde_ctrl/logger"

	"github.com/gin-gonic/gin"
)

func LogHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			logger.Error("middle_ware_errors", c.Errors.ByType(gin.ErrorTypePrivate).String(), c.Errors[0])
			return
		}
		logger.Info("middle_ware", sourceWithCtx(c))
	}
}
