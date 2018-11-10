package credentials

type KeyPairCredential struct {
	PrivateKey        string
	PublicKeyId       string
	SessionExpiration int
}
