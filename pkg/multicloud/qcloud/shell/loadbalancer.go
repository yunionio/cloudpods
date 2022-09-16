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

package shell

import (
	"os"

	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LbListOptions struct {
	}
	shellutils.R(&LbListOptions{}, "lb-list", "List loadbalancers", func(cli *qcloud.SRegion, args *LbListOptions) error {
		lbs, err := cli.GetILoadBalancers()
		if err != nil {
			return err
		}

		printList(lbs, 0, 0, 0, []string{})
		return nil
	})

	type LbCertListOptions struct {
	}
	shellutils.R(&LbCertListOptions{}, "lbcert-list", "List certs", func(cli *qcloud.SRegion, args *LbCertListOptions) error {
		certs, err := cli.GetCertificates("", "", "")
		if err != nil {
			return err
		}

		printList(certs, 0, 0, 0, []string{})
		return nil
	})

	type LbCertIdOptions struct {
		ID string `json:"id" help:"certificate id"`
	}
	shellutils.R(&LbCertIdOptions{}, "lbcert-show", "Show cert", func(cli *qcloud.SRegion, args *LbCertIdOptions) error {
		cert, err := cli.GetCertificate(args.ID)
		if err != nil {
			return err
		}

		printObject(cert)
		return nil
	})

	shellutils.R(&LbCertIdOptions{}, "lbcert-delete", "delete cert", func(cli *qcloud.SRegion, args *LbCertIdOptions) error {
		err := cli.DeleteCertificate(args.ID)
		if err != nil {
			return err
		}

		return nil
	})

	type LbCertUploadOptions struct {
		PublicKeyPath  string `json:"public_key_path"`
		PrivateKeyPath string `json:"private_key_path"`
		CertType       string `json:"cert_type"`
		Desc           string `json:"desc"`
	}

	shellutils.R(&LbCertUploadOptions{}, "lbcert-upload", "Upload cert", func(cli *qcloud.SRegion, args *LbCertUploadOptions) error {
		public := ""
		if len(args.PublicKeyPath) > 0 {
			_public, err := os.ReadFile(args.PublicKeyPath)
			if err != nil {
				return err
			}

			public = string(_public)
		}

		private := ""
		if len(args.PrivateKeyPath) > 0 {
			_private, err := os.ReadFile(args.PrivateKeyPath)
			if err != nil {
				return err
			}

			private = string(_private)
		}
		certId, err := cli.CreateCertificate("", public, private, args.CertType, args.Desc)
		if err != nil {
			return err
		}

		print(certId)
		return nil
	})
}
