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

package k8s

import "yunion.io/x/jsonutils"

type StorageClassCreateOptions struct {
	ClusterResourceCreateOptions
	// Provisioner string `help:"StorageClass provisioner"`
}

func (o *StorageClassCreateOptions) Params(provisioner string) *jsonutils.JSONDict {
	paramsObj, _ := o.ClusterResourceCreateOptions.Params()
	params := paramsObj.(*jsonutils.JSONDict)
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString(provisioner), "provisioner")
	return params
}

type StorageClassCephCSIRBDTestOptions struct {
	StorageClassCreateOptions
	CLUSTERID       string `help:"Ceph cluster id"`
	SecretName      string `help:"Ceph credentials with required access to the pool"`
	SecretNamespace string `help:"Ceph credentials secret namespace"`
}

func (o *StorageClassCephCSIRBDTestOptions) Params() (jsonutils.JSONObject, error) {
	params := o.StorageClassCreateOptions.Params("rbd.csi.ceph.com")
	input, err := o.getInput()
	if err != nil {
		return nil, err
	}
	params.Add(input, "cephCSIRBD")
	return params, nil
}

func (o *StorageClassCephCSIRBDTestOptions) getInput() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.CLUSTERID), "clusterId")
	params.Add(jsonutils.NewString(o.SecretName), "secretName")
	params.Add(jsonutils.NewString(o.SecretNamespace), "secretNamespace")
	return params, nil
}

type StorageClassCephCSIRBDCreateOptions struct {
	StorageClassCephCSIRBDTestOptions
	POOL          string `help:"Ceph RBD pool"`
	ImageFeatures string `help:"RBD image features" default:"layering"`
	FsType        string `help:"CSI default volume filesystem type" default:"ext4"`
}

func (o *StorageClassCephCSIRBDCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := o.StorageClassCephCSIRBDTestOptions.Params()
	if err != nil {
		return nil, err
	}
	input, err := o.getInput()
	if err != nil {
		return nil, err
	}
	params.(*jsonutils.JSONDict).Add(input, "cephCSIRBD")
	return params, nil
}

func (o *StorageClassCephCSIRBDCreateOptions) getInput() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.POOL), "pool")
	params.Add(jsonutils.NewString(o.CLUSTERID), "clusterId")
	params.Add(jsonutils.NewString(o.FsType), "csiFsType")
	params.Add(jsonutils.NewString(o.ImageFeatures), "imageFeatures")
	params.Add(jsonutils.NewString(o.SecretName), "secretName")
	params.Add(jsonutils.NewString(o.SecretNamespace), "secretNamespace")
	return params, nil
}
