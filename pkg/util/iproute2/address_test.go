// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package iproute2

import (
	"testing"
)

func TestAddress(t *testing.T) {
	ifname := genDummyName(t)
	dum := addDummy(t, ifname)
	defer delDummy(t, dum)

	emptyT := func(t *testing.T) {
		l := NewAddress(ifname)
		l.Exact()
		if err := l.Err(); err != nil {
			t.Fatalf("got err: %v", err)
		}
		if ipnets, err := l.List4(); err != nil {
			t.Fatalf("list4 err: %v", err)
		} else if len(ipnets) > 0 {
			t.Fatalf("want empty, got %#v", ipnets)
		}
	}

	t.Run("exact some", func(t *testing.T) {
		want := "10.168.222.236/24"
		l := NewAddress(ifname, want, "fe80::222:d5ff:fe9e:28d1/64")
		l.Exact()
		if err := l.Err(); err != nil {
			t.Fatalf("got err: %v", err)
		}
		if ipnets, err := l.List4(); err != nil {
			t.Fatalf("list4 err: %v", err)
		} else if len(ipnets) != 1 {
			t.Fatalf("want 1, got %#v", ipnets)
		} else if got := ipnets[0]; got.String() != want {
			t.Fatalf("want %s, got %s", want, got.String())
		}
		t.Run("empty", emptyT)
		t.Run("empty empty", emptyT)
	})

}

func TestAddress_nopriv(t *testing.T) {
	t.Run("bad address", func(t *testing.T) {
		addresses := []string{
			"192.168.2.1.1",
			"192.168.2.1/33",
			"192.168.2.1",
			"0.0.0.0",
		}
		address := NewAddress("lo", addresses...)
		address.Exact().Add().Del().List4()
		if nerr := len(address.errs); nerr != len(addresses) {
			t.Errorf("want %d err, got %d: %v ", len(addresses), nerr, address.Err())
		}
	})
	t.Run("good address", func(t *testing.T) {
		address := NewAddress("lo",
			"192.168.2.1/0",
			"192.168.2.1/1",
			"0.0.0.0/0",
		)
		address.List4()
		if err := address.Err(); err != nil {
			t.Errorf("got err: %v", err)
		}
	})
}
