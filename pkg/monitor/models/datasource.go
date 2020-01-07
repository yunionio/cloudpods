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

package models

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/wait"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/monitor/registry"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

var (
	DataSourceManager *SDataSourceManager
)

const (
	DefaultDataSource = "default"
)

const (
	ErrDataSourceDefaultNotFound = errors.Error("Default data source not found")
)

func init() {
	DataSourceManager = &SDataSourceManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SDataSource{},
			"datasources_tbl",
			"datasource",
			"datasources",
		),
	}
	DataSourceManager.SetVirtualObject(DataSourceManager)
	registry.RegisterService(DataSourceManager)
}

type SDataSourceManager struct {
	db.SStandaloneResourceBaseManager
}

func (_ *SDataSourceManager) IsDisabled() bool {
	return false
}

func (_ *SDataSourceManager) Init() error {
	return nil
}

func (man *SDataSourceManager) Run(ctx context.Context) error {
	errgrp, ctx := errgroup.WithContext(ctx)
	errgrp.Go(func() error { return man.initDefaultDataSource(ctx) })
	return errgrp.Wait()
}

func (man *SDataSourceManager) initDefaultDataSource(ctx context.Context) error {
	region := options.Options.Region
	initF := func() {
		ds, err := man.GetDefaultSource()
		if err != nil && err != ErrDataSourceDefaultNotFound {
			log.Errorf("Get default datasource: %v", err)
			return
		}
		if ds != nil {
			return
		}
		s := auth.GetAdminSessionWithPublic(ctx, region, "")
		if s == nil {
			log.Errorf("get empty public session for region %s", region)
			return
		}
		url, err := s.GetServiceURL("influxdb", auth.PublicEndpointType)
		if err != nil {
			log.Errorf("get influxdb public url: %v", err)
			return
		}
		ds = &SDataSource{
			Type: monitor.DataSourceTypeInfluxdb,
			Url:  url,
		}
		ds.Name = DefaultDataSource
		if err := man.TableSpec().Insert(ds); err != nil {
			log.Errorf("insert default influxdb: %v", err)
		}
	}
	wait.Forever(initF, 30*time.Second)
	return nil
}

func (man *SDataSourceManager) GetDefaultSource() (*SDataSource, error) {
	obj, err := man.FetchByName(nil, DefaultDataSource)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDataSourceDefaultNotFound
		} else {
			return nil, err
		}
	}
	return obj.(*SDataSource), nil
}

type SDataSource struct {
	db.SStandaloneResourceBase

	Type      string            `nullable:"false" list:"user"`
	Url       string            `nullable:"false" list:"user"`
	User      string            `width:"64" charset:"utf8" nullable:"true"`
	Password  string            `width:"64" charset:"utf8" nullable:"true"`
	Database  string            `width:"64" charset:"utf8" nullable:"true"`
	IsDefault tristate.TriState `nullable:"false" default:"false" create:"optional"`
	/*
		TimeInterval string
		BasicAuth bool
		BasicAuthUser string
		BasicAuthPassword string
	*/
}

func (m *SDataSourceManager) GetSource(id string) (*SDataSource, error) {
	ret, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return ret.(*SDataSource), nil
}

func (ds *SDataSource) ToTSDBDataSource(db string) *tsdb.DataSource {
	if db == "" {
		db = ds.Database
	}
	return &tsdb.DataSource{
		Id:       ds.GetId(),
		Name:     ds.GetName(),
		Type:     ds.Type,
		Url:      ds.Url,
		User:     ds.User,
		Password: ds.Password,
		Database: db,
		Updated:  ds.UpdatedAt,
		/*BasicAuth: ds.BasicAuth,
		BasicAuthUser: ds.BasicAuthUser,
		BasicAuthPassword: ds.BasicAuthPassword,
		TimeInterval: ds.TimeInterval,*/
	}
}
