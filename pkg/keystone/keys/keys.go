package keys

import (
	"yunion.io/x/onecloud/pkg/util/fernetool"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

var (
	TokenKeysManager     = fernetool.SFernetKeyManager{}
	CredentialKeyManager = fernetool.SFernetKeyManager{}
)

func Init(tokenKeyRepo, credKeyRepo string) error {
	err := TokenKeysManager.LoadKeys(tokenKeyRepo)
	if err != nil {
		return err
	}
	if fileutils2.IsDir(credKeyRepo) {
		err = CredentialKeyManager.LoadKeys(credKeyRepo)
	} else {
		err = CredentialKeyManager.InitEmpty()
	}
	if err != nil {
		return err
	}
	return nil
}
