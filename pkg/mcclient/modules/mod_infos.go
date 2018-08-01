package modules

import (
	"fmt"
	"io"
	"net/http"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
)

type InfoManager struct {
	ResourceManager
}

var (
	Info InfoManager
)

func (this *InfoManager) Update(s *mcclient.ClientSession, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s", this.URLPath())
	resp, err := this.rawRequest(s, "POST", path, header, body)
	_, json, err := s.ParseJSONResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return json.Get(this.Keyword)
}

func init() {
	Info = InfoManager{NewYunionAgentManager("info", "infos",
		[]string{},
		[]string{})}
	register(&Info)
}
