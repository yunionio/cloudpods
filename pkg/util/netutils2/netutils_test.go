package netutils2

import (
	"reflect"
	"testing"
)

func TestSNetInterface_getRoutes(t *testing.T) {
	n := SNetInterface{name: "br0"}
	routes := []string{"Kernel IP routing table",
		"Destination     Gateway         Genmask         Flags Metric Ref    Use Iface",
		"0.0.0.0         10.168.222.1    0.0.0.0         UG    0      0        0 br0",
		"10.168.222.0    0.0.0.0         255.255.255.0   U     0      0        0 br0",
		"169.254.169.254 10.168.222.1    255.255.255.255 UGH   0      0        0 br0",
		""}

	want := [][]string{{"0.0.0.0", "10.168.222.1", "0.0.0.0"}, {"169.254.169.254", "10.168.222.1", "255.255.255.255"}}

	if got := n.getRoutes(true, routes); !reflect.DeepEqual(got, want) {
		t.Errorf("getParams() = %v, want %v", got, want)
	}
}

func TestSNetInterface_getAddresses(t *testing.T) {
	output := []string{"4: br0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UNKNOWN qlen 1000",
		"link/ether 00:22:d5:9e:28:d1 brd ff:ff:ff:ff:ff:ff",
		"inet 10.168.222.236/24 brd 10.168.222.255 scope global br0",
		"valid_lft forever preferred_lft forever",
		"inet6 fe80::222:d5ff:fe9e:28d1/64 scope link",
		"valid_lft forever preferred_lft forever",
		""}
	want := [][]string{{"10.168.222.236", "24"}}

	type fields struct {
		name string
		Addr string
		Mask []byte
		Mac  string
	}
	type args struct {
		output []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   [][]string
	}{
		{
			name: "br0 test",
			args: args{output},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &SNetInterface{
				name: tt.fields.name,
				Addr: tt.fields.Addr,
				Mask: tt.fields.Mask,
				Mac:  tt.fields.Mac,
			}
			if got := n.getAddresses(tt.args.output); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SNetInterface.getAddresses() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNetlen2Mask(t *testing.T) {
	type args struct {
		netmasklen int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "mask 0",
			args: args{0},
			want: "0.0.0.0",
		},
		{
			name: "mask 24",
			args: args{24},
			want: "255.255.255.0",
		},
		{
			name: "mask 32",
			args: args{32},
			want: "255.255.255.255",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Netlen2Mask(tt.args.netmasklen); got != tt.want {
				t.Errorf("Netlen2Mask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNetBytes2Mask(t *testing.T) {
	type args struct {
		mask []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"test 255.255.255.255",
			args{[]byte{255, 255, 255, 255}},
			"255.255.255.255",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NetBytes2Mask(tt.args.mask); got != tt.want {
				t.Errorf("NetBytes2Mask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatMac(t *testing.T) {
	type args struct {
		macStr string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test-format-mac-1",
			args: args{"FFFFFFFFFFFF"},
			want: "ff:ff:ff:ff:ff:ff",
		},
		{
			name: "test-format-mac-2",
			args: args{"FFFFFFFFFF"},
			want: "",
		},
		{
			name: "test-format-mac-3",
			args: args{"FFDDEECCBBAA"},
			want: "ff:dd:ee:cc:bb:aa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatMac(tt.args.macStr); got != tt.want {
				t.Errorf("FormatMac() = %v, want %v", got, tt.want)
			}
		})
	}
}
