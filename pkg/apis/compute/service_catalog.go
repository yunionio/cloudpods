package compute

import "yunion.io/x/onecloud/pkg/apis"

type ServiceCatalogCreateInput struct {
	apis.SharableVirutalResourceCreateInput
	// description: service catalog icon url
	// example: https://yunion.io/files/hello.png
	IconUrl string `json:"icon_url"`

	// description: the id or name of guest template
	// example: good
	GuestTemplate string `json:"guest_template"`
}

type ServiceCatalogDeploy struct {

	// description: name of the new vm
	// example: hello
	Name string `json:"name"`

	// description: generate name automatically if name is repeated, and only one of name and this shoudle be given
	// example: hello
	GenerateName string `json:"generate_name"`

	// description: the count of the new vm
	// example: 1
	Count int `json:"count"`
}
