package aws

import "time"

type VirtualMFADevice struct {
	EnableDate       time.Time
	SerialNumber     string
	Base32StringSeed string
	QRCodePNG        string
	User             struct {
		UserName   string
		Arn        string
		UserId     string
		CreateDate time.Time
	}
}

func (cli *SAwsClient) GetVirtualMFADevices() ([]VirtualMFADevice, error) {
	ret := []VirtualMFADevice{}
	for {
		params := map[string]string{}
		part := struct {
			Marker            string             `xml:"Marker"`
			VirtualMFADevices []VirtualMFADevice `xml:"VirtualMFADevices>member"`
		}{}
		err := cli.iamRequest("ListVirtualMFADevices", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.VirtualMFADevices...)
		if len(part.VirtualMFADevices) == 0 || len(part.Marker) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (cli *SAwsClient) DeleteVirtualMFADevice(serialNumber, userName string) error {
	params := map[string]string{
		"SerialNumber": serialNumber,
	}
	if len(userName) > 0 {
		params["UserName"] = userName
	}
	return cli.iamRequest("DeactivateMFADevice", params, nil)
}

func (cli *SAwsClient) CreateVirtualMFADevice(name string) (*VirtualMFADevice, error) {
	params := map[string]string{
		"VirtualMFADeviceName": name,
	}
	ret := struct {
		VirtualMFADevice VirtualMFADevice `xml:"VirtualMFADevice"`
	}{}
	err := cli.iamRequest("CreateVirtualMFADevice", params, &ret)
	if err != nil {
		return nil, err
	}
	return &ret.VirtualMFADevice, nil
}

func (cli *SAwsClient) ResyncMFADevice(serialNumber, userName, code1, code2 string) error {
	params := map[string]string{
		"SerialNumber":        serialNumber,
		"AuthenticationCode1": code1,
		"AuthenticationCode2": code2,
		"UserName":            userName,
	}
	return cli.iamRequest("ResyncMFADevice", params, nil)
}
