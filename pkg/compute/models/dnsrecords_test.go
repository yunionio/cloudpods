package models

import (
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestDnsRecordsParseInputInfo(t *testing.T) {
	mustJ := func(s string) *jsonutils.JSONDict {
		data, err := jsonutils.ParseString(s)
		if err != nil {
			t.Fatalf("invalid json string: %s\n%s", err, s)
		}
		return data.(*jsonutils.JSONDict)
	}
	cases := []struct {
		name  string
		in    *jsonutils.JSONDict
		out   []string
		isErr bool
	}{
		{
			name: "A/AAAA",
			in: mustJ(`{
				"A.0": "1.2.3.4",
				"A.1": "10.20.30.40",
				"AAAA.0": "::1",
			}`),
			out: []string{"A:1.2.3.4", "A:10.20.30.40", "AAAA:::1"},
		},
		{
			name: "SRV",
			in: mustJ(`{
				"SRV.0": "etcd0.a.com:2379:10:0",
				"SRV.1": "etcd1.a.com:12379:20:1",
				"SRV_host": "etcd3.a.com",
				"SRV_port": 22379,
			}`),
			out: []string{"SRV:etcd0.a.com:2379:10:0", "SRV:etcd1.a.com:12379:20:1", "SRV:etcd3.a.com:22379:100:0"},
		},
		{
			name: "CNAME",
			in: mustJ(`{
				"CNAME": "x.a.com",
			}`),
			out: []string{"CNAME:x.a.com"},
		},
		{
			name: "PTR",
			in: mustJ(`{
				"name": "4.3.2.1.in-addr.arpa",
				"PTR": "a.com",
			}`),
			out: []string{"PTR:a.com"},
		},
		{
			name: "empty",
			in:   mustJ(`{}`),
			out:  []string{},
		},
		// bad
		{
			name: "SRV (bad port)",
			in: mustJ(`{
				"SRV.0": "etcd0.a.com:0",
			}`),
			isErr: true,
		},
		{
			name: "SRV (bad weight)",
			in: mustJ(`{
				"SRV.0": "etcd0.a.com:2379:65536",
			}`),
			isErr: true,
		},
		{
			name: "SRV (bad priority)",
			in: mustJ(`{
				"SRV.0": "etcd0.a.com:2379:10:-1",
			}`),
			isErr: true,
		},
		{
			name: "PTR (reversed)",
			in: mustJ(`{
				"PTR": "4.3.2.1.in-addr.arpa",
				"name": "a.com",
			}`),
			isErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := DnsRecordManager.ParseInputInfo(c.in)
			if err != nil {
				if !c.isErr {
					t.Errorf("unexpected error: %s", err)
				}
				if len(got) > 0 {
					t.Errorf("non empty result on error: %s, %#v", err, got)
				}
			} else {
				if c.isErr {
					t.Errorf("should error, got nil")
				}
				if !reflect.DeepEqual(c.out, got) {
					t.Errorf("want %#v, got %#v", c.out, got)
				}
			}
		})
	}
}
