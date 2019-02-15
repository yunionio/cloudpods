package fileutils2

import (
	"testing"
)

func TestIsBlockDeviceUsed(t *testing.T) {
	type args struct {
		dev string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nbd1",
			args: args{"/dev/nbd1"},
			want: false,
		},
		{
			name: "sda",
			args: args{"/dev/sda"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBlockDeviceUsed(tt.args.dev); got != tt.want {
				t.Errorf("IsBlockDeviceUsed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDevId(t *testing.T) {
	type args struct {
		spath string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDevId(tt.args.spath); got != tt.want {
				t.Errorf("GetDevId() = %v, want %v", got, tt.want)
			}
		})
	}
}
