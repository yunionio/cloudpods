package aws

import (
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInternetGateway struct {
	region *SRegion

	Attachments       []InternetGatewayAttachment `json:"Attachments"`
	InternetGatewayID string                      `json:"InternetGatewayId"`
	OwnerID           string                      `json:"OwnerId"`
}

type InternetGatewayAttachment struct {
	State string `json:"State"`
	VpcID string `json:"VpcId"`
}

func (i *SInternetGateway) GetId() string {
	return i.InternetGatewayID
}

func (i *SInternetGateway) GetName() string {
	return i.InternetGatewayID
}

func (i *SInternetGateway) GetGlobalId() string {
	return i.GetId()
}

func (i *SInternetGateway) GetStatus() string {
	return ""
}

func (i *SInternetGateway) Refresh() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Refresh")
}

func (i *SInternetGateway) IsEmulated() bool {
	return false
}

func (i *SInternetGateway) GetMetadata() *jsonutils.JSONDict {
	return nil
}
