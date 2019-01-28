package ipmitool

import (
	"reflect"
	"testing"
)

func TestGetSysInfo(t *testing.T) {
	type args struct {
		exector IPMIExecutor
	}
	tests := []struct {
		name    string
		args    args
		want    *SystemInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSysInfo(tt.args.exector)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSysInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSysInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
