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

package rbdutils

import (
	"fmt"

	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"

	"yunion.io/x/pkg/errors"
)

type SCluster struct {
	conn *rados.Conn
}

func (self *SCluster) withCluster(doFunc func(*rados.Conn) (interface{}, error)) (interface{}, error) {
	defer self.conn.Shutdown()
	return doFunc(self.conn)
}

type SPool struct {
	name    string
	cluster *SCluster
}

func (self *SPool) withIOContext(doFunc func(*rados.IOContext) (interface{}, error)) (interface{}, error) {
	return self.cluster.withCluster(func(conn *rados.Conn) (interface{}, error) {
		ioctx, err := conn.OpenIOContext(self.name)
		if err != nil {
			return nil, errors.Wrapf(err, "OpenIOContext(%s)", self.name)
		}
		return doFunc(ioctx)
	})
}

func (self *SPool) GetCluster() *SCluster {
	return self.cluster
}

func NewCluster(monHost, key string) (*SCluster, error) {
	conn, err := rados.NewConn()
	if err != nil {
		return nil, err
	}
	for k, v := range map[string]string{"mon_host": monHost, "key": key} {
		if len(v) > 0 {
			err = conn.SetConfigOption(k, v)
			if err != nil {
				return nil, errors.Wrapf(err, "SetConfigOption %s %s", k, v)
			}
		}
	}

	for k, v := range map[string]int64{
		"rados_osd_op_timeout": 20 * 60,
		"rados_mon_op_timeout": 5,
		"client_mount_timeout": 2 * 60,
	} {
		err = conn.SetConfigOption(k, fmt.Sprintf("%d", v))
		if err != nil {
			return nil, errors.Wrapf(err, "SetConfigOption %s %d", k, v)
		}
	}
	err = conn.Connect()
	if err != nil {
		return nil, errors.Wrapf(err, "conn.Connect")
	}

	return &SCluster{conn: conn}, nil
}

func (self *SCluster) GetPool(name string) (*SPool, error) {
	return &SPool{name: name, cluster: self}, nil
}

func (self *SCluster) ListPools() ([]string, error) {
	pools, err := self.withCluster(func(conn *rados.Conn) (interface{}, error) {
		return conn.ListPools()
	})
	if err != nil {
		return nil, errors.Wrapf(err, "ListPools")
	}
	return pools.([]string), nil
}

func (self *SCluster) GetClusterStats() (rados.ClusterStat, error) {
	stat, err := self.withCluster(func(conn *rados.Conn) (interface{}, error) {
		return conn.GetClusterStats()
	})
	if err != nil {
		return rados.ClusterStat{}, errors.Wrapf(err, "GetClusterStats")
	}
	return stat.(rados.ClusterStat), nil
}

func (self *SCluster) GetFSID() (string, error) {
	fsid, err := self.withCluster(func(conn *rados.Conn) (interface{}, error) {
		return conn.GetFSID()
	})
	if err != nil {
		return "", errors.Wrapf(err, "GetFSID")
	}
	return fsid.(string), nil
}

func (self *SCluster) DeletePool(pool string) error {
	_, err := self.withCluster(func(conn *rados.Conn) (interface{}, error) {
		return nil, conn.DeletePool(pool)
	})
	return errors.Wrapf(err, "DeletePool")
}

type cmdOutput struct {
	Buffer string
	Info   string
}

func (self *SCluster) MonCommand(args []byte) (cmdOutput, error) {
	result := cmdOutput{}
	_, err := self.withCluster(func(conn *rados.Conn) (interface{}, error) {
		buffer, info, err := conn.MonCommand(args)
		if err != nil {
			return nil, errors.Wrapf(err, "MonCommand")
		}
		result.Buffer = string(buffer)
		result.Info = info
		return nil, nil
	})
	return result, errors.Wrapf(err, "DeletePool")
}

func (self *SPool) ListImages() ([]string, error) {
	images, err := self.withIOContext(func(ioctx *rados.IOContext) (interface{}, error) {
		return rbd.GetImageNames(ioctx)
	})
	if err != nil {
		return nil, errors.Wrapf(err, "GetImageNames")
	}
	return images.([]string), nil
}
