package fsdriver

import (
	"reflect"
	"testing"
)

func TestGenSshdConfig(t *testing.T) {
	cases := []struct {
		config        []string
		loginAccount  string
		loginPassword string
		sshPort       int
		expected      []string
	}{
		{
			config: []string{
				"PermitRootLogin no",
				"PasswordAuthentication no",
				"# Port 22",
			},
			loginAccount:  "root",
			loginPassword: "123456",
			sshPort:       22,
			expected: []string{
				"PermitRootLogin yes",
				"PasswordAuthentication yes",
				"# Port 22",
			},
		},
		{
			config: []string{
				"PermitRootLogin no",
				"PasswordAuthentication no",
				"#Port 22",
			},
			loginAccount:  "yunion",
			loginPassword: "123456",
			sshPort:       22,
			expected: []string{
				"PermitRootLogin no",
				"PasswordAuthentication yes",
				"#Port 22",
			},
		},
		{
			config: []string{
				"PermitRootLogin no",
				"PasswordAuthentication no",
				"Port 22",
			},
			loginAccount:  "yunion",
			loginPassword: "123456",
			sshPort:       9000,
			expected: []string{
				"PermitRootLogin no",
				"PasswordAuthentication yes",
				"Port 9000",
			},
		},
		{
			config: []string{
				"# PermitRootLogin no",
				"PasswordAuthentication no",
				"Port 22",
			},
			loginAccount:  "root",
			loginPassword: "123456",
			sshPort:       9000,
			expected: []string{
				"# PermitRootLogin no",
				"PasswordAuthentication yes",
				"Port 9000",
				"PermitRootLogin yes",
			},
		},
		{
			config: []string{
				"# PermitRootLogin no",
				"PasswordAuthentication no",
				"Port 22",
			},
			loginAccount:  "root",
			loginPassword: "123456",
			sshPort:       22,
			expected: []string{
				"# PermitRootLogin no",
				"PasswordAuthentication yes",
				"Port 22",
				"PermitRootLogin yes",
			},
		},
		{
			config: []string{
				"# PermitRootLogin no",
				"PasswordAuthentication no",
				"Port 22",
			},
			loginAccount:  "root",
			loginPassword: "",
			sshPort:       22,
			expected: []string{
				"# PermitRootLogin no",
				"PasswordAuthentication no",
				"Port 22",
				"PermitRootLogin yes",
			},
		},
	}

	for _, c := range cases {
		actual := genSshdConfig(c.config, c.loginAccount, c.loginPassword, c.sshPort)
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("expected %v, got %v", c.expected, actual)
		}
	}
}
