package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/gin-gonic/gin.v1"
)

func InstallHandler(r *gin.Engine) {
	r.Any("/metrics", gin.WrapH(prometheus.Handler()))
}
