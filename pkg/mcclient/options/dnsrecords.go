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

package options

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type DNSRecordOptions struct {
	A     []string `help:"DNS A record" metavar:"A_RECORD" positional:"false"`
	AAAA  []string `help:"DNS AAAA record" metavar:"AAAA_RECORD" positional:"false"`
	CNAME string   `help:"DNS CNAME record" metavar:"CNAME_RECORD" positional:"false"`
	PTR   string   `help:"DNS PTR record" metavar:"PTR_RECORD" positional:"false"`

	SRVHost string   `help:"(deprecated) DNS SRV record, server of service" metavar:"SRV_RECORD_HOST" positional:"false"`
	SRVPort int64    `help:"(deprecated) DNS SRV record, port of service" metavar:"SRV_RECORD_PORT" positional:"false"`
	SRV     []string `help:"DNS SRV record, in the format of host:port:weight:priority" metavar:"SRV_RECORD" positional:"false"`
}

func parseDNSRecords(opts *DNSRecordOptions, params *jsonutils.JSONDict) {
	if len(opts.A) > 0 || len(opts.AAAA) > 0 {
		for i, a := range opts.A {
			params.Add(jsonutils.NewString(a), fmt.Sprintf("A.%d", i))
		}
		for i, a := range opts.AAAA {
			params.Add(jsonutils.NewString(a), fmt.Sprintf("AAAA.%d", i))
		}
	} else if len(opts.CNAME) > 0 {
		params.Add(jsonutils.NewString(opts.CNAME), "CNAME")
	} else if len(opts.SRV) > 0 || (len(opts.SRVHost) > 0 && opts.SRVPort > 0) {
		for i, s := range opts.SRV {
			params.Set(fmt.Sprintf("SRV.%d", i), jsonutils.NewString(s))
		}
		// Keep using the original argument passing method in case a
		// newer climc is used against old service
		if len(opts.SRVHost) > 0 && opts.SRVPort > 0 {
			params.Set("SRV_host", jsonutils.NewString(opts.SRVHost))
			params.Set("SRV_port", jsonutils.NewInt(opts.SRVPort))
		}
	} else if len(opts.PTR) > 0 {
		params.Add(jsonutils.NewString(opts.PTR), "PTR")
	}
}

type DNSCreateOptions struct {
	NAME     string `help:"DNS name to create"`
	TTL      int64  `help:"TTL in seconds" positional:"false"`
	Desc     string `help:"Description" json:"description"`
	IsPublic *bool  `help:"Make the newly created record public to all"`

	DNSRecordOptions
}

func (opts *DNSCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := StructToParams(opts)
	if err != nil {
		return nil, err
	}
	parseDNSRecords(&opts.DNSRecordOptions, params)
	return params, nil
}

type DNSUpdateOptions struct {
	ID   string `help:"ID of DNS record to update" json:"-"`
	Name string `help:"Domain name"`
	TTL  int64  `help:"TTL in seconds" positional:"false"`
	Desc string `help:"Description" json:"description"`

	DNSRecordOptions
}

func (opts *DNSUpdateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := StructToParams(opts)
	if err != nil {
		return nil, err
	}
	parseDNSRecords(&opts.DNSRecordOptions, params)
	return params, nil
}

type DNSUpdateRecordsOptions struct {
	ID string `help:"ID of dns record to modify" json:"-"`

	DNSRecordOptions
}

func (opts *DNSUpdateRecordsOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	parseDNSRecords(&opts.DNSRecordOptions, params)
	if params.Size() == 0 {
		return nil, fmt.Errorf("Nothing to add")
	}
	return params, nil
}

type DNSListOptions struct {
	BaseListOptions

	IsPublic string `choices:"0|1"`
}

type DNSGetOptions struct {
	ID string `help:"ID of DNS record to show" json:"-"`
}
