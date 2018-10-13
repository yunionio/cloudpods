package seclib2

import (
	"fmt"

	"github.com/tredoe/osutil/user/crypt/sha512_crypt"

	"yunion.io/x/pkg/util/seclib"
)

func GeneratePassword(passwd string) (string, error) {
	salt := seclib.RandomPassword(8)
	sha512Crypt := sha512_crypt.New()
	return sha512Crypt.Generate([]byte(passwd), []byte(fmt.Sprintf("$6$%s", salt)))
}

func VerifyPassword(passwd string, hash string) error {
	sha512Crypt := sha512_crypt.New()
	return sha512Crypt.Verify(hash, []byte(passwd))
}
