/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"net"
	"sync"

	"gitee.com/chunanyong/dm/i18n"
)

// 发版标记
var version = "8.1.3.100"
var build_date = "2023.12.14"
var svn = "20657"

var globalDmDriver = newDmDriver()

func init() {
	sql.Register("dm", globalDmDriver)

	i18n.InitConfig(i18n.Messages_zh_CN)
	i18n.InitConfig(i18n.Messages_en_US)
	i18n.InitConfig(i18n.Messages_zh_HK)
}

func driverInit(svcConfPath string) {
	load(svcConfPath)
	if GlobalProperties != nil && GlobalProperties.Len() > 0 {
		setDriverAttributes(GlobalProperties)
	}
	globalDmDriver.createFilterChain(nil, GlobalProperties)
}

type DmDriver struct {
	filterable
	mu sync.Mutex
	//readPropMutex sync.Mutex
}

func newDmDriver() *DmDriver {
	d := new(DmDriver)
	d.idGenerator = dmDriverIDGenerator
	return d
}

// 支持自定义连接网络地址，返回标准net.Conn对象，相关数据库操作的消息包都将发送到该对象
type DialFunc func(addr string) (net.Conn, error)

// 支持自定义连接网络地址，返回标准net.Conn对象，相关数据库操作的消息包都将发送到该对象
type DialContextFunc func(ctx context.Context, addr string) (net.Conn, error)

var (
	dialsLock sync.RWMutex
	dials     map[string]DialContextFunc
)

// 注册自定义连接方法
func RegisterDial(dialName string, dial DialFunc) {
	RegisterDialContext(dialName, func(_ context.Context, addr string) (net.Conn, error) {
		return dial(addr)
	})
}

// 注册自定义连接方法
func RegisterDialContext(dialName string, dial DialContextFunc) {
	dialsLock.Lock()
	defer dialsLock.Unlock()
	if dials == nil {
		dials = make(map[string]DialContextFunc)
	}
	dials[dialName] = dial
}

/*************************************************************
 ** PUBLIC METHODS AND FUNCTIONS
 *************************************************************/
func (d *DmDriver) Open(dsn string) (driver.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.open(dsn)
}

func (d *DmDriver) OpenConnector(dsn string) (driver.Connector, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.openConnector(dsn)
}

func (d *DmDriver) open(dsn string) (*DmConnection, error) {
	c, err := d.openConnector(dsn)
	if err != nil {
		return nil, err
	}
	return c.connect(context.Background())
}

func (d *DmDriver) openConnector(dsn string) (*DmConnector, error) {
	connector := new(DmConnector).init()
	connector.url = dsn
	connector.dmDriver = d
	//d.readPropMutex.Lock()
	err := connector.mergeConfigs(dsn)
	//d.readPropMutex.Unlock()
	if err != nil {
		return nil, err
	}
	connector.createFilterChain(connector, nil)
	return connector, nil
}
