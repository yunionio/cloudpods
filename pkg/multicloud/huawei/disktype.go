package huawei

import "strings"

type SDiskType struct {
	ExtraSpecs ExtraSpecs `json:"extra_specs"`
	Name       string     `json:"name"`
	QosSpecsID string     `json:"qos_specs_id"`
	ID         string     `json:"id"`
	IsPublic   bool       `json:"is_public"`
}

type ExtraSpecs struct {
	VolumeBackendName                        string `json:"volume_backend_name"`
	AvailabilityZone                         string `json:"availability-zone"`
	RESKEYAvailabilityZones                  string `json:"RESKEY:availability_zones"`
	OSVendorExtendedSoldOutAvailabilityZones string `json:"os-vendor-extended:sold_out_availability_zones"`
}

func (self *SDiskType) IsAvaliableInZone(zoneId string) bool {
	if len(self.QosSpecsID) > 0 && strings.Contains(zoneId, self.ExtraSpecs.RESKEYAvailabilityZones) && !strings.Contains(zoneId, self.ExtraSpecs.OSVendorExtendedSoldOutAvailabilityZones) {
		return true
	}

	return false
}
