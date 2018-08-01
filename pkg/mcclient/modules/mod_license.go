package modules

import (
	"fmt"
	"io"
	"net/http"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
)

type LicenseManager struct {
	ResourceManager
}

var (
	License LicenseManager
)

func (this *LicenseManager) Upload(s *mcclient.ClientSession, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s", this.URLPath())
	resp, err := this.rawRequest(s, "POST", path, header, body)
	_, json, err := s.ParseJSONResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return json.Get(this.Keyword)
}

func init() {
	License = LicenseManager{NewYunionAgentManager("license", "licenses",
		[]string{},
		[]string{})}
	register(&License)
}
