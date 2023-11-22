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

package datasource

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/util/wait"

	commontsdb "yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/monitor/registry"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

func init() {
	registry.RegisterService(GetManager())
}

var (
	dsMan *dataSourceManager
)

type DataSourceManager interface {
	registry.Service

	GetDefaultSource(db string) *tsdb.DataSource
}

func GetManager() DataSourceManager {
	if dsMan == nil {
		dsMan = newDataSourceManager()
	}
	return dsMan
}

func GetDefaultSource(db string) (*tsdb.DataSource, error) {
	src := GetManager().GetDefaultSource(db)
	if src == nil {
		return nil, errors.Errorf("default data source is not initialization")
	}
	return src, nil
}

func GetDefaultQueryEndpoint() (tsdb.TsdbQueryEndpoint, error) {
	ds, err := GetDefaultSource("")
	if err != nil {
		return nil, errors.Wrap(err, "get default datasource")
	}
	return tsdb.GetTsdbQueryEndpointFor(ds)
}

func newDataSourceManager() *dataSourceManager {
	return &dataSourceManager{
		defaultDS: nil,
	}
}

type dataSourceManager struct {
	defaultDS *tsdb.DataSource
}

func (m *dataSourceManager) Init() error {
	return m.initDefaultDataSource(context.Background())
}
func (m *dataSourceManager) Run(ctx context.Context) error {
	errgrp, ctx := errgroup.WithContext(ctx)
	errgrp.Go(func() error {
		m.run(ctx)
		return nil
	})
	return errgrp.Wait()
}

func (m *dataSourceManager) GetDefaultSource(db string) *tsdb.DataSource {
	ds := m.defaultDS
	return &tsdb.DataSource{
		Id:                ds.Id,
		Name:              ds.Name,
		Type:              ds.Type,
		Url:               ds.Url,
		User:              ds.User,
		Password:          ds.Password,
		Database:          db,
		BasicAuth:         ds.BasicAuth,
		BasicAuthUser:     ds.BasicAuthUser,
		BasicAuthPassword: ds.BasicAuthPassword,
		TimeInterval:      ds.TimeInterval,
		Updated:           ds.Updated,
	}
}

func (man *dataSourceManager) shouldChangeDataSource(dsType string, url string) bool {
	if man.defaultDS == nil {
		return true
	}
	if man.defaultDS.Type != dsType {
		return true
	}
	if man.defaultDS.Url != url {
		return true
	}
	return false
}

func (man *dataSourceManager) setDataSource(dsType string, url string) {
	if !man.shouldChangeDataSource(dsType, url) {
		return
	}

	log.Infof("set TSDB data source %q: %q", dsType, url)
	man.defaultDS = &tsdb.DataSource{
		Id:      stringutils.UUID4(),
		Name:    dsType,
		Type:    dsType,
		Url:     url,
		Updated: time.Now(),
	}
}

func (man *dataSourceManager) initDefaultDataSource(ctx context.Context) error {
	region := options.Options.Region
	epType := options.Options.SessionEndpointType
	s := auth.GetAdminSession(ctx, region)
	//dsSvc := options.Options.MonitorDataSource
	if s == nil {
		return errors.Errorf("get empty public session for region %s", region)
	}
	source, err := commontsdb.GetDefaultServiceSource(s, epType)
	if err != nil {
		return errors.Wrap(err, "get default TSDB source")
	}
	dsSvc := source.Type
	if err := tsdb.IsValidDataSource(dsSvc); err != nil {
		return errors.Wrapf(err, "invalid type %q", dsSvc)
	}
	url, err := s.GetServiceURL(dsSvc, epType)
	if err != nil {
		return errors.Errorf("get %q public url: %v", dsSvc, err)
	}
	man.setDataSource(dsSvc, url)
	return nil
}

func (m *dataSourceManager) run(ctx context.Context) {
	wait.Forever(func() {
		if err := m.initDefaultDataSource(ctx); err != nil {
			log.Errorf("init default source")
		}
	}, 30*time.Second)
}
