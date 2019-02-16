package openstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/version"
	"yunion.io/x/pkg/utils"
)

type SKeypair struct {
	Fingerprint string
	Name        string
	Type        string
	PublicKey   string
}

type SKeyPair struct {
	Keypair SKeypair
}

func (region *SRegion) GetKeypairs() ([]SKeyPair, error) {
	_, resp, err := region.List("compute", "/os-keypairs", "", nil)
	if err != nil {
		return nil, err
	}
	keypairs := []SKeyPair{}
	return keypairs, resp.Unmarshal(&keypairs, "keypairs")
}

func (region *SRegion) CreateKeypair(name, publicKey, Type string) (*SKeyPair, error) {
	if len(Type) > 0 && !utils.IsInStringArray(Type, []string{"ssh", "x509"}) {
		return nil, fmt.Errorf("only support ssh or x509 type")
	}
	params := map[string]map[string]string{
		"keypair": {
			"name":       name,
			"public_key": publicKey,
		},
	}
	_, maxVersion, _ := region.GetVersion("compute")
	if len(Type) > 0 && version.GE(maxVersion, "2.2") {
		params["keypair"]["type"] = Type
	}
	_, resp, err := region.Post("compute", "/os-keypairs", maxVersion, jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	keypair := &SKeyPair{}
	return keypair, resp.Unmarshal(keypair)
}

func (region *SRegion) DeleteKeypair(name string) error {
	_, err := region.Delete("compute", "/os-keypairs/"+name, "")
	return err
}

func (region *SRegion) GetKeypair(name string) (*SKeyPair, error) {
	_, resp, err := region.Get("compute", "/os-keypairs/"+name, "", nil)
	if err != nil {
		return nil, err
	}
	keypair := &SKeyPair{}
	return keypair, resp.Unmarshal(keypair)
}

func (region *SRegion) syncKeypair(namePrefix, publicKey string) (string, error) {
	keypairs, err := region.GetKeypairs()
	if err != nil {
		return "", err
	}

	for _, keypair := range keypairs {
		if keypair.Keypair.PublicKey == publicKey {
			return keypair.Keypair.Name, nil
		}
	}
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("%s-%d", namePrefix, i)
		if _, err := region.GetKeypair(name); err != nil {
			if err == cloudprovider.ErrNotFound {
				keypair, err := region.CreateKeypair(name, publicKey, "ssh")
				if err != nil {
					return "", err
				}
				return keypair.Keypair.Name, nil
			}
		}
	}
	return "", fmt.Errorf("failed to find uniq name for keypair")
}
