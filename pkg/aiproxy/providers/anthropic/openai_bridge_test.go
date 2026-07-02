package anthropic

import (
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
)

func TestOpenAINativeBridgeBuildDeepseekAnthropicURL(t *testing.T) {
	p := OpenAINativeBridge()
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString("deepseek-chat"))
	user := jsonutils.NewDict()
	user.Set("role", jsonutils.NewString("user"))
	user.Set("content", jsonutils.NewString("hello"))
	body.Set("messages", jsonutils.NewArray(user))

	req, err := p.BuildUpstreamRequest(&providerapi.ChatContext{
		BaseURL:       "https://api.deepseek.com/anthropic",
		APIKey:        "ds-key",
		UpstreamModel: "deepseek-chat",
	}, body, false)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://api.deepseek.com/anthropic/v1/messages" {
		t.Fatalf("url: %s", req.URL)
	}
	if req.Headers["x-api-key"] != "ds-key" {
		t.Fatal("missing x-api-key header")
	}
}
