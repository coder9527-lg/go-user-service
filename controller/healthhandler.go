package controller

import (
	"net/http"

	"github.com/coder9527-lg/go-user-service/db"
	"github.com/gin-gonic/gin"
)

// Ready 就绪探针：除进程存活外，还确认数据库可连接。
// 容器编排（K8s readinessProbe）用它判断能否承接流量。
func Ready(c *gin.Context) {
	if err := db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "database unavailable",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ready"})
}
