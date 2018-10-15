package aws

type SKeypair struct {
	KeyPairFingerPrint string
	KeyPairName        string
}

func (self *SRegion) GetKeypairs(finger string, name string, offset int, limit int) ([]SKeypair, int, error) {
	return nil, 0, nil
}

func (self *SRegion) ImportKeypair(name string, pubKey string) (*SKeypair, error) {
	return nil, nil
}

func (self *SRegion) AttachKeypair(instanceId string, name string) error {
	return nil
}

func (self *SRegion) DetachKeyPair(instanceId string, name string) error {
	return nil
}

func (self *SRegion) lookUpAwsKeypair(publicKey string) (string, error) {
	return "", nil
}

func (self *SRegion) importAwsKeypair(publicKey string) (string, error) {
	return "", nil
}

func (self *SRegion) syncKeypair(publicKey string) (string, error) {
	return "", nil
}