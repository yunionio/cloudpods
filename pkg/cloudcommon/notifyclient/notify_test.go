package notifyclient

import (
	"html/template"
	"strings"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestNotifyTemplate(t *testing.T) {
	cases := []struct {
		template string
		data     interface{}
		want     string
	}{
		{
			`云主机{{ .name }}创建成功`,
			struct {
				Name string
			}{
				Name: "testsrv-1",
			},
			`云主机testsrv-1创建成功`,
		},
		{
			`您的云主机{{ .name }}已经创建成功，服务器IP地址为{{ .ips }}，{{ if .account }}初始帐号为{{ .account }}，{{ end }}{{ if .keypair }}访问密钥为{{ .keypair }}，{{ end }}{{ if len .password }}初始密码为{{ .password }}，{{ end }}请使用{{ if .windows }}远程桌面连接器(RDC){{ else }}SSH{{ end }}或控制面板控制台访问云主机。`,
			struct {
				Name     string
				Ips      string
				Account  string
				Keypair  string
				Password string
				Windows  bool
			}{
				Name:     "testsrv-1",
				Ips:      "10.168.222.23",
				Account:  "root",
				Password: "1234567",
				Windows:  false,
			},
			`您的云主机testsrv-1已经创建成功，服务器IP地址为10.168.222.23，初始帐号为root，初始密码为1234567，请使用SSH或控制面板控制台访问云主机。`,
		},
	}
	for _, c := range cases {
		temp, err := template.New("template").Parse(c.template)
		if err != nil {
			t.Errorf("parse template %s fail %s", c.template, err)
		} else {
			strBuild := strings.Builder{}
			jsonData := jsonutils.Marshal(c.data)
			t.Logf("jsonData: %s", jsonData)
			err = temp.Execute(&strBuild, jsonData.Interface())
			if err != nil {
				t.Error("execute template fail %s", err)
			} else {
				if strBuild.String() != c.want {
					t.Error("fail: got %s want %s", strBuild.String(), c.want)
				}
			}
		}
	}
}
