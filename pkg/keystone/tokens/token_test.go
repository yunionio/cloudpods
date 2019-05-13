package tokens

import (
	"testing"
	"time"

	"github.com/golang-plus/uuid"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/util/fernetool"
)

func newUuid() string {
	u, _ := uuid.NewV4()
	return u.Format(uuid.StyleWithoutDash)
}

func TestSAuthToken_Encode(t *testing.T) {
	fm := fernetool.SFernetKeyManager{}
	err := fm.InitKeys("", 2)
	if err != nil {
		t.Fatalf("SFernetKeyManager InitKeys fail %s", err)
	}

	for i := 0; i < 10; i += 1 {
		token := SAuthToken{}
		token.UserId = newUuid()
		token.Method = api.AUTH_METHOD_PASSWORD
		token.ProjectId = newUuid()
		token.ExpiresAt = time.Now()
		token.AuditIds = []string{newUuid()}

		tk, err := token.Encode()
		if err != nil {
			t.Fatalf("SAuthToken encode fail %s", err)
		}

		ft, err := fm.Encrypt(tk)
		if err != nil {
			t.Fatalf("SFernetKeyManager encrypt fail %s", err)
		}

		dtm := fm.Decrypt(ft, time.Hour)
		token2 := SAuthToken{}
		err = token2.Decode(dtm)
		if err != nil {
			t.Fatalf("SAuthToken decode fail %s", err)
		}

		if token.UserId != token2.UserId {
			t.Fatalf("recovery uuid fail %s != %s", token.UserId, token2.UserId)
		}
	}
}
