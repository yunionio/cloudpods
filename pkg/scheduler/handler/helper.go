package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
)

func SendJSON(c *gin.Context, code int, obj interface{}) {
	c.Render(http.StatusOK, JSON{Data: obj})
}

type JSON struct {
	Data interface{}
}

func (r JSON) Render(w http.ResponseWriter) error {
	appsrv.SendJSON(w, jsonutils.Marshal(r.Data))
	return nil
}

func (r JSON) WriteContentType(w http.ResponseWriter) {
}
