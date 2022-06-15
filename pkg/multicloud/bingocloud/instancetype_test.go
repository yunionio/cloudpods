package bingocloud

import "testing"

// {
//    "instanceType":"dp1.xlarge",
//    "cpu":"64",
//    "hba":"0",
//    "displayName":"64核256G8GPU",
//    "ram":"262144",
//    "ssd":"0",
//    "isBareMetal":"false",
//    "gpu":"8",
//    "sriov":"0",
//    "max":"12",
//    "description":"",
//    "disk":"0",
//    "hdd":"0",
//    "available":"1"
//}
type fields struct {
	Available    string
	CPU          string
	Description  string
	Disk         string
	DisplayName  string
	Gpu          string
	Hba          string
	Hdd          string
	InstanceType string
	IsBareMetal  string
	Max          string
	RAM          string
	Sriov        string
	Ssd          string
}

var testField = fields{
	Available:    "dp1.xlarge",
	CPU:          "64",
	Description:  "",
	Disk:         "0",
	DisplayName:  "64核256G8GPU",
	Gpu:          "8",
	Hba:          "0",
	Hdd:          "0",
	InstanceType: "dp1.xlarge",
	IsBareMetal:  "false",
	Max:          "12",
	RAM:          "262144",
	Sriov:        "0",
	Ssd:          "0",
}

func TestSInstanceType_GetId(t *testing.T) {
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
		{"id", testField, "dp1.xlarge"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self := &SInstanceType{
				Available:    tt.fields.Available,
				CPU:          tt.fields.CPU,
				Description:  tt.fields.Description,
				Disk:         tt.fields.Disk,
				DisplayName:  tt.fields.DisplayName,
				Gpu:          tt.fields.Gpu,
				Hba:          tt.fields.Hba,
				Hdd:          tt.fields.Hdd,
				InstanceType: tt.fields.InstanceType,
				IsBareMetal:  tt.fields.IsBareMetal,
				Max:          tt.fields.Max,
				RAM:          tt.fields.RAM,
				Sriov:        tt.fields.Sriov,
				Ssd:          tt.fields.Ssd,
			}
			if got := self.GetId(); got != tt.want {
				t.Errorf("GetId() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSInstanceType_GetCpuArch(t *testing.T) {
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
		{"cpuArch", testField, "dp1.xlarge"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self := &SInstanceType{
				Available:    tt.fields.Available,
				CPU:          tt.fields.CPU,
				Description:  tt.fields.Description,
				Disk:         tt.fields.Disk,
				DisplayName:  tt.fields.DisplayName,
				Gpu:          tt.fields.Gpu,
				Hba:          tt.fields.Hba,
				Hdd:          tt.fields.Hdd,
				InstanceType: tt.fields.InstanceType,
				IsBareMetal:  tt.fields.IsBareMetal,
				Max:          tt.fields.Max,
				RAM:          tt.fields.RAM,
				Sriov:        tt.fields.Sriov,
				Ssd:          tt.fields.Ssd,
			}
			if got := self.GetId(); got != tt.want {
				t.Errorf("GetId() = %v, want %v", got, tt.want)
			}
		})
	}
}
