package policy

import (
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestSerialize(t *testing.T) {
	type SEmbed struct {
		UserCred mcclient.TokenCredential
	}
	token := &mcclient.SSimpleToken{}
	token.User = "Test"

	emb := SEmbed{}
	emb.UserCred = token

	jsonEmb := jsonutils.Marshal(&emb)

	t.Logf("json: %s", jsonEmb)

	nemb := SEmbed{}

	err := jsonEmb.Unmarshal(&nemb)
	if err != nil {
		t.Errorf("fail to unmarshal: %s", err)
	} else {
		jsonEmb2 := jsonutils.Marshal(&nemb)
		t.Logf("unmarshal: %s", jsonEmb2)
	}
}

func TestSerialize2(t *testing.T) {
	type SEmbed struct {
		UserCred mcclient.TokenCredential
	}
	type SUnEmbed struct {
		UserCred struct {
			TokenCredential mcclient.TokenCredential
		}
	}
	token := &SPolicyTokenCredential{
		&mcclient.SSimpleToken{
			User: "Test",
		},
	}

	emb := SEmbed{}
	emb.UserCred = token

	jsonEmb := jsonutils.Marshal(&emb)

	t.Logf("json: %s", jsonEmb)

	nemb := SUnEmbed{}

	err := jsonEmb.Unmarshal(&nemb)
	if err != nil {
		t.Errorf("fail to unmarshal: %s", err)
	} else {
		jsonEmb2 := jsonutils.Marshal(&nemb)
		t.Logf("unmarshal: %s", jsonEmb2)
	}
}
