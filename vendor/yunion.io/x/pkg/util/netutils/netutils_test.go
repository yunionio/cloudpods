package netutils

import "testing"

func TestIsExitAddress(t *testing.T) {
	type args struct {
		addr IPV4Addr
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExitAddress(tt.args.addr); got != tt.want {
				t.Errorf("IsExitAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}
