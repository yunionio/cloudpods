package tokens

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func FernetTokenVerifier(tokenStr string) (mcclient.TokenCredential, error) {
	token := SAuthToken{}
	err := token.ParseFernetToken(tokenStr)
	if err != nil {
		return nil, httperrors.NewInvalidCredentialError("invalid token %s", err)
	}
	userCred, err := token.GetSimpleUserCred(tokenStr)
	if err != nil {
		return nil, err
	}
	log.Debugf("FernetTokenVerify %s %#v %#v", tokenStr, token, userCred)
	return userCred, nil
}
